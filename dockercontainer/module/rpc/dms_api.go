/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"errors"
	"fmt"
	"io"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module/core"

	"chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"go.uber.org/zap"
)

type ProcessPoolInterface interface {
	RetrieveHandlerFromProcess(processName string) *core.ProcessHandler
}

type DMSApi struct {
	logger      *zap.SugaredLogger
	processPool ProcessPoolInterface
}

func NewDMSApi(processPool ProcessPoolInterface) *DMSApi {
	return &DMSApi{
		logger:      logger.NewDockerLogger(logger.MODULE_CDM_SERVER),
		processPool: processPool,
	}
}

func (s *DMSApi) DMSCommunicate(stream protogo.DMSRpc_DMSCommunicateServer) error {

	// get process from process pool
	registerMsg, err := stream.Recv()
	if err != nil {
		s.logger.Errorf("fail to receive register message")
		return err
	}
	processName := string(registerMsg.Payload)
	handler := s.processPool.RetrieveHandlerFromProcess(processName)
	handler.SetStream(stream)
	s.logger.Debugf("get handler: %s", processName)

	err = handler.HandleMessage(registerMsg)
	if err != nil {
		s.logger.Errorf("fail to handle register msg: [%s] -- msg: [%s]", err, registerMsg)
		return err
	}

	// begin loop to receive msg
	type recvMsg struct {
		msg *protogo.DMSMessage
		err error
	}

	msgAvail := make(chan *recvMsg, 1)
	defer close(msgAvail)

	receiveMessage := func() {
		in, err := stream.Recv()
		msgAvail <- &recvMsg{in, err}
	}

	go receiveMessage()

	for {
		select {
		case rmsg := <-msgAvail:
			switch {
			case rmsg.err == io.EOF:
				s.logger.Debugf("received EOF, ending contract stream")
				return nil
			case rmsg.err != nil:
				err := fmt.Errorf("receive failed: %s", rmsg.err)
				return err
			case rmsg.msg == nil:
				err := errors.New("received nil message, ending contract stream")
				return err
			default:
				err := handler.HandleMessage(rmsg.msg)
				if err != nil {
					err = fmt.Errorf("error handling message: %s", err)
					return err
				}
			}

			go receiveMessage()
		}

	}

}
