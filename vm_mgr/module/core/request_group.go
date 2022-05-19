/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"fmt"
	"go.uber.org/zap"
	"math"
	"path/filepath"
	"strconv"
)

const (
	// requestGroupEventChSize is request scheduler event chan size
	requestGroupEventChSize = 15000

	// origTxChSize is orig tx chan size
	origTxChSize = 10000

	// crossTxChSize is cross tx chan size
	crossTxChSize = 10000

	// reqNumPerOrigProcess is the request num for one process can handle
	reqNumPerOrigProcess = 5

	// reqNumPerCrossProcess is the request num for one process can handle
	reqNumPerCrossProcess = 5
)

// contractState is the contract state of request group
type contractState int

const (
	contractEmpty   contractState = iota // contract is not ready
	contractWaiting                      // waiting for contract manager to load contract
	contractReady                        // contract is ready
)

//// TxType is the type of tx request
//type TxType int
//
//const (
//	origTx  TxType = iota // original tx, send by chain, depth = 0
//	crossTx               // cross contract tx, send by sandbox, invoked by contract, depth > 0
//)

// txController handle the tx request chan and process status
type txController struct {
	txCh           chan *protogo.DockerVMMessage
	processWaiting bool
	processMgr     interfaces.ProcessManager
}

// RequestGroup is a batch of txs group by contract name
type RequestGroup struct {
	logger *zap.SugaredLogger // request group logger

	contractName    string // contract name
	contractVersion string // contract version

	contractManager *ContractManager // contract manager, request contract / receive contract ready signal
	contractState   contractState    // handle tx with different contract state

	requestScheduler *RequestScheduler // used for return err req to chain
	eventCh          chan interface{}  // request group invoking handler
	stopCh           chan struct{}     // stop request group

	origTxController  *txController // original tx controller
	crossTxController *txController // cross contract tx controller
}

// NewRequestGroup returns new request group
func NewRequestGroup(
	contractName string,
	contractVersion string,
	oriPMgr interfaces.ProcessManager,
	crossPMgr interfaces.ProcessManager,
	cMgr *ContractManager,
	scheduler *RequestScheduler) *RequestGroup {
	return &RequestGroup{
		logger: logger.NewDockerLogger(logger.MODULE_REQUEST_GROUP),

		contractName:    contractName,
		contractVersion: contractVersion,

		contractManager: cMgr,
		contractState:   contractEmpty,

		requestScheduler: scheduler,
		eventCh:          make(chan interface{}, requestGroupEventChSize),
		stopCh:           make(chan struct{}),

		origTxController: &txController{
			txCh:       make(chan *protogo.DockerVMMessage, origTxChSize),
			processMgr: oriPMgr,
		},
		crossTxController: &txController{
			txCh:       make(chan *protogo.DockerVMMessage, crossTxChSize),
			processMgr: crossPMgr,
		},
	}
}

// Start request manager, listen event chan,
// event chan req types: DockerVMType_TX_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (r *RequestGroup) Start() {

	r.logger.Debugf("start request group routine")

	go func() {
		for {
			select {
			case msg := <-r.eventCh:
				r.logger.Debugf("recv msg from event channel, msg: [%+v]", msg)
				switch msg.(type) {
				case *protogo.DockerVMMessage:
					m := msg.(*protogo.DockerVMMessage)
					switch m.Type {
					case protogo.DockerVMType_TX_REQUEST:
						if err := r.handleTxReq(m); err != nil {
							r.logger.Errorf("failed to handle tx request, %v", err)
						}

					case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
						r.handleContractReadyResp()

					default:
						r.logger.Errorf("unknown msg type, msg: %+v", msg)
					}
				case *messages.GetProcessRespMsg:
					m := msg.(*messages.GetProcessRespMsg)
					if err := r.handleProcessReadyResp(m); err != nil {
						r.logger.Errorf("failed to handle process ready resp, %v", err)
					}
				}
			case <-r.stopCh:
				return
			}
		}
	}()
}

// PutMsg put invoking requests into chan, waiting for request group to handle request
//  @param req types include DockerVMType_TX_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (r *RequestGroup) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage, *messages.GetProcessRespMsg:
		r.eventCh <- msg
		r.logger.Debugf("curr eventCh size: %d", len(r.eventCh))
	case *messages.CloseMsg:
		r.stopCh <- struct{}{}
	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// GetContractPath returns contract path
func (r *RequestGroup) GetContractPath() string {

	contractKey := utils.ConstructRequestGroupKey(r.contractName, r.contractVersion)
	return filepath.Join(r.contractManager.GetContractMountDir(), contractKey)
}

// GetTxCh returns tx chan
func (r *RequestGroup) GetTxCh(isOrig bool) chan *protogo.DockerVMMessage {

	if isOrig {
		return r.origTxController.txCh
	}
	return r.crossTxController.txCh

}

// handleTxReq handle all tx request
func (r *RequestGroup) handleTxReq(req *protogo.DockerVMMessage) error {

	r.logger.Debugf("handle tx request: [%s]", req.TxId)

	// put tx request into chan at first
	err := r.putTxReqToCh(req)
	if err != nil {
		return fmt.Errorf("failed to handle tx req, %v", err)
	}

	switch r.contractState {
	// try to get contract for first tx.
	case contractEmpty:
		err = r.contractManager.PutMsg(&protogo.DockerVMMessage{
			TxId: req.TxId,
			Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
			Request: &protogo.TxRequest{
				ContractName:    r.contractName,
				ContractVersion: r.contractVersion,
			},
		})
		if err != nil {
			return err
		}
		// avoid duplicate getting bytecode
		r.contractState = contractWaiting

	// only enqueue
	case contractWaiting:
		r.logger.Debugf("tx %s enqueue, waiting for contract", req.TxId)

	// see if we should get new processes, if so, try to get
	case contractReady:
		if req.CrossContext.CurrentDepth == 0 || !utils.HasUsed(req.CrossContext.CrossInfo) {
			_, err = r.getProcesses(true)
		} else {
			_, err = r.getProcesses(false)
		}
		if err != nil {
			return fmt.Errorf("failed to get processes, %v", err)
		}
	}

	return nil
}

