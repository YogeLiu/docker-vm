/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"google.golang.org/grpc"
)

type ClientMgr interface {
	GetTxSendCh() chan *protogo.CDMMessage

	GetSysCallRespSendCh() chan *protogo.CDMMessage

	GetAndDeleteReceiveChan(txId string) chan *protogo.CDMMessage

	GetReceiveChan(txId string) chan *protogo.CDMMessage

	GetVMConfig() *config.DockerVMConfig

	PutEvent(event *Event)
}

type CDMClient struct {
	id          uint64
	chainId     string
	clientMgr   ClientMgr
	stream      protogo.CDMRpc_CDMCommunicateClient
	logger      *logger.CMLogger
	stopSend    chan struct{}
	stopReceive chan struct{}
}

func NewCDMClient(_id uint64, _chainId string, _logger *logger.CMLogger, _clientMgr ClientMgr) *CDMClient {

	return &CDMClient{
		id:          _id,
		chainId:     _chainId,
		clientMgr:   _clientMgr,
		stream:      nil,
		logger:      _logger,
		stopSend:    make(chan struct{}),
		stopReceive: make(chan struct{}),
	}
}

func (c *CDMClient) StartClient() error {

	c.logger.Infof("start cdm client[%d]", c.id)
	conn, err := c.NewClientConn()
	if err != nil {
		c.logger.Errorf("client[%d] fail to create connection: %s", c.id, err)
		return err
	}

	stream, err := GetCDMClientStream(conn)
	if err != nil {
		c.logger.Errorf("client[%d] fail to get connection stream: %s", c.id, err)
		return err
	}

	c.stream = stream

	go c.sendMsgRoutine()

	go c.receiveMsgRoutine()

	return nil
}

func (c *CDMClient) sendMsgRoutine() {

	c.logger.Infof("client[%d] start sending cdm message", c.id)

	var err error

	for {
		select {
		case txMsg := <-c.clientMgr.GetTxSendCh():
			c.logger.Debugf("client[%d] [%s] send tx req, chan len: [%d]", c.id, txMsg.TxId,
				len(c.clientMgr.GetTxSendCh()))
			err = c.sendCDMMsg(txMsg)
		case stateMsg := <-c.clientMgr.GetSysCallRespSendCh():
			c.logger.Debugf("client[%d] [%s] send syscall resp, chan len: [%d]", c.id, stateMsg.TxId,
				len(c.clientMgr.GetSysCallRespSendCh()))
			err = c.sendCDMMsg(stateMsg)
		case <-c.stopSend:
			c.logger.Debugf("client[%d] close cdm send goroutine", c.id)
			return
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			c.logger.Errorf("client[%d] fail to send msg: err: %s, err massage: %s, err code: %s", c.id, err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				close(c.stopReceive)
				return
			}
		}
	}
}

func (c *CDMClient) receiveMsgRoutine() {

	c.logger.Infof("client[%d] start receiving cdm message", c.id)

	defer func() {
		c.clientMgr.PutEvent(&Event{
			id:        c.id,
			eventType: connectionStopped,
		})
	}()

	var waitCh chan *protogo.CDMMessage

	for {

		select {
		case <-c.stopReceive:
			c.logger.Debugf("client[%d] close cdm client receive goroutine", c.id)
			return
		default:
			receivedMsg, revErr := c.stream.Recv()

			if revErr == io.EOF {
				c.logger.Error("client[%d] receive eof and exit receive goroutine", c.id)
				close(c.stopSend)
				return
			}

			if revErr != nil {
				c.logger.Errorf("client[%d] receive err and exit receive goroutine %s", c.id, revErr)
				close(c.stopSend)
				return
			}

			c.logger.Debugf("client[%d] receive msg from docker manager [%s]", c.id, receivedMsg.TxId)

			switch receivedMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
				waitCh = c.clientMgr.GetAndDeleteReceiveChan(receivedMsg.TxId)
				if waitCh == nil {
					c.logger.Warnf("client[%d] [%s] fail to retrieve response chan, tx response chan is nil",
						c.id, receivedMsg.TxId)
					continue
				}
				waitCh <- receivedMsg
			case protogo.CDMType_CDM_TYPE_GET_STATE, protogo.CDMType_CDM_TYPE_GET_BYTECODE,
				protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR, protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR,
				protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER, protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER,
				protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS:
				waitCh = c.clientMgr.GetReceiveChan(receivedMsg.TxId)
				if waitCh == nil {
					c.logger.Warnf("client[%d] [%s] fail to retrieve response chan, response chan is nil", c.id,
						receivedMsg.TxId)
					continue
				}
				waitCh <- receivedMsg
			default:
				c.logger.Errorf("client[%d] unknown message type, received msg: [%v]", c.id, receivedMsg)
			}
		}
	}
}

func (c *CDMClient) sendCDMMsg(msg *protogo.CDMMessage) error {
	c.logger.Debugf("client[%d] send message: [%s]", c.id, msg)
	return c.stream.Send(msg)
}

// NewClientConn create rpc connection
func (c *CDMClient) NewClientConn() (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(int(utils.GetMaxRecvMsgSizeFromConfig(c.clientMgr.GetVMConfig())*1024*1024)),
			grpc.MaxCallSendMsgSize(int(utils.GetMaxSendMsgSizeFromConfig(c.clientMgr.GetVMConfig())*1024*1024)),
		),
	}

	// just for mac development and pprof testing
	if !c.clientMgr.GetVMConfig().DockerVMUDSOpen {
		ip := "0.0.0.0"
		url := fmt.Sprintf("%s:%s", ip, config.TestPort)
		return grpc.Dial(url, dialOpts...)
	}

	dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
		unixAddress, _ := net.ResolveUnixAddr("unix", sock)
		conn, err := net.DialUnix("unix", nil, unixAddress)
		return conn, err
	}))

	sockAddress := filepath.Join(c.clientMgr.GetVMConfig().DockerVMMountPath, c.chainId, config.SockDir, config.SockName)

	return grpc.DialContext(context.Background(), sockAddress, dialOpts...)

}

// GetCDMClientStream get rpc stream
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}
