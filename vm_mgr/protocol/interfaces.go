/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package protocol

import (
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/module/tx_requests"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/pb_sdk/protogo"
)

type Scheduler interface {
	// GetTxReqCh get tx req chan
	GetTxReqCh() chan *protogo.TxRequest

	// GetTxResponseCh get tx response chan
	GetTxResponseCh() chan *protogo.TxResponse

	// GetGetStateReqCh get get_state request chan
	GetGetStateReqCh() chan *protogo.CDMMessage

	// RegisterResponseCh register response chan
	RegisterResponseCh(chainId, txId string, responseCh chan *protogo.CDMMessage)

	// RegisterCrossContractResponseCh register cross contract response chan
	RegisterCrossContractResponseCh(chainId, responseId string, responseCh chan *SDKProtogo.DMSMessage)

	// GetCrossContractResponseCh get cross contract response chan
	GetCrossContractResponseCh(chainId, responseId string) chan *SDKProtogo.DMSMessage

	// GetResponseChByTxId get response chan
	GetResponseChByTxId(chainId, txId string) chan *protogo.CDMMessage

	// GetByteCodeReqCh get get_bytecode request chan
	GetByteCodeReqCh() chan *protogo.CDMMessage

	GetCrossContractReqCh() chan *protogo.TxRequest

	ReturnErrorResponse(chainId, txId, errMsg string)

	ReturnErrorCrossContractResponse(txRequest *protogo.TxRequest, resp *SDKProtogo.DMSMessage)

	// time statistics
	RegisterTxElapsedTime(txRequest *protogo.TxRequest, startTime int64)
	AddTxSysCallElapsedTime(txId string, sysCallElapsedTime *tx_requests.SysCallElapsedTime)
	AddTxCallContractElapsedTime(txId string, callContractElapsedTime *tx_requests.SysCallElapsedTime)
	RemoveTxElapsedTime(txId string)
	GetTxElapsedTime(txId string) *tx_requests.TxElapsedTime
}

type UserController interface {
	// GetAvailableUser get available user
	GetAvailableUser() (*security.User, error)
	// FreeUser free user
	FreeUser(user *security.User) error
}
