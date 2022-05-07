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

// TxType is the type of tx request
type TxType int

const (
	origTx  TxType = iota // original tx, send by chain, depth = 0
	crossTx               // cross contract tx, send by sandbox, invoked by contract, depth > 0
)

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

	requestScheduler *RequestScheduler             // used for return err req to chain
	eventCh          chan *protogo.DockerVMMessage // request group invoking handler

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

		contractManager:  cMgr,
		contractState:    contractEmpty,
		requestScheduler: scheduler,
		eventCh:          make(chan *protogo.DockerVMMessage, requestGroupEventChSize),
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
	go func() {
		for {
			select {
			case msg := <-r.eventCh:
				switch msg.Type {
				case protogo.DockerVMType_TX_REQUEST:
					if err := r.handleTxReq(msg); err != nil {
						r.logger.Errorf("failed to handle tx request, %v", err)
					}

				case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
					r.handleContractReadyResp()

				default:
					r.logger.Errorf("unknown req type")
				}
			}
		}
	}()
}

// PutMsg put invoking requests into chan, waiting for request group to handle request
//  @param req types include DockerVMType_TX_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (r *RequestGroup) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		m, _ := msg.(*protogo.DockerVMMessage)
		r.eventCh <- m
	default:
		return fmt.Errorf("unknown req type")
	}
	return nil
}

// GetContractPath returns contract path
func (r *RequestGroup) GetContractPath() string {

	contractKey := utils.ConstructRequestGroupKey(r.contractName, r.contractVersion)
	return filepath.Join(r.contractManager.GetContractMountDir(), contractKey)
}

// GetTxCh returns tx chan
func (r *RequestGroup) GetTxCh(isCross bool) chan *protogo.DockerVMMessage {

	if isCross {
		return r.crossTxController.txCh
	}
	return r.origTxController.txCh
}

// handleTxReq handle all tx request
func (r *RequestGroup) handleTxReq(req *protogo.DockerVMMessage) error {

	// put tx request into chan at first
	err := r.putTxReqToCh(req)
	if err != nil {
		return fmt.Errorf("failed to handle tx req, %v", err)
	}

	switch r.contractState {
	// try to get contract for first tx.
	case contractEmpty:
		err = r.contractManager.PutMsg(protogo.DockerVMMessage{
			TxId: req.TxId,
			Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
			Request: &protogo.TxRequest{
				ContractName: r.contractName,
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
			_, err = r.getProcesses(origTx)
		} else {
			_, err = r.getProcesses(crossTx)
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
			return fmt.Errorf(msg, err)
		}
		return fmt.Errorf(msg)
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
func (r *RequestGroup) getProcesses(txType TxType) (int, error) {

	var controller *txController
	var reqNumPerProcess int

	// get corresponding controller and request number per process
	switch txType {
	case origTx:
		controller = r.origTxController
		reqNumPerProcess = reqNumPerOrigProcess

	case crossTx:
		controller = r.crossTxController
		reqNumPerProcess = reqNumPerCrossProcess
	default:
		return 0, fmt.Errorf("unknown tx type")
	}

	// calculate how many processes it needs:
	// (currProcessNum + needProcessNum) * reqNumPerProcess = processingReqNum + inQueueReqNum
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
		err = controller.processMgr.PutMsg(messages.GetProcessReqMsg{
			ContractName: r.contractName,
			ContractVersion: r.contractVersion,
			ProcessNum:   needProcessNum,
		})
		// avoid duplicate getting processes
		controller.processWaiting = true
		if err != nil {
			return 0, err
		}
	} else { // do not need any process
		// stop to get processes only if it is waiting
		if !controller.processWaiting {
			return 0, nil
		}
		err = controller.processMgr.PutMsg(messages.GetProcessReqMsg{
			ContractName: r.contractName,
			ContractVersion: r.contractVersion,
			ProcessNum:   0, // 0 for no need
		})
		// avoid duplicate stopping to get processes
		controller.processWaiting = false
		if err != nil {
			return 0, err
		}
	}
	return needProcessNum, nil
}

// handleContractReadyResp set the request group's contract state to contractReady
func (r *RequestGroup) handleContractReadyResp() {

	r.contractState = contractReady
}

// handleProcessReadyResp handles process ready response
func (r *RequestGroup) handleProcessReadyResp(txType TxType) error {

	// restore the state of request group to idle
	switch txType {
	case origTx:
		r.origTxController.processWaiting = false

	case crossTx:
		r.crossTxController.processWaiting = false

	default:
		return fmt.Errorf("unknown tx type")
	}

	// try to get processes from process manager
	if _, err := r.getProcesses(txType); err != nil {
		return fmt.Errorf("failed to handle contract ready resp, %v", err)
	}
	return nil
}
