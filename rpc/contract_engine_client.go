/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"path/filepath"
	"sync"

	"chainmaker.org/chainmaker/common/v2/ca"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

func NewContractEngineClient(chainId string, id uint64, logger protocol.Logger, cm interfaces.ContractEngineClientMgr) *ContractEngineClient {

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

func (c *ContractEngineClient) StartClient() error {

	c.logger.Infof("start contract engine client[%d]", c.id)
	conn, err := c.NewClientConn()
	if err != nil {
		c.logger.Errorf("client[%d] fail to create connection: %s", c.id, err)
		return err
	}

	go func() {
		select {
		case <-c.stopReceive:
			_ = conn.Close()
		case <-c.stopSend:
			_ = conn.Close()
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

func (c *ContractEngineClient) sendMsgRoutine() {

	c.logger.Infof("start sending contract engine message ")

	var err error

	for {
		select {
		case txReq := <-c.clientMgr.GetTxSendCh():
			c.logger.Debugf("[%s] send tx req, chan len: [%d]", txReq.TxId, c.clientMgr.GetTxSendChLen())
			err = c.sendMsg(txReq)
		case getByteCodeResp := <-c.clientMgr.GetByteCodeRespSendCh():
			c.logger.Debugf("[%s] send GetByteCode resp, chan len: [%d]", getByteCodeResp.TxId, c.clientMgr.GetByteCodeRespChLen())
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
				notify := c.clientMgr.GetReceiveNotify(c.chainId, receivedMsg.TxId)
				if notify == nil {
					c.logger.Warnf("[%s] fail to retrieve notify, tx notify is nil",
						receivedMsg.TxId)
					continue
				}
				c.clientMgr.DeleteNotify(c.chainId, receivedMsg.TxId)
				notify(receivedMsg)
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
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
	c.logger.Debugf("send message: [%s]", msg)
	return c.stream.Send(msg)
}

// NewClientConn create rpc connection
func (c *ContractEngineClient) NewClientConn() (*grpc.ClientConn, error) {

	// just for mac development and pprof testing
	if !c.config.DockerVMUDSOpen {
		url := fmt.Sprintf("%s:%s", c.config.ContractEngine.Host, c.config.ContractEngine.Port)
		dialOpts := []grpc.DialOption{
			grpc.WithInsecure(),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(int(c.config.ContractEngine.MaxSendMsgSize)*1024*1024),
				grpc.MaxCallSendMsgSize(int(c.config.ContractEngine.MaxSendMsgSize)*1024*1024),
			),
		}
		if !c.config.ContractEngine.TLSConfig.Enabled {
			return grpc.Dial(url, dialOpts...)
		}

		var caCert string
		if c.config.ContractEngine.TLSConfig.Cert != "" {
			caCert = c.config.ContractEngine.TLSConfig.Cert

		} else if c.config.ContractEngine.TLSConfig.CertFile != "" {
			certStr, err := ioutil.ReadFile(c.config.ContractEngine.TLSConfig.CertFile)
			if err != nil {
				return nil, err
			}
			caCert = string(certStr)

		} else {
			return nil, errors.New("new client connection failed, invalid tls config")
		}

		var key string
		if c.config.ContractEngine.TLSConfig.Key != "" {
			key = c.config.ContractEngine.TLSConfig.Key

		} else if c.config.ContractEngine.TLSConfig.PrivKeyFile != "" {
			keyStr, err := ioutil.ReadFile(c.config.ContractEngine.TLSConfig.PrivKeyFile)
			if err != nil {
				return nil, err
			}
			key = string(keyStr)

		} else {
			return nil, errors.New("new client connection failed, invalid tls config")
		}

		tlsRPCClient := ca.CAClient{
			ServerName: c.config.ContractEngine.TLSConfig.TLSHostName,
			CaPaths:    c.config.ContractEngine.TLSConfig.TrustRootPaths,
			CertBytes:  []byte(caCert),
			KeyBytes:   []byte(key),
			Logger:     c.logger,
		}

		cred, err := tlsRPCClient.GetCredentialsByCA()
		if err != nil {
			c.logger.Errorf("new gRPC failed, GetTLSCredentialsByCA err: %v", err)
			return nil, err
		}

		dialOpts = append(dialOpts, grpc.WithTransportCredentials(*cred))
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

	sockAddress := filepath.Join(c.config.DockerVMMountPath, c.chainId, config.SockDir, config.SockName)

	return grpc.DialContext(context.Background(), sockAddress, udsDialOpts...)

}

// GetClientStream get rpc stream
func GetClientStream(conn *grpc.ClientConn) (protogo.DockerVMRpc_DockerVMCommunicateClient, error) {
	return protogo.NewDockerVMRpcClient(conn).DockerVMCommunicate(context.Background())
}
