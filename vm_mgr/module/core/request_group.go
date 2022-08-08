/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
	"go.uber.org/zap"
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
	_contractEmpty   contractState = iota // contract is not ready
	_contractWaiting                      // waiting for contract manager to load contract
	_contractReady                        // contract is ready
)

// _getByteCodeTimeout is the timeout(s) of get byte code, request group will clean all txs and exit then.
const _getByteCodeTimeout = 6

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

	requestScheduler interfaces.RequestScheduler   // used for return err req to chain
	eventCh          chan interface{}              // request group invoking handler
	txCh             chan *protogo.DockerVMMessage // tx event chan
	bufCh            chan *protogo.DockerVMMessage // before ready: txCh -> bufCh bufCh, after ready: bufCh -> orig/cross txCh
	stopCh           chan struct{}                 // stop request group

	getBytecodeTimer *time.Timer

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
		eventCh:          make(chan interface{}, _requestGroupEventChSize),
		txCh:             make(chan *protogo.DockerVMMessage, _requestGroupTxChSize),
		bufCh:            make(chan *protogo.DockerVMMessage, _requestGroupTxChSize),
		stopCh:           make(chan struct{}),

		getBytecodeTimer: time.NewTimer(math.MaxInt32 * time.Second), //initial tx timer, never triggered

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
				switch msg.(type) {
				case *messages.GetProcessRespMsg:
					if err := r.handleProcessReadyResp(msg.(*messages.GetProcessRespMsg)); err != nil {
						r.logger.Errorf("failed to handle process ready resp, %v", err)
					}
				case *messages.BadContractResp:
					if err := r.handleBadContractResp(msg.(*messages.BadContractResp)); err != nil {
						r.logger.Errorf("failed to handle retry to get bytecode, %v", err)
					}
				}

			case msg := <-r.txCh:
				switch msg.Type {
				case protogo.DockerVMType_TX_REQUEST:
					if err := r.handleTxReq(msg); err != nil {
						r.logger.Errorf("failed to handle tx request, %v", err)
					}

				case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
					if err := r.handleContractReadyResp(msg); err != nil {
						r.logger.Errorf("failed to handle contract ready resp, %v", err)
					}

				default:
					r.logger.Errorf("unknown msg type, msg: %+v", msg)
				}

			case <-r.stopCh:
				if err := r.handleStopRequestGroup(); err != nil {
					r.logger.Errorf("failed to handle stop request group, %v", err)
				}
				return

			case <-r.getBytecodeTimer.C:
				if err := r.handleGetBytecodeTimeout(); err != nil {
					r.logger.Errorf("failed to handle get bytecode timeout, %v", err)
				}
			}
		}
	}()
}

