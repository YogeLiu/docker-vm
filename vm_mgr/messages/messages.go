package messages

import "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"

type GetProcessReqMsg struct {
	ContractName    string
	ContractVersion string
	ProcessNum      int
	RequestGroup    *interfaces.RequestGroup
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