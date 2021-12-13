package protocol

import (
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/vm_mgr/pb_sdk/protogo"
)

type Scheduler interface {
	// GetTxReqCh get tx req chan
	GetTxReqCh() chan *protogo.TxRequest

	// GetTxResponseCh get tx response chan
	GetTxResponseCh() chan *protogo.TxResponse

	// GetGetStateReqCh get get_state request chan
	GetGetStateReqCh() chan *protogo.CDMMessage

	// RegisterResponseCh register response chan
	RegisterResponseCh(txId string, responseCh chan *protogo.CDMMessage)

	// RegisterCrossContractResponseCh register cross contract response chan
	RegisterCrossContractResponseCh(responseId string, responseCh chan *SDKProtogo.DMSMessage)

	// GetCrossContractResponseCh get cross contract response chan
	GetCrossContractResponseCh(responseId string) chan *SDKProtogo.DMSMessage

	// GetResponseChByTxId get response chan
	GetResponseChByTxId(txId string) chan *protogo.CDMMessage

	// GetByteCodeReqCh get get_bytecode request chan
	GetByteCodeReqCh() chan *protogo.CDMMessage

	GetCrossContractReqCh() chan *protogo.TxRequest
}

type UserController interface {
	// GetAvailableUser get available user
	GetAvailableUser() (*security.User, error)
	// FreeUser free user
	FreeUser(user *security.User) error
}

type Peer interface {
	// IsAlive get peer alive state
	IsAlive() bool
	// Size get peer waiting queue size
	Size() int
	// AddTx add new tx into peer waiting queue
	AddTx(tx int)
}

type Balancer interface {
	// SetStrategy set balancer strategy
	SetStrategy(_strategy int)
	// AddPeer add new peer into balancer
	AddPeer(key string, peer Peer) error
	// GetPeer get avaiable peer
	GetPeer(key string) Peer
}
