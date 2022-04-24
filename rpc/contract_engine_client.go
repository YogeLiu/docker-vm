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

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.uber.org/atomic"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"
)

const (
	txSize = 15000
)

type ContractEngineClient struct {
	// TxId anti duplication
	count *atomic.Uint64
	// Distinguish sock files of different chains
	chainId string
	// Not passed directly, through function call
	txSendCh chan *protogo.DockerVMMessage // used to send tx to docker-go instance
	// syscall response channel
	respSendCh chan *protogo.DockerVMMessage // used to receive message from docker-go
	lock       sync.RWMutex
	// key: txId, value: chan, used to receive tx response from docker-go
	notifies map[string]func(msg *protogo.DockerVMMessage)
	stream   protogo.DockerVMRpc_DockerVMCommunicateClient

	logger      *logger.CMLogger
	stopSend    chan struct{}
	stopReceive chan struct{}
	config      *config.DockerVMConfig
}

func NewCDMClient(chainId string, vmConfig *config.DockerVMConfig) *ContractEngineClient {

	return &ContractEngineClient{
		count:       atomic.NewUint64(0),
		chainId:     chainId,
		txSendCh:    make(chan *protogo.DockerVMMessage, txSize), // tx request
		notifies:    make(map[string]func(msg *protogo.DockerVMMessage)),
		stream:      nil,
		logger:      logger.GetLoggerByChain(logger.MODULE_VM, chainId),
		stopSend:    make(chan struct{}),
		stopReceive: make(chan struct{}),
		config:      vmConfig,
	}
}

func (c *ContractEngineClient) PutTxRequest(msg *protogo.DockerVMMessage) {
	c.txSendCh <- msg
}

func (c *ContractEngineClient) GetTxSendChLen() int {
	return len(c.txSendCh)
}

func (c *ContractEngineClient) PutResponse(msg *protogo.DockerVMMessage) {
	c.respSendCh <- msg
}

func (c *ContractEngineClient) GetResponseSendChLen() int {
	return len(c.respSendCh)
}

// func (c *ContractEngineClient) GetStateResponseSendCh() chan *protogo.DockerVMMessage {
// 	return c.stateResponseSendCh
// }

func (c *ContractEngineClient) GetCMConfig() *config.DockerVMConfig {
	return c.config
}

func (c *ContractEngineClient) PutTxRequestWithNotify(msg *protogo.DockerVMMessage, notify func(msg *protogo.DockerVMMessage)) error {
	if err := c.registerNotify(msg.TxId, notify); err != nil {
		return err
	}

	c.PutTxRequest(msg)
	return nil
}

func (c *ContractEngineClient) registerNotify(txKey string, notify func(msg *protogo.DockerVMMessage)) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("register receive notify for [%s]", txKey)

	_, ok := c.notifies[txKey]
	if ok {
		c.logger.Errorf("[%s] fail to register receive notify cause notify already registered", txKey)
	}

	c.notifies[txKey] = notify

	return nil
}

func (c *ContractEngineClient) getCallback(txKey string) func(msg *protogo.DockerVMMessage) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.logger.Debugf("get callback for [%s]", txKey)
	return c.notifies[txKey]
}

func (c *ContractEngineClient) getAndDeleteCallback(txKey string) func(msg *protogo.DockerVMMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("get callback for [%s] and delete", txKey)
	callback, ok := c.notifies[txKey]
	if !ok {
		c.logger.Warnf("cannot find callback for [%s] and return nil", txKey)
		return nil
	}
	delete(c.notifies, txKey)
	return callback
}

func (c *ContractEngineClient) DeleteNotify(txId string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("[%s] delete receive chan", txId)
	_, ok := c.notifies[txId]
	if ok {
		delete(c.notifies, txId)
		return true
	}
	c.logger.Debugf("[%s] delete receive chan fail, receive chan is already deleted", txId)
	return false
}

func (c *ContractEngineClient) StartClient() bool {

	c.logger.Debugf("start contract engine client rpc..")
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

	go c.sendMsgRoutine()

	go c.receiveMsgRoutine()

	return true
}

//todo: test if server is killed, does sendMsg receive error or not
func (c *ContractEngineClient) sendMsgRoutine() {

	c.logger.Infof("start sending cdm message ")

	var err error

	for {
		select {
		case txMsg := <-c.txSendCh:
			c.logger.Debugf("[%s] send tx req, chan len: [%d]", txMsg.TxId, len(c.txSendCh))
			err = c.sendCDMMsg(txMsg)
		case stateMsg := <-c.respSendCh:
			c.logger.Debugf("[%s] send syscall resp, chan len: [%d]", stateMsg.TxId, len(c.respSendCh))
			err = c.sendCDMMsg(stateMsg)
		case <-c.stopSend:
			c.logger.Debugf("close cdm send goroutine")
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

	c.logger.Infof("start receiving cdm message ")

	// var waitCh chan *protogo.DockerVMMessage
	// var callback func(msg *protogo.DockerVMMessage)

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

			c.logger.Debugf("[%s] receive msg from docker manager", receivedMsg.TxId)

			switch receivedMsg.Type {
			case protogo.DockerVMType_TX_RESPONSE:
				callback := c.getAndDeleteCallback(receivedMsg.TxId)
				if callback == nil {
					c.logger.Warnf("[%s] fail to retrieve callback, tx callback is nil",
						receivedMsg.TxId)
					continue
				}
				callback(receivedMsg)
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				callback := c.getCallback(receivedMsg.TxId)
				if callback == nil {
					c.logger.Warnf("[%s] fail to retrieve callback, tx callback is nil", receivedMsg.TxId)
					continue
				}
				callback(receivedMsg)
			default:
				c.logger.Errorf("unknown message type, received msg: [%v]", receivedMsg)
			}
		}
	}
}

func (c *ContractEngineClient) sendCDMMsg(msg *protogo.DockerVMMessage) error {
	c.logger.Debugf("send message: [%s]", msg)
	return c.stream.Send(msg)
}

// NewClientConn create rpc connection
func (c *ContractEngineClient) NewClientConn() (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(int(utils.GetMaxRecvMsgSizeFromConfig(c.config)*1024*1024)),
			grpc.MaxCallSendMsgSize(int(utils.GetMaxSendMsgSizeFromConfig(c.config)*1024*1024)),
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
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.DockerVMRpc_DockerVMCommunicateClient, error) {
	return protogo.NewDockerVMRpcClient(conn).DockerVMCommunicate(context.Background())
}

func (c *ContractEngineClient) GetUniqueTxKey(txId string) string {
	var sb strings.Builder
	nextCount := c.count.Add(1)
	sb.WriteString(txId)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(nextCount, 10))
	return sb.String()
}
