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
	"path/filepath"
	"strconv"
)

const (

	// _requestGroupTxChSize is request scheduler event chan size
	_requestGroupTxChSize = 30000

	// _requestGroupEventChSize is request scheduler event chan size
	_requestGroupEventChSize = 100

	// _origTxChSize is orig tx chan size
	_origTxChSize = 30000

	// _crossTxChSize is cross tx chan size
	_crossTxChSize = 10000

	// reqNumPerOrigProcess is the request num for one process can handle
	reqNumPerOrigProcess = 1

	// reqNumPerCrossProcess is the request num for one process can handle
	reqNumPerCrossProcess = 1
)

// contractState is the contract state of request group
type contractState int

const (
	_contractEmpty   contractState = iota // contract is not _ready
	_contractWaiting                      // waiting for contract manager to load contract
	_contractReady                        // contract is _ready
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

	chainID         string
	contractName    string // contract name
	contractVersion string // contract version

	contractState contractState // handle tx with different contract state

	requestScheduler interfaces.RequestScheduler      // used for return err req to chain
	eventCh          chan *messages.GetProcessRespMsg // request group invoking handler
	txCh             chan *protogo.DockerVMMessage
	stopCh           chan struct{} // stop request group

	origTxController  *txController // original tx controller
	crossTxController *txController // cross contract tx controller
}

// check interface implement
var _ interfaces.RequestGroup = (*RequestGroup)(nil)

