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
	"sync"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/config"
	"chainmaker.org/chainmaker/vm-engine/v2/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ContractEngineClient .
type ContractEngineClient struct {
	id          uint64
	clientMgr   interfaces.ContractEngineClientMgr
	chainId     string // Distinguish sock files of different chains
	lock        *sync.RWMutex
	stream      protogo.DockerVMRpc_DockerVMCommunicateClient
	logger      protocol.Logger
	stopSend    chan struct{}
	stopReceive chan struct{}
	config      *config.DockerVMConfig
}

// NewContractEngineClient .
func NewContractEngineClient(
	chainId string,
	id uint64,
	logger protocol.Logger,
	cm interfaces.ContractEngineClientMgr,
) *ContractEngineClient {

	return &ContractEngineClient{
		id:          id,
		clientMgr:   cm,
		chainId:     chainId,
		lock:        &sync.RWMutex{},
		stream:      nil,
		logger:      logger,
		stopSend:    make(chan struct{}),
		stopReceive: make(chan struct{}),
		config:      cm.GetVMConfig(),
	}
}

// Start .
func (c *ContractEngineClient) Start() error {

	c.logger.Infof("start contract engine client[%d]", c.id)
	conn, err := c.NewClientConn()
	if err != nil {
		c.logger.Errorf("client[%d] fail to create connection: %s", c.id, err)
		return err
	}

	go func() {
		select {
		case <-c.stopReceive:
			if err = conn.Close(); err != nil {
				c.logger.Warnf("failed to close connection")
			}
		case <-c.stopSend:
			if err = conn.Close(); err != nil {
				c.logger.Warnf("failed to close connection")
			}
		}
	}()

	stream, err := GetClientStream(conn)
	if err != nil {
		c.logger.Errorf("fail to get connection stream: %s", err)
		return err
	}

	c.stream = stream

	go c.sendMsgRoutine()

	go c.receiveMsgRoutine()

	return nil
}

// Stop .
func (c *ContractEngineClient) Stop() {
	err := c.stream.CloseSend()
	if err != nil {
		c.logger.Errorf("close stream failed: ", err)
	}
}

func (c *ContractEngineClient) sendMsgRoutine() {

	c.logger.Infof("start sending contract engine message ")

	var err error

	for {
		select {
		case txReq := <-c.clientMgr.GetTxSendCh():
			c.logger.Debugf("[%s] send tx req, chan len: [%d]", txReq.TxId, c.clientMgr.GetTxSendChLen())
			err = c.sendMsg(txReq)
		case getByteCodeResp := <-c.clientMgr.GetByteCodeRespSendCh():
			c.logger.Debugf(
				"[%s] send GetByteCode resp, chan len: [%d]",
				getByteCodeResp.TxId,
				c.clientMgr.GetByteCodeRespChLen(),
			)
			err = c.sendMsg(getByteCodeResp)
		case <-c.stopSend:
			c.logger.Debugf("close contract engine send goroutine")
			return
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			c.logger.Errorf("fail to send msg: err: %s, err massage: %s, err code: %s", err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				close(c.stopReceive)
				return
			}
		}
	}
}

func (c *ContractEngineClient) receiveMsgRoutine() {

	c.logger.Infof("start receiving contract engine message ")
	defer func() {
		c.clientMgr.PutEvent(&interfaces.Event{
			Id:        c.id,
			EventType: interfaces.EventType_ConnectionStopped,
		})
	}()

	for {
		select {
		case <-c.stopReceive:
			c.logger.Debugf("close contract engine client receive goroutine")
			return
		default:
			receivedMsg, revErr := c.stream.Recv()

			if revErr == io.EOF {
				c.logger.Error("client receive eof and exit receive goroutine")
				close(c.stopSend)
				return
			}

			if revErr != nil {
				c.logger.Errorf("client receive err and exit receive goroutine %s", revErr)
				close(c.stopSend)
				return
			}

			c.logger.Debugf("[%s] receive msg from docker manager, msg type [%s]", receivedMsg.TxId, receivedMsg.Type)

			switch receivedMsg.Type {
			case protogo.DockerVMType_TX_RESPONSE:
				notify := c.clientMgr.GetReceiveNotify(c.chainId, receivedMsg.TxId)
				if notify == nil {
					c.logger.Warnf("[%s] fail to retrieve notify, tx notify is nil",
						receivedMsg.TxId)
					continue
				}
				notify(receivedMsg)
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				notify := c.clientMgr.GetReceiveNotify(c.chainId, receivedMsg.TxId)
				if notify == nil {
					c.logger.Warnf("[%s] fail to retrieve notify, tx notify is nil", receivedMsg.TxId)
					continue
				}
				notify(receivedMsg)

			case protogo.DockerVMType_ERROR:
				notify := c.clientMgr.GetReceiveNotify(c.chainId, receivedMsg.TxId)
				if notify == nil {
					c.logger.Warnf("[%s] fail to retrieve notify, tx notify is nil", receivedMsg.TxId)
					continue
				}
				notify(receivedMsg)

			default:
				c.logger.Errorf("unknown message type, received msg: [%v]", receivedMsg)
			}
		}
	}
}

func (c *ContractEngineClient) sendMsg(msg *protogo.DockerVMMessage) error {
	c.logger.Debugf("send message[%s], type: [%s]", msg.TxId, msg.Type)
	//c.logger.Debugf("msg [%+v]", msg)
	return c.stream.Send(msg)
}

// NewClientConn create rpc connection
func (c *ContractEngineClient) NewClientConn() (*grpc.ClientConn, error) {

	// just for mac development and pprof testing
	if c.config.ConnectionProtocol == config.TCPProtocol {
		url := fmt.Sprintf("%s:%d", c.config.ContractEngine.Host, c.config.ContractEngine.Port)
		dialOpts := []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(int(c.config.MaxRecvMsgSize)*1024*1024),
				grpc.MaxCallSendMsgSize(int(c.config.MaxSendMsgSize)*1024*1024),
			),
		}
		return grpc.Dial(url, dialOpts...)
	}

	udsDialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(int(utils.GetMaxRecvMsgSizeFromConfig(c.config)*1024*1024)),
			grpc.MaxCallSendMsgSize(int(utils.GetMaxSendMsgSizeFromConfig(c.config)*1024*1024)),
		),
	}

	udsDialOpts = append(udsDialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
		unixAddress, _ := net.ResolveUnixAddr("unix", sock)
		conn, err := net.DialUnix("unix", nil, unixAddress)
		return conn, err
	}))

	sockAddress := filepath.Join(c.config.DockerVMMountPath, config.SockDir, config.EngineSockName)

	return grpc.DialContext(context.Background(), sockAddress, udsDialOpts...)

}

// GetClientStream get rpc stream
func GetClientStream(conn *grpc.ClientConn) (protogo.DockerVMRpc_DockerVMCommunicateClient, error) {
	return protogo.NewDockerVMRpcClient(conn).DockerVMCommunicate(context.Background())
}
