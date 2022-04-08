/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package protocol

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"
)

type Scheduler interface {
	// GetTxReqCh get tx req chan
	GetTxReqCh(string) chan *protogo.TxRequest

	// GetTxResponseCh get tx response chan
	GetTxResponseCh(string) chan *protogo.TxResponse

	// GetGetStateReqCh get get_state request chan
	GetGetStateReqCh(string) chan *protogo.CDMMessage

	// RegisterResponseCh register response chan
	RegisterResponseCh(chainId, txId string, responseCh chan *protogo.CDMMessage)

	// RegisterCrossContractResponseCh register cross contract response chan
	RegisterCrossContractResponseCh(chainId, responseId string, responseCh chan *SDKProtogo.DMSMessage)

	// GetCrossContractResponseCh get cross contract response chan
	GetCrossContractResponseCh(chainId, responseId string) chan *SDKProtogo.DMSMessage

	// GetResponseChByTxId get response chan
	GetResponseChByTxId(chainId, txId string) chan *protogo.CDMMessage

	// GetByteCodeReqCh get get_bytecode request chan
	GetByteCodeReqCh(string) chan *protogo.CDMMessage

	GetCrossContractReqCh(string) chan *protogo.TxRequest

	ReturnErrorResponse(string, string, string)

	ReturnErrorCrossContractResponse(txRequest *protogo.TxRequest, resp *SDKProtogo.DMSMessage)
}

type UserController interface {
	// GetAvailableUser get available user
	GetAvailableUser() (*security.User, error)
	// FreeUser free user
	FreeUser(user *security.User) error
}
