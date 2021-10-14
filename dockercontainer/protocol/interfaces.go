package protocol

import (
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/dockercontainer/pb_sdk/protogo"
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
