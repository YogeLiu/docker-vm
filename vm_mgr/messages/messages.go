/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package messages

import (
	"time"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
)

// GetProcessReqMsg is the get process request msg (request group -> process manager)
type GetProcessReqMsg struct {
	ChainID         string
	ContractName    string
	ContractVersion string
	ProcessNum      int
}

// GetProcessRespMsg is the get process request msg (process manager -> request group)
type GetProcessRespMsg struct {
	IsOrig     bool
	ProcessNum int
}

//// LaunchSandboxRespMsg is the launch sandbox resp msg (process -> process manager)
//type LaunchSandboxRespMsg struct {
//	ContractName    string
//	ContractVersion string
//	ProcessName     string
//	Err             error
//}

//// CloseSandboxReqMsg is the close sandbox req msg (process manager -> process)
//type CloseSandboxReqMsg struct {
//	ContractName    string
//	ContractVersion string
//	ProcessName     string
//}

//// CloseSandboxRespMsg is the close sandbox resp msg (process -> process manager)
//type CloseSandboxRespMsg struct {
//	ContractName    string
//	ContractVersion string
//	ProcessName     string
//	Err             error
//}

// SandboxExitMsg is the sandbox exit resp msg (process -> process manager)
type SandboxExitMsg struct {
	ChainID         string
	ContractName    string
	ContractVersion string
	ProcessName     string
	Err             error
}

//// ChangeStateReqMsg is the change state req msg (process -> process manager)
//type ChangeStateReqMsg struct {
//	ContractName    string
//	ContractVersion string
//	ProcessName     string
//}

//// ChangeSandboxReqMsg is the change sandbox req msg (process manager -> process)
//type ChangeSandboxReqMsg struct {
//	ContractName    string
//	ContractVersion string
//	ProcessName     string
//}

// TxCompleteMsg is the tx complete msg
type TxCompleteMsg struct {
	TxId string
}

// RequestGroupKey is the request group key msg
type RequestGroupKey struct {
	ChainID         string
	ContractName    string
	ContractVersion string
}

// CloseMsg is the universal close msg
type CloseMsg struct {
	Msg string
}

// BadContractResp retry get bytecode
type BadContractResp struct {
	Tx                  *protogo.DockerVMMessage
	IsOrig              bool
	ContractFileVersion int64
}

// TxPayload is the tx payload msg
type TxPayload struct {
	Tx        *protogo.DockerVMMessage
	StartTime time.Time
}
