/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"fmt"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
)

const rpcEventChSize = 10240

// ChainRPCService handles all messages of chain client (1 to 1)
// Receive message types: tx request, get bytecode response
// Response message types: get bytecode request, process error
type ChainRPCService struct {
	logger     *zap.SugaredLogger                            // chain rpc service logger
	scheduler  interfaces.RequestScheduler                   // tx request scheduler
	eventCh    chan *protogo.DockerVMMessage                 // invoking handler
	stopSendCh chan struct{}                                 // stop send message goroutine
	stopRecvCh chan struct{}                                 // stop receive message goroutine
	wg         *sync.WaitGroup                               // send / receive goroutine waiting group
	stream     protogo.DockerVMRpc_DockerVMCommunicateServer // rpc stream
}

// NewChainRPCService returns a chain rpc service
//  @param scheduler is tx request scheduler
//  @param manager is process manager
//  @return *ChainRPCService
func NewChainRPCService() *ChainRPCService {
	return &ChainRPCService{
		logger:     logger.NewDockerLogger(logger.MODULE_CHAIN_RPC_SERVICE),
		eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
		stopSendCh: make(chan struct{}),
		stopRecvCh: make(chan struct{}),
		wg:         new(sync.WaitGroup),
		stream:     nil,
	}
}

// SetScheduler sets request scheduler
func (s *ChainRPCService) SetScheduler(scheduler interfaces.RequestScheduler) {
	s.scheduler = scheduler
}

// PutMsg put invoking requests into channel, waiting for chainRPCService to handle request
//  @param msg types include DockerVMType_GET_BYTECODE_REQUEST and DockerVMType_ERROR
//  @return error
func (s *ChainRPCService) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		m, _ := msg.(*protogo.DockerVMMessage)
		s.eventCh <- m
	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// DockerVMCommunicate is docker vm stream for chain
//  @param stream is grpc stream
//  @return error
func (s *ChainRPCService) DockerVMCommunicate(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error {
	s.logger.Infof("DockerVMCommunicate")
	s.stream = stream
	s.wg.Add(2)
	go s.recvMsgRoutine()
	go s.sendMsgRoutine()
	s.wg.Wait()
	s.logger.Infof("s connection end")
	return nil
}

// recvMsgRoutine handles messages received from stream
// message types include: DockerVMType_TX_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (s *ChainRPCService) recvMsgRoutine() {
	s.logger.Debugf("start receive_msg_routine...")
	for {
		select {
		case <-s.stopRecvCh:
			s.logger.Debugf("stop receive_msg_routine...")
			s.wg.Done()
			return
		default:
			msg, err := s.recvMsg()
			if err != nil {
				close(s.stopSendCh)
				s.wg.Done()
				return
			}
			switch msg.Type {
			case protogo.DockerVMType_TX_REQUEST, protogo.DockerVMType_GET_BYTECODE_RESPONSE:
				err = s.scheduler.PutMsg(msg)
				if err != nil {
					s.logger.Errorf("fail to put msg into request scheduler chan: [%s]", err)
				}
			default:
				s.logger.Errorf("unknown msg type, msg: %+v", msg)
			}
		}
	}
}

// sendMsgRoutine send messages (<- eventCh) to chain
// message types include: DockerVMType_GET_BYTECODE_REQUEST and DockerVMType_ERROR
func (s *ChainRPCService) sendMsgRoutine() {
	s.logger.Infof("start send_msg_routine...")
	var err error
	for {
		select {
		case <-s.stopSendCh:
			s.wg.Done()
			s.logger.Debugf("stop send_msg_routine...")
			return

		case msg := <-s.eventCh:
			switch msg.Type {
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				s.logger.Debugf("contract engine -> chain, send get bytecode request, txId: [%s], chan len: [%d]", msg.TxId, len(s.eventCh))
				err = s.sendMsg(msg)

			case protogo.DockerVMType_ERROR:
				s.logger.Debugf("contract engine -> chain, send err msg, txId: [%s], chan len: [%d]", msg.TxId, len(s.eventCh))
				err = s.sendMsg(msg)

			//case protogo.DockerVMType_INIT:
			//	s.logger.Debugf("contract engine -> sandbox, send init msg, txId: [%s], chan len: [%d]", msg.TxId, len(s.eventCh))
			//	err = s.sendMsg(msg)
			//
			//case protogo.DockerVMType_INVOKE:
			//	s.logger.Debugf("contract engine -> sandbox, send invoke msg, txId: [%s], chan len: [%d]", msg.TxId, len(s.eventCh))
			//	err = s.sendMsg(msg)
			//
			//case protogo.DockerVMType_CALL_CONTRACT_RESPONSE:
			//	s.logger.Debugf("contract engine -> sandbox, send call contract response, txId: [%s], chan len: [%d]", msg.TxId, len(s.eventCh))
			//	err = s.sendMsg(msg)

			default:
				s.logger.Errorf("unknown msg type, msg: %+v", msg)
			}
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			s.logger.Errorf("fail to send msg: err: %s, err massage: %s, err code: %s", err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				close(s.stopRecvCh)
				s.wg.Done()
				return
			}
		}
	}
}

// recvMsg receives messages from chainmaker
func (s *ChainRPCService) recvMsg() (*protogo.DockerVMMessage, error) {
	msg, err := s.stream.Recv()
	if err != nil {
		s.logger.Errorf("receive err %s, existed...", err)
		return nil, err
	}
	s.logger.Debugf("receive msg [%s]", msg)
	return msg, nil
}

// sendMsg sends messages to chainmaker
func (s *ChainRPCService) sendMsg(msg *protogo.DockerVMMessage) error {
	s.logger.Debugf("send message [%s]", msg)
	return s.stream.Send(msg)
}
