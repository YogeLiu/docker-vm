/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	recvChMap     map[string]chan *protogo.CDMMessage
	stream        protogo.CDMRpc_CDMCommunicateClient
	logger        *logger.CMLogger
	config        *config.DockerVMConfig
	ReconnectChan chan bool
	ConnStatus    bool
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
		config:              vmConfig,
		ReconnectChan:       make(chan bool),
		ConnStatus:          false,
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
	if c.ConnStatus == false {
		c.logger.Errorf("cdm client stream not ready, waiting reconnect, txid: %s", txId)
		return errors.New("cdm client not connected")
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("register receive chan for [%s]", txId)

	_, ok := c.recvChMap[txId]
	if ok {
		c.logger.Errorf("[%s] fail to register receive chan cause chan already registered", txId)
		return utils.ErrDuplicateTxId
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

func (c *CDMClient) getAndDeleteRecvChan(txId string) chan *protogo.CDMMessage {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("get receive chan for [%s] and delete", txId)
	receiveChan, ok := c.recvChMap[txId]
	if ok {
		delete(c.recvChMap, txId)
		return receiveChan
	}
	c.logger.Warnf("cannot find receive chan for [%s] and return nil", txId)
	return nil
}

func (c *CDMClient) DeleteRecvChan(txId string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.logger.Debugf("[%s] delete receive chan", txId)
	_, ok := c.recvChMap[txId]
	if ok {
		delete(c.recvChMap, txId)
		return true
	}
	c.logger.Debugf("[%s] delete receive chan fail, receive chan is already deleted", txId)
	return false
}

func (c *CDMClient) StartClient() bool {

	c.logger.Debugf("start cdm rpc..")
	conn, err := c.NewClientConn()
	if err != nil {
		c.logger.Errorf("fail to create connection: %s", err)
		return false
	}

	// todo 是否一定需要添加 connection close 逻辑
	go func() {
		<-c.ReconnectChan
		conn.Close()
	}()

	stream, err := GetCDMClientStream(conn)
	if err != nil {
		c.logger.Errorf("fail to get connection stream: %s", err)
		conn.Close()
		return false
	}

	c.stream = stream

	go c.sendMsgRoutine()

	go c.receiveMsgRoutine()

	return true
}

//todo: test if server is killed, does sendMsg receive error or not
func (c *CDMClient) sendMsgRoutine() {

	c.logger.Infof("start sending cdm message ")

	var err error

	for {
		select {
		case txMsg := <-c.txSendCh:
			c.logger.Debugf("[%s] send tx req, chan len: [%d]", txMsg.TxId, len(c.txSendCh))
			err = c.sendCDMMsg(txMsg)
		case stateMsg := <-c.stateResponseSendCh:
			c.logger.Debugf("[%s] send syscall resp, chan len: [%d]", stateMsg.TxId, len(c.stateResponseSendCh))
			err = c.sendCDMMsg(stateMsg)
		case <-c.ReconnectChan:
			c.logger.Debugf("close cdm send goroutine")
			return
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			c.logger.Errorf("fail to send msg: err: %s, err message: %s, err code: %s", err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				// run into error need reconnect
				c.ConnStatus = false
				close(c.ReconnectChan)
				return
			}
		}
	}
}

func (c *CDMClient) receiveMsgRoutine() {

	c.logger.Infof("start receiving cdm message ")

	var waitCh chan *protogo.CDMMessage

	for {

		select {
		case <-c.ReconnectChan:
			c.logger.Debugf("close cdm client receive goroutine")
			return
		default:
			receivedMsg, revErr := c.stream.Recv()

			if revErr != nil {
				// run into error need reconnect
				c.logger.Errorf("client receive err and exit receive goroutine %s", revErr)
				c.ConnStatus = false
				close(c.ReconnectChan)
				return
			}

			c.logger.Debugf("[%s] receive msg from docker manager", receivedMsg.TxId)

			switch receivedMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
				waitCh = c.getAndDeleteRecvChan(receivedMsg.TxId)
				if waitCh == nil {
					c.logger.Warnf("[%s] fail to retrieve response chan, tx response chan is nil",
						receivedMsg.TxId)
					continue
				}
				waitCh <- receivedMsg
			case protogo.CDMType_CDM_TYPE_GET_STATE, protogo.CDMType_CDM_TYPE_GET_BYTECODE,
				protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR, protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR,
				protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER, protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER,
				protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS:
				waitCh = c.getRecvChan(receivedMsg.TxId)
				if waitCh == nil {
					c.logger.Warnf("[%s] fail to retrieve response chan, response chan is nil", receivedMsg.TxId)
					continue
				}
				waitCh <- receivedMsg
			default:
				c.logger.Errorf("unknown message type, received msg: [%v]", receivedMsg)
			}
		}
	}
}

func (c *CDMClient) sendCDMMsg(msg *protogo.CDMMessage) error {
	c.logger.Debugf("send message: [%s]", msg.TxId)
	return c.stream.Send(msg)
}

// NewClientConn create rpc connection
func (c *CDMClient) NewClientConn() (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(int(utils.GetMaxRecvMsgSizeFromConfig(c.config)*1024*1024)),
			grpc.MaxCallSendMsgSize(int(utils.GetMaxSendMsgSizeFromConfig(c.config)*1024*1024)),
		),
	}

	if c.config.DockerVMUDSOpen {
		// connect unix domain socket
		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
			unixAddress, _ := net.ResolveUnixAddr("unix", sock)
			conn, err := net.DialUnix("unix", nil, unixAddress)
			return conn, err
		}))

		sockAddress := filepath.Join(c.config.DockerVMMountPath, c.chainId, config.SockDir, config.SockName)

		c.logger.Infof("connect docker vm manager: %s", sockAddress)
		return grpc.DialContext(context.Background(), sockAddress, dialOpts...)
	} else {
		// connect vm from tcp
		url := utils.GetURLFromConfig(c.config)
		c.logger.Infof("connect docker vm manager: %s", url)
		return grpc.Dial(url, dialOpts...)
	}

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

func (c *CDMClient) NeedSendContractByteCode() bool {
	return !c.config.DockerVMUDSOpen
}
