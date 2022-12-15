/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"errors"
	"fmt"
	"io"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/config"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/module/core"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/pb_sdk/protogo"
	"go.uber.org/zap"
)

type ProcessManager interface {
	GetProcess(processName string) *core.Process
}

type DMSApi struct {
	logger         *zap.SugaredLogger
	processManager ProcessManager
}

func NewDMSApi(processManager ProcessManager) *DMSApi {
	return &DMSApi{
		logger:         logger.NewDockerLogger(logger.MODULE_DMS_SERVER, config.DockerLogDir),
		processManager: processManager,
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

	// check cross contract or original contract
	process := s.processManager.GetProcess(processName)
	handler := process.Handler
	handler.SetStream(stream)
	s.logger.Debugf("get handler: %s", processName)

	err = handler.HandleMessage(registerMsg)
	if err != nil {
		s.logger.Errorf("fail to handle register msg: [%s] -- msg: [%s], process [%s]", err, registerMsg, process.ProcessName())
		return err
	}

	// begin loop to receive msg
	type recvMsg struct {
		msg *protogo.DMSMessage
		err error
	}

	msgAvail := make(chan *recvMsg, 1)

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
				s.logger.Warnf("received EOF, ending contract stream [%s]", process.ProcessName())
				return nil
			case rmsg.err != nil:
				s.logger.Warnf("[%s] received fail %s", process.ProcessName(), rmsg.err)
				err := fmt.Errorf("receive failed: %s", rmsg.err)
				return err
			case rmsg.msg == nil:
				s.logger.Warnf("[%s] received nil message, ending contract stream", process.ProcessName())
				err := errors.New("received nil message, ending contract stream")
				return err
			default:
				s.logger.Debugf("[%s] handle msg [%v]", process.ProcessName(), rmsg)
				err := handler.HandleMessage(rmsg.msg)
				if err != nil {
					s.logger.Errorf("[%s] err handling message: %s", process.ProcessName(), err)
					err = fmt.Errorf("error handling message: %s process [%s]", err, process.ProcessName())
					return err
				}
			}

			go receiveMessage()
		}

	}

}