// NewRequestGroup returns new request group
func NewRequestGroup(chainID, contractName, contractVersion string, oriPMgr, crossPMgr interfaces.ProcessManager,
	scheduler interfaces.RequestScheduler) *RequestGroup {
	return &RequestGroup{

		logger: logger.NewDockerLogger(logger.GenerateRequestGroupLoggerName(
			utils.ConstructContractKey(chainID, contractName, contractVersion))),

		chainID:         chainID,
		contractName:    contractName,
		contractVersion: contractVersion,

		contractState: _contractEmpty,

		requestScheduler: scheduler,
		eventCh:          make(chan *messages.GetProcessRespMsg, _requestGroupEventChSize),
		txCh:             make(chan *protogo.DockerVMMessage, _requestGroupTxChSize),
		stopCh:           make(chan struct{}),

		origTxController: &txController{
			txCh:       make(chan *protogo.DockerVMMessage, _origTxChSize),
			processMgr: oriPMgr,
		},
		crossTxController: &txController{
			txCh:       make(chan *protogo.DockerVMMessage, _crossTxChSize),
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
				if err := r.handleProcessReadyResp(msg); err != nil {
					r.logger.Errorf("failed to handle process _ready resp, %v", err)
				}

			case msg := <-r.txCh:
				switch msg.Type {
				case protogo.DockerVMType_TX_REQUEST:
					if err := r.handleTxReq(msg); err != nil {
						r.logger.Errorf("failed to handle tx request, %v", err)
					}

				case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
					r.handleContractReadyResp()

				default:
					r.logger.Errorf("unknown msg type, msg: %+v", msg)
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
	case *messages.GetProcessRespMsg:
		r.eventCh <- msg.(*messages.GetProcessRespMsg)
	case *protogo.DockerVMMessage:
		r.txCh <- msg.(*protogo.DockerVMMessage)
	case *messages.CloseMsg:
		r.stopCh <- struct{}{}
	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// GetContractPath returns contract path
func (r *RequestGroup) GetContractPath() string {

	contractKey := utils.ConstructContractKey(r.chainID, r.contractName, r.contractVersion)
	return filepath.Join(r.requestScheduler.GetContractManager().GetContractMountDir(), contractKey)
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
	case _contractEmpty:
		err = r.requestScheduler.GetContractManager().PutMsg(&protogo.DockerVMMessage{
			TxId: req.TxId,
			Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
			Request: &protogo.TxRequest{
				ChainId:         r.chainID,
				ContractName:    r.contractName,
				ContractVersion: r.contractVersion,
			},
		})
		if err != nil {
			return err
		}
		// avoid duplicate getting bytecode
		r.contractState = _contractWaiting

	// only enqueue
	case _contractWaiting:
		r.logger.Debugf("tx %s enqueue, waiting for contract", req.TxId)

	// see if we should get new processes, if so, try to get
	case _contractReady:
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

	if req.CrossContext == nil {
		return fmt.Errorf("nil cross context")
	}
	// call contract depth overflow
	if req.CrossContext.CurrentDepth > protocol.CallContractDepth {

		msg := "current depth exceed " + strconv.Itoa(protocol.CallContractDepth)

		// send err req to request scheduler
		err := r.requestScheduler.PutMsg(&protogo.DockerVMMessage{
			TxId: req.TxId,
			Type: protogo.DockerVMType_ERROR,
			Response: &protogo.TxResponse{
				Code:            protogo.DockerVMCode_FAIL,
				Message:         msg,
				ChainId:         r.chainID,
				ContractName:    req.Request.ContractName,
				ContractVersion: req.Request.ContractVersion,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to put msg into request scheduler, %s, %s", msg, err.Error())
		}
		return fmt.Errorf("failed to put msg into request scheduler, %s", msg)
	}

	// original tx, send to original tx chan
	if req.CrossContext.CurrentDepth == 0 || !utils.HasUsed(req.CrossContext.CrossInfo) {
		r.logger.Debugf("put tx request [%s] into orig chan, curr ch size [%d]", req.TxId, len(r.origTxController.txCh))
		r.origTxController.txCh <- req
		return nil
	}

	// cross contract tx, send to cross contract tx chan
	r.logger.Debugf("put tx request [%s] into cross chan, curr ch size [%d]", req.TxId, len(r.crossTxController.txCh))
	r.crossTxController.txCh <- req
	return nil
}

// getProcesses try to get processes from process manager
func (r *RequestGroup) getProcesses(isOrig bool) (int, error) {

	var controller *txController
	//var reqNumPerProcess int

	// get corresponding controller and request number per process
	if isOrig {
		controller = r.origTxController
		//reqNumPerProcess = reqNumPerOrigProcess
	} else {
		controller = r.crossTxController
		//reqNumPerProcess = reqNumPerCrossProcess
	}

	// calculate how many processes it needs:
	// (currProcessNum + needProcessNum) * reqNumPerProcess = processingReqNum + inQueueReqNum
	//currProcessNum := controller.processMgr.GetProcessNumByContractKey(r.contractName, r.contractVersion)
	//currChSize := len(controller.txCh)
	//
	//needProcessNum := int(math.Ceil(float64(currProcessNum+currChSize)/float64(reqNumPerProcess))) - currProcessNum

	currProcessNum := controller.processMgr.GetProcessNumByContractKey(r.chainID, r.contractName, r.contractVersion)
	needProcessNum := len(controller.txCh) - currProcessNum
	r.logger.Debugf("tx chan size: [%d], process num: [%d], need process num: [%d]",
		len(controller.txCh), currProcessNum, needProcessNum)
	var err error
	// need more processes
	if needProcessNum > 0 {
		// try to get processes only if it is not waiting
		if controller.processWaiting {
			return 0, nil
		}
		r.logger.Debugf("try to get %d process(es)", needProcessNum)
		err = controller.processMgr.PutMsg(&messages.GetProcessReqMsg{
			ChainID:         r.chainID,
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
		r.logger.Debugf("stop waiting for processes")
		err = controller.processMgr.PutMsg(&messages.GetProcessReqMsg{
			ChainID:         r.chainID,
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

// handleContractReadyResp set the request group's contract state to _contractReady
func (r *RequestGroup) handleContractReadyResp() {

	r.logger.Debugf("handle contract _ready resp")

	r.contractState = _contractReady
	_, err := r.getProcesses(true)
	if err != nil {
		r.logger.Errorf("failed to get orig processes, %v", err)
	}
	_, err = r.getProcesses(false)
	if err != nil {
		r.logger.Errorf("failed to get cross processes, %v", err)
	}
}

// handleProcessReadyResp handles process _ready response
func (r *RequestGroup) handleProcessReadyResp(msg *messages.GetProcessRespMsg) error {

	r.logger.Debugf("handle process _ready resp: %+v", msg)

	// restore the state of request group to _idle
	if msg.IsOrig {
		r.updateControllerState(true, false)
	} else {
		r.updateControllerState(false, false)
	}

	// try to get processes from process manager
	if _, err := r.getProcesses(msg.IsOrig); err != nil {
		return fmt.Errorf("failed to handle contract _ready resp, %v", err)
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
