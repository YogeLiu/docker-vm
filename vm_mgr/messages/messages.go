/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package messages

type GetProcessReqMsg struct {
	ContractName    string
	ContractVersion string
	ProcessNum      int
}

type LaunchSandboxRespMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
	Err          error
}

type ChangeSandboxRespMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
	Err          error
}

type CloseSandboxReqMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
}

type CloseSandboxRespMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
	Err          error
}

type SandboxExitRespMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
	Err          error
}

type ChangeStateReqMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
}

type ChangeSandboxReqMsg struct {
	ContractName string
	ContractVersion string
	ProcessName  string
}

type TxCompleteMsg struct {
	TxId string
}

type RequestGroupKey struct {
	ContractName string
	ContractVersion string
}

type CloseMsg struct {
	Msg string
}