// putTxReqToCh put tx request into chan
func (r *RequestGroup) putTxReqToCh(req *protogo.DockerVMMessage) error {

	// call contract depth overflow
	if req.CrossContext.CurrentDepth > protocol.CallContractDepth {

		msg := "current depth exceed " + strconv.Itoa(protocol.CallContractDepth)

		// send err req to request scheduler
		err := r.requestScheduler.PutMsg(&protogo.DockerVMMessage{
			TxId: req.TxId,
			Type: protogo.DockerVMType_ERROR,
			Response: &protogo.TxResponse{
				Code:         protogo.DockerVMCode_FAIL,
				Message:      msg,
				ContractName: req.Request.ContractName,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to put msg into request scheduler, %s, %s", msg, err.Error())
		}
		return fmt.Errorf("failed to put msg into request scheduler, %s", msg)
	}

	// original tx, send to original tx chan
	if req.CrossContext.CurrentDepth == 0 || !utils.HasUsed(req.CrossContext.CrossInfo) {
		r.logger.Debugf("put tx request [txId: %s] into orig chan", req.TxId)
		r.origTxController.txCh <- req
		return nil
	}

	// cross contract tx, send to cross contract tx chan
	r.logger.Debugf("put tx request [txId: %s] into cross chan", req.TxId)
	r.crossTxController.txCh <- req
	return nil
}

// getProcesses try to get processes from process manager
func (r *RequestGroup) getProcesses(isOrig bool) (int, error) {

	var controller *txController
	var reqNumPerProcess int

	// get corresponding controller and request number per process
	if isOrig {
		controller = r.origTxController
		reqNumPerProcess = reqNumPerOrigProcess
	} else {
		controller = r.crossTxController
		reqNumPerProcess = reqNumPerCrossProcess
	}

	// calculate how many processes it needs:
	// (currProcessNum + needProcessNum) * reqNumPerProcess = processingReqNum + inQueueReqNum
	// TODO: 有未启动进程则迅速拉起
	currProcessNum := controller.processMgr.GetProcessNumByContractKey(r.contractName, r.contractVersion)
	currChSize := len(controller.txCh)

	needProcessNum := int(math.Ceil(float64(currProcessNum+currChSize)/float64(reqNumPerProcess))) - currProcessNum

	var err error
	// need more processes
	if needProcessNum > 0 {
		// try to get processes only if it is not waiting
		if controller.processWaiting {
			return 0, nil
		}
		r.logger.Debugf("request group %s try to get %d processes, tx chan size: %d",
			utils.ConstructContractKey(r.contractName, r.contractVersion), needProcessNum, len(controller.txCh))
		err = controller.processMgr.PutMsg(&messages.GetProcessReqMsg{
			ContractName:    r.contractName,
			ContractVersion: r.contractVersion,
			ProcessNum:      needProcessNum,
		})
		// avoid duplicate getting processes
		r.updateControllerState(isOrig, true)
		if err != nil {
			return 0, err
		}
	} else { // do not need any process
		// stop to get processes only if it is waiting
		if !controller.processWaiting {
			return 0, nil
		}
		r.logger.Debugf("request group %s stop waiting for processes, tx chan size: %d",
			utils.ConstructContractKey(r.contractName, r.contractVersion), len(controller.txCh))
		err = controller.processMgr.PutMsg(&messages.GetProcessReqMsg{
			ContractName:    r.contractName,
			ContractVersion: r.contractVersion,
			ProcessNum:      0, // 0 for no need
		})
		// avoid duplicate stopping to get processes
		r.updateControllerState(isOrig, false)
		if err != nil {
			return 0, err
		}
	}
	return needProcessNum, nil
}

// handleContractReadyResp set the request group's contract state to contractReady
func (r *RequestGroup) handleContractReadyResp() {

	r.logger.Debugf("handle contract ready resp")

	r.contractState = contractReady
	_, err := r.getProcesses(true)
	if err != nil {
		r.logger.Errorf("failed to get orig processes, %v", err)
	}
	_, err = r.getProcesses(false)
	if err != nil {
		r.logger.Errorf("failed to get cross processes, %v", err)
	}
}

// handleProcessReadyResp handles process ready response
func (r *RequestGroup) handleProcessReadyResp(msg *messages.GetProcessRespMsg) error {

	r.logger.Debugf("handle process ready resp: %+v", msg)

	// restore the state of request group to idle
	if msg.IsOrig {
		r.updateControllerState(true, false)
	} else {
		r.updateControllerState(false, false)
	}

	// try to get processes from process manager
	if _, err := r.getProcesses(msg.IsOrig); err != nil {
		return fmt.Errorf("failed to handle contract ready resp, %v", err)
	}

	return nil
}

// updateControllerState update the controller state
func (r *RequestGroup) updateControllerState(isOrig, toWaiting bool) {

	r.logger.Debugf("update controller state, is original: %v, to waiting: %v", isOrig, toWaiting)

	var controller *txController
	if isOrig {
		controller = r.origTxController
	} else {
		controller = r.crossTxController
	}

	if toWaiting {
		controller.processWaiting = true
	} else {
		controller.processWaiting = false
	}
}
