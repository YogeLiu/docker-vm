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

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"google.golang.org/grpc"
)

const (
	txSize = 15000
)

type CDMClient struct {
	chainId             string
	txSendCh            chan *protogo.CDMMessage // used to send tx to docker-go instance
	stateResponseSendCh chan *protogo.CDMMessage // used to receive message from docker-go
	lock                sync.RWMutex
	// key: txId, value: chan, used to receive tx response from docker-go
	recvChMap map[string]chan *protogo.CDMMessage
	stream    protogo.CDMRpc_CDMCommunicateClient
	logger    *logger.CMLogger
	stop      chan struct{}
	config    *config.DockerVMConfig
}

func NewCDMClient(chainId string, vmConfig *config.DockerVMConfig) *CDMClient {

	return &CDMClient{
		chainId:             chainId,
		txSendCh:            make(chan *protogo.CDMMessage, txSize),   // tx request
		stateResponseSendCh: make(chan *protogo.CDMMessage, txSize*8), // get_state response and bytecode response
		recvChMap:           make(map[string]chan *protogo.CDMMessage),
		stream:              nil,
		logger:              logger.GetLoggerByChain(logger.MODULE_VM, chainId),
		stop:                nil,
		config:              vmConfig,
	}
}

func (c *CDMClient) GetTxSendCh() chan *protogo.CDMMessage {
	return c.txSendCh
}

func (c *CDMClient) GetStateResponseSendCh() chan *protogo.CDMMessage {
	return c.stateResponseSendCh
}

func (c *CDMClient) RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.logger.Debugf("register receive chan for [%s]", txId)
	c.recvChMap[txId] = recvCh
}

func (c *CDMClient) GetCMConfig() *config.DockerVMConfig {
	return c.config
}

func (c *CDMClient) deleteRecvChan(txId string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.logger.Debugf("delete receive chan for [%s]", txId)
	delete(c.recvChMap, txId)
}

func (c *CDMClient) getRecvChan(txId string) chan *protogo.CDMMessage {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.recvChMap[txId]
}

func (c *CDMClient) StartClient() bool {

	c.logger.Debugf("start cdm rpc..")
	conn, err := c.NewClientConn()
	if err != nil {
		c.logger.Errorf("fail to create connection: %s", err)
		return false
	}

	stream, err := GetCDMClientStream(conn)
	if err != nil {
		c.logger.Errorf("fail to get connection stream: %s", err)
		return false
	}

	c.stream = stream
	c.stop = make(chan struct{})

	go c.sendMsgRoutine()

	go c.recvMsgRoutine()

	return true
}

func (c *CDMClient) closeConnection() {
	// close two goroutine
	close(c.stop)
	// close stream
	err := c.stream.CloseSend()
	if err != nil {
		return
	}
}

func (c *CDMClient) sendMsgRoutine() {

	c.logger.Debugf("start sending cdm message ")
	// listen three chan:
	// txSendCh: used to send tx to docker manager
	// stateResponseSendCh: used to send get state response or bytecode response to docker manager
	// stopCh

	var err error

	for {
		select {
		case txMsg := <-c.txSendCh:
			c.logger.Debugf("[%s] send tx request to docker manager", txMsg.TxId)
			err = c.sendCDMMsg(txMsg)
		case stateMsg := <-c.stateResponseSendCh:
			c.logger.Debugf("[%s] send request to docker manager", stateMsg.TxId)
			err = c.sendCDMMsg(stateMsg)
		case <-c.stop:
			c.logger.Debugf("close send cdm msg")
			return
		}

		if err != nil {
			c.logger.Errorf("fail to send msg: %s", err)
		}
	}

}

func (c *CDMClient) recvMsgRoutine() {

	c.logger.Debugf("start receiving cdm message ")

	var err error

	for {

		select {
		case <-c.stop:
			c.logger.Debugf("close recv cdm msg")
			return
		default:
			recvMsg, recvErr := c.stream.Recv()

			if recvErr == io.EOF {
				c.closeConnection()
				continue
			}

			if recvErr != nil {
				c.closeConnection()
				continue
			}

			c.logger.Debugf("[%s] receive msg from docker manager", recvMsg.TxId)

			switch recvMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
				c.deleteRecvChan(recvMsg.TxId)
			case protogo.CDMType_CDM_TYPE_GET_STATE:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_GET_BYTECODE:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg
			case protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS:
				waitCh := c.getRecvChan(recvMsg.TxId)
				waitCh <- recvMsg

			default:
				c.logger.Errorf("unknown message type")
			}

			if err != nil {
				c.logger.Error(err)
			}

		}
	}

}

func (c *CDMClient) sendCDMMsg(msg *protogo.CDMMessage) error {
	c.logger.Debugf("send message: [%s]", msg)
	return c.stream.Send(msg)
}

// NewClientConn create rpc connection
func (c *CDMClient) NewClientConn() (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxRecvSize*1024*1024),
			grpc.MaxCallSendMsgSize(config.MaxSendSize*1024*1024),
		),
	}

	// just for mac development and pprof testing
	if !c.config.DockerVMUDSOpen {
		ip := "0.0.0.0"
		url := fmt.Sprintf("%s:%s", ip, config.TestPort)
		return grpc.Dial(url, dialOpts...)
	}

	dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
		unixAddress, _ := net.ResolveUnixAddr("unix", sock)
		conn, err := net.DialUnix("unix", nil, unixAddress)
		return conn, err
	}))

	sockAddress := filepath.Join(c.config.DockerVMMountPath, c.chainId, config.SockDir, config.SockName)

	return grpc.DialContext(context.Background(), sockAddress, dialOpts...)

}

// GetCDMClientStream get rpc stream
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}
