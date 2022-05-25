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
	return nil
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

func (p *MockProcess) SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer) {}

func (p *MockProcess) ChangeSandbox(contractName, contractVersion, processName string) error {
	return nil
}

func (p *MockProcess) CloseSandbox() error {
	panic("implement me")
}

type MockRequestScheduler struct{}

func (rs *MockRequestScheduler) Start() {
	//TODO implement me
	panic("implement me")
}

func (rs *MockRequestScheduler) PutMsg(msg interface{}) error {
	return nil
}

func (rs *MockRequestScheduler) GetRequestGroup(contractName, contractVersion string) (interfaces.RequestGroup, bool) {
	//TODO implement me
	panic("implement me")
}

type MockRequestGroup struct{}

func (rg *MockRequestGroup) Start() {
	//TODO implement me
	panic("implement me")
}

func (rg *MockRequestGroup) PutMsg(msg interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (rg *MockRequestGroup) GetContractPath() string {
	//TODO implement me
	panic("implement me")
}

func (rg *MockRequestGroup) GetTxCh(isOrig bool) chan *protogo.DockerVMMessage {
	//TODO implement me
	panic("implement me")
}