// PutMsg put invoking requests into chan, waiting for request group to handle request
//  @param req types include DockerVMType_TX_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (r *RequestGroup) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *messages.GetProcessRespMsg, *messages.BadContractResp:
		r.eventCh <- msg
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

	switch r.contractState {
	// try to get contract for first tx.
	case _contractEmpty:
		r.bufCh <- req
		if err := r.sendGetContractReq(req); err != nil {
			return fmt.Errorf("failed to send get contract req")
		}

	// only enqueue
	case _contractWaiting:
		r.bufCh <- req
		r.logger.Debugf("tx %s enqueue, waiting for contract", req.TxId)

	// see if we should get new processes, if so, try to get
	case _contractReady:
		// put tx request into chan at first
		err := r.putTxReqToCh(req)
		if err != nil {
			return fmt.Errorf("failed to handle tx request, %v", err)
		}
		if _, err = r.getProcesses(utils.IsOrig(req)); err != nil {
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
			ChainId: r.chainID,
			TxId:    req.TxId,
			Type:    protogo.DockerVMType_ERROR,
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
	if utils.IsOrig(req) {
		r.origTxController.txCh <- req
		r.logger.Debugf("put tx request [%s] into orig chan, curr ch size [%d]", req.TxId, len(r.origTxController.txCh))
		return nil
	}

	// cross contract tx, send to cross contract tx chan
	r.crossTxController.txCh <- req
	r.logger.Debugf("put tx request [%s] into cross chan, curr ch size [%d]", req.TxId, len(r.crossTxController.txCh))
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
	if !isOrig {
		needProcessNum = len(controller.txCh)
	}
	r.logger.Debugf("tx chan size: [%d], process num: [%d], need process num (isOrig: %v): [%d]",
		len(controller.txCh), currProcessNum, isOrig, needProcessNum)
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

// sendGetContractReq send get contract req
func (r *RequestGroup) sendGetContractReq(req *protogo.DockerVMMessage) error {

	// info contract manager to get bytecode
	if err := r.requestScheduler.GetContractManager().PutMsg(&protogo.DockerVMMessage{
		ChainId: r.chainID,
		TxId:    req.TxId,
		Type:    protogo.DockerVMType_GET_BYTECODE_REQUEST,
		Request: &protogo.TxRequest{
			ChainId:         r.chainID,
			ContractName:    r.contractName,
			ContractVersion: r.contractVersion,
		},
	}); err != nil {
		return fmt.Errorf("failed to put get bytecode req into contract manager chan, %v", err)
	}

	// reset get bytecode timer
	r.getBytecodeTimer.Reset(_getByteCodeTimeout * time.Second)

	// avoid duplicate getting bytecode
	r.contractState = _contractWaiting

	return nil
}

// handleBadContractResp retry to get bytecode
func (r *RequestGroup) handleBadContractResp(msg *messages.BadContractResp) error {

	r.logger.Debugf("handle retry get bytecode")

	// reset contract state to empty
	r.contractState = _contractEmpty

	// retry next tx when chan size > 0
	if len(r.origTxController.txCh) > 0 || len(r.crossTxController.txCh) > 0 {
		r.logger.Debugf("retry to get bytecode")
		if err := r.sendGetContractReq(msg.Tx); err != nil {
			return fmt.Errorf("failed to send get contract req, %v", err)
		}
	}

	return nil
}

// handleContractReadyResp set the request group's contract state to _contractReady
func (r *RequestGroup) handleContractReadyResp(msg *protogo.DockerVMMessage) error {

	r.logger.Debugf("handle contract ready resp")

	// pop tx timer, use Stop() to ensure old timer stopped to avoid timer reached after length judgement
	if !r.getBytecodeTimer.Stop() && len(r.getBytecodeTimer.C) > 0 {
		<-r.getBytecodeTimer.C
	}

	if msg.Response.Code == protogo.DockerVMCode_FAIL {
		if err := r.handleGetBytecodeFailed(); err != nil {
			return fmt.Errorf("failed to handle get bytecode fail, %v", err)
		}
		return fmt.Errorf("get bytecode response failed")
	}

	r.contractState = _contractReady

	// put all tx from group txCh to process txCh
moveTxs:
	for {
		select {
		case tx := <-r.bufCh:
			if utils.IsOrig(tx) {
				// original tx, send to original tx chan
				r.origTxController.txCh <- tx
				r.logger.Debugf("put tx request [%s] into orig chan, curr ch size [%d]", tx.TxId, len(r.origTxController.txCh))
			} else {
				// cross contract tx, send to cross contract tx chan
				r.crossTxController.txCh <- tx
				r.logger.Debugf("put tx request [%s] into cross chan, curr ch size [%d]", tx.TxId, len(r.crossTxController.txCh))
			}
		default:
			break moveTxs
		}
	}

	// try to get original process to handle original txs
	if _, err := r.getProcesses(true); err != nil {
		r.logger.Errorf("failed to get orig processes, %v", err)
	}

	// try to get cross process to handle cross txs
	if _, err := r.getProcesses(false); err != nil {
		r.logger.Errorf("failed to get cross processes, %v", err)
	}

	return nil
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

// handleGetBytecodeTimeout handles get bytecode timeout
func (r *RequestGroup) handleGetBytecodeTimeout() error {

	r.logger.Errorf("handle get bytecode timeout")

	if err := r.handleGetBytecodeErr(); err != nil {
		return fmt.Errorf("failed to handle get bytecode err, %v", err)
	}

	return nil
}

// handleGetBytecodeFailed handles get bytecode failed
func (r *RequestGroup) handleGetBytecodeFailed() error {

	r.logger.Debugf("handle get bytecode failed")

	if err := r.handleGetBytecodeErr(); err != nil {
		return fmt.Errorf("failed to handle get bytecode err, %v", err)
	}

	return nil
}

// handleGetBytecodeErr handles error get bytecode resposne
func (r *RequestGroup) handleGetBytecodeErr() error {

	r.logger.Debugf("handle get bytecode error, pop first tx")

	// reset contract state to empty
	r.contractState = _contractEmpty

	// pop first tx
	tx := <-r.bufCh

	// return tx error response
	if err := r.returnTxErrorResp(tx.TxId, "get bytecode error"); err != nil {
		r.logger.Errorf("failed to return tx error response, %v", err)
	}

	// retry next tx for chan size > 0
	if len(r.bufCh) > 0 {
		r.logger.Debugf("retry to get bytecode for tx")
		if err := r.sendGetContractReq(tx); err != nil {
			return fmt.Errorf("failed to send get contract req, %v", err)
		}
	}

	return nil
}

// handleStopRequestGroup handles stop request group
func (r *RequestGroup) handleStopRequestGroup() error {

	r.logger.Debugf("handle exit request group")

popTx:
	for {
		select {
		case msg := <-r.txCh:
			switch msg.Type {
			case protogo.DockerVMType_TX_REQUEST:
				if err := r.returnTxErrorResp(msg.TxId, "tx error because request group exited"); err != nil {
					r.logger.Errorf("failed to return tx error response, %v", err)
				}
			}
		default:
			break popTx
		}
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

// returnTxErrorResp return error to request scheduler
func (r *RequestGroup) returnTxErrorResp(txId string, errMsg string) error {
	errResp := &protogo.DockerVMMessage{
		ChainId: r.chainID,
		Type:    protogo.DockerVMType_ERROR,
		TxId:    txId,
		Response: &protogo.TxResponse{
			Code:    protogo.DockerVMCode_FAIL,
			Result:  nil,
			Message: errMsg,
		},
	}
	r.logger.Errorf("return error result of tx [%s]", txId)
	if err := r.requestScheduler.PutMsg(errResp); err != nil {
		return fmt.Errorf("failed to invoke request scheduler PutMsg, %v", err)
	}
	return nil
}
