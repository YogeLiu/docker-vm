package interfaces

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
)

type EventType int

type Event struct {
	Id        uint64
	EventType EventType
}

const (
	EventType_ConnectionStopped EventType = iota
)

type ContractEngineClientMgr interface {
	forClient
	forRuntimeInstance
	Start() error
	Stop() error
}

type forClient interface {
	GetTxSendCh() chan *protogo.DockerVMMessage                               // runtime instance 向Mgr中放入消息，client消费消息
	PutEvent(event *Event)                                                    // 关闭CLIENT等事件
	GetByteCodeRespSendCh() chan *protogo.DockerVMMessage                     // runtime instance 向Mgr中放入消息，client消费消息
	GetReceiveNotify(chainId, txId string) func(msg *protogo.DockerVMMessage) // 接收 GetByteCodeReq 消息和错误消息
	GetVMConfig() *config.DockerVMConfig                                      // 获取VM配置
}

// TODO: rename
type forRuntimeInstance interface {
	PutTxRequestWithNotify(
		txRequest *protogo.DockerVMMessage,
		chainId string,
		notify func(msg *protogo.DockerVMMessage),
	) error
	PutByteCodeResp(getByteCodeResp *protogo.DockerVMMessage)
	DeleteNotify(chainId, txId string) bool
	GetUniqueTxKey(txId string) string
	NeedSendContractByteCode() bool
	HasActiveConnections() bool
	GetVMConfig() *config.DockerVMConfig // 获取VM配置
	GetTxSendChLen() int
	GetByteCodeRespChLen() int
}

type RuntimeService interface {
	RegisterSandboxMsgNotify(
		chainId, txId string,
		respNotify func(
			msg *protogo.DockerVMMessage,
			sendF func(*protogo.DockerVMMessage),
		),
	) error
	DeleteSandboxMsgNotify(chainId, txId string) bool
}
