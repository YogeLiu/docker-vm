/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msgbus

//go:generate stringer -type=Topic
type Topic int

const (
	Invalid Topic = iota
	ProposedBlock
	VerifyBlock
	VerifyResult
	CommitBlock
	ProposeState
	TxPoolSignal
	BlockInfo
	ContractEventInfo

	// For Net Service
	SendConsensusMsg
	RecvConsensusMsg
	SendSyncBlockMsg
	RecvSyncBlockMsg
	SendTxPoolMsg
	RecvTxPoolMsg

	BuildProposal
)
