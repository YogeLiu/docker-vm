package mocks

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

type MockProcessManager struct{}

func (pm *MockProcessManager) Start() {
	panic("implement me")
}

func (pm *MockProcessManager) SetScheduler(scheduler interfaces.RequestScheduler) {
	panic("implement me")
}

func (pm *MockProcessManager) PutMsg(msg interface{}) error {
	panic("implement me")
}

func (pm *MockProcessManager) GetProcessByName(processName string) (interfaces.Process, bool) {
	if processName == "wrong" {
		return nil, false
	}
	return &MockProcess{}, true
}

func (pm *MockProcessManager) GetProcessNumByContractKey(contractName, contractVersion string) int {
	panic("implement me")
}

func (pm *MockProcessManager) ChangeProcessState(processName string, toBusy bool) error {
	return nil
}

type MockProcess struct {
	ProcessName     string
	ContractName    string
	ContractVersion string
}

func (p *MockProcess) PutMsg(msg *protogo.DockerVMMessage) {
}

func (p *MockProcess) Start() {
	panic("implement me")
}

func (p *MockProcess) GetProcessName() string {
	return p.ProcessName
}

func (p *MockProcess) GetContractName() string {
	return p.ContractName
}

func (p *MockProcess) GetContractVersion() string {
	return p.ContractVersion
}

func (p *MockProcess) GetUser() interfaces.User {
	panic("implement me")
}

func (p *MockProcess) SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error {
	return nil
}

func (p *MockProcess) ChangeSandbox(contractName, contractVersion, processName string) error {
	return nil
}

func (p *MockProcess) CloseSandbox() error {
	panic("implement me")
}
