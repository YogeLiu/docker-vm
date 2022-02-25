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
	"strconv"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"
	"go.uber.org/atomic"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"google.golang.org/grpc"
)

const (
	txSize = 15000
)

type CDMClient struct {
	count               *atomic.Uint64
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
		count:               atomic.NewUint64(0),
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

func (c *CDMClient) GetCMConfig() *config.DockerVMConfig {
	return c.config
}

func (c *CDMClient) RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("register receive chan for [%s]", txId)

	_, ok := c.recvChMap[txId]
	if ok {
		c.logger.Errorf("[%s] fail to register receive chan cause chan already registered", txId)
		return utils.DuplicateTxIdError
	}

	c.recvChMap[txId] = recvCh
	return nil
}

func (c *CDMClient) getRecvChan(txId string) chan *protogo.CDMMessage {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.logger.Debugf("get receive chan for [%s]", txId)
	return c.recvChMap[txId]
}

func (c *CDMClient) deleteRecvChan(txId string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("delete receive chan for [%s]", txId)
	_, ok := c.recvChMap[txId]
	if ok {
		delete(c.recvChMap, txId)
	}
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

	go c.revMsgRoutine()

	return true
}

func (c *CDMClient) closeConnection() {
	c.logger.Errorf("client close grpc connection")
	// close two goroutine
	close(c.stop)
	// close stream
	err := c.stream.CloseSend()
	if err != nil {
		c.logger.Errorf("fail to close send stream %s", err)
		return
	}
}

//todo: test if server is killed, does sendMsg receive error or not
func (c *CDMClient) sendMsgRoutine() {

	c.logger.Infof("start sending cdm message ")

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
			c.logger.Debugf("close cdm send goroutine")
			return
		}

		if err != nil {
			c.logger.Errorf("fail to send msg: %s", err)
			// todo: exit goroutine and reconnect with server
		}
	}
}

func (c *CDMClient) revMsgRoutine() {

	c.logger.Infof("start receiving cdm message ")

	var waitCh chan *protogo.CDMMessage

	for {
		revMsg, revErr := c.stream.Recv()

		if revErr == io.EOF {
			c.logger.Error("client receive eof and exit receive goroutine")
			close(c.stop)
			return
		}

		if revErr != nil {
			c.logger.Errorf("client receive err and exit receive goroutine %s", revErr)
			close(c.stop)
			return
		}

		c.logger.Debugf("[%s] receive msg from docker manager", revMsg.TxId)

		switch revMsg.Type {
		case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
			waitCh = c.getRecvChan(revMsg.TxId)
			if waitCh == nil {
				c.logger.Errorf("[%s] fail to retrieve response chan, response chan is nil", revMsg.TxId)
				continue
			}
			waitCh <- revMsg
			c.deleteRecvChan(revMsg.TxId)
		case protogo.CDMType_CDM_TYPE_GET_STATE, protogo.CDMType_CDM_TYPE_GET_BYTECODE,
			protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR, protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR,
			protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER, protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER,
			protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS:
			waitCh = c.getRecvChan(revMsg.TxId)
			if waitCh == nil {
				c.logger.Errorf("[%s] fail to retrieve response chan, response chan is nil", revMsg.TxId)
				continue
			}
			waitCh <- revMsg
		default:
			c.logger.Errorf("unknown message type")
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

func (c *CDMClient) GetUniqueTxKey(txId string) string {
	var sb strings.Builder
	nextCount := c.count.Add(1)
	sb.WriteString(txId)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(nextCount, 10))
	return sb.String()
}
