// Code generated by MockGen. DO NOT EDIT.
// Source: ./interfaces.go

// Package mock_protocol is a generated GoMock package.
package protocol

import (
	reflect "reflect"

	protogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"

	security "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	protogo0 "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	gomock "github.com/golang/mock/gomock"
)

// MockScheduler is a mock of Scheduler interface.
type MockScheduler struct {
	ctrl     *gomock.Controller
	recorder *MockSchedulerMockRecorder
}

// MockSchedulerMockRecorder is the mock recorder for MockScheduler.
type MockSchedulerMockRecorder struct {
	mock *MockScheduler
}

// NewMockScheduler creates a new mock instance.
func NewMockScheduler(ctrl *gomock.Controller) *MockScheduler {
	mock := &MockScheduler{ctrl: ctrl}
	mock.recorder = &MockSchedulerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockScheduler) EXPECT() *MockSchedulerMockRecorder {
	return m.recorder
}

// GetByteCodeReqCh mocks base method.
func (m *MockScheduler) GetByteCodeReqCh() chan *protogo0.CDMMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByteCodeReqCh")
	ret0, _ := ret[0].(chan *protogo0.CDMMessage)
	return ret0
}

// GetByteCodeReqCh indicates an expected call of GetByteCodeReqCh.
func (mr *MockSchedulerMockRecorder) GetByteCodeReqCh() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByteCodeReqCh", reflect.TypeOf((*MockScheduler)(nil).GetByteCodeReqCh))
}

// GetCrossContractReqCh mocks base method.
func (m *MockScheduler) GetCrossContractReqCh() chan *protogo0.TxRequest {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCrossContractReqCh")
	ret0, _ := ret[0].(chan *protogo0.TxRequest)
	return ret0
}

// GetCrossContractReqCh indicates an expected call of GetCrossContractReqCh.
func (mr *MockSchedulerMockRecorder) GetCrossContractReqCh() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCrossContractReqCh", reflect.TypeOf((*MockScheduler)(nil).GetCrossContractReqCh))
}

// GetCrossContractResponseCh mocks base method.
func (m *MockScheduler) GetCrossContractResponseCh(responseId string) chan *protogo.DMSMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCrossContractResponseCh", responseId)
	ret0, _ := ret[0].(chan *protogo.DMSMessage)
	return ret0
}

// GetCrossContractResponseCh indicates an expected call of GetCrossContractResponseCh.
func (mr *MockSchedulerMockRecorder) GetCrossContractResponseCh(responseId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCrossContractResponseCh", reflect.TypeOf((*MockScheduler)(nil).GetCrossContractResponseCh), responseId)
}

// GetGetStateReqCh mocks base method.
func (m *MockScheduler) GetGetStateReqCh() chan *protogo0.CDMMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetGetStateReqCh")
	ret0, _ := ret[0].(chan *protogo0.CDMMessage)
	return ret0
}

// GetGetStateReqCh indicates an expected call of GetGetStateReqCh.
func (mr *MockSchedulerMockRecorder) GetGetStateReqCh() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetGetStateReqCh", reflect.TypeOf((*MockScheduler)(nil).GetGetStateReqCh))
}

// GetResponseChByTxId mocks base method.
func (m *MockScheduler) GetResponseChByTxId(txId string) chan *protogo0.CDMMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResponseChByTxId", txId)
	ret0, _ := ret[0].(chan *protogo0.CDMMessage)
	return ret0
}

// GetResponseChByTxId indicates an expected call of GetResponseChByTxId.
func (mr *MockSchedulerMockRecorder) GetResponseChByTxId(txId interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResponseChByTxId", reflect.TypeOf((*MockScheduler)(nil).GetResponseChByTxId), txId)
}

// GetTxReqCh mocks base method.
func (m *MockScheduler) GetTxReqCh() chan *protogo0.TxRequest {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTxReqCh")
	ret0, _ := ret[0].(chan *protogo0.TxRequest)
	return ret0
}

// GetTxReqCh indicates an expected call of GetTxReqCh.
func (mr *MockSchedulerMockRecorder) GetTxReqCh() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTxReqCh", reflect.TypeOf((*MockScheduler)(nil).GetTxReqCh))
}

// GetTxResponseCh mocks base method.
func (m *MockScheduler) GetTxResponseCh() chan *protogo0.TxResponse {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTxResponseCh")
	ret0, _ := ret[0].(chan *protogo0.TxResponse)
	return ret0
}

// GetTxResponseCh indicates an expected call of GetTxResponseCh.
func (mr *MockSchedulerMockRecorder) GetTxResponseCh() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTxResponseCh", reflect.TypeOf((*MockScheduler)(nil).GetTxResponseCh))
}

// RegisterCrossContractResponseCh mocks base method.
func (m *MockScheduler) RegisterCrossContractResponseCh(responseId string, responseCh chan *protogo.DMSMessage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterCrossContractResponseCh", responseId, responseCh)
}

// RegisterCrossContractResponseCh indicates an expected call of RegisterCrossContractResponseCh.
func (mr *MockSchedulerMockRecorder) RegisterCrossContractResponseCh(responseId, responseCh interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterCrossContractResponseCh", reflect.TypeOf((*MockScheduler)(nil).RegisterCrossContractResponseCh), responseId, responseCh)
}

// RegisterResponseCh mocks base method.
func (m *MockScheduler) RegisterResponseCh(txId string, responseCh chan *protogo0.CDMMessage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterResponseCh", txId, responseCh)
}

// RegisterResponseCh indicates an expected call of RegisterResponseCh.
func (mr *MockSchedulerMockRecorder) RegisterResponseCh(txId, responseCh interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterResponseCh", reflect.TypeOf((*MockScheduler)(nil).RegisterResponseCh), txId, responseCh)
}

// MockUserController is a mock of UserController interface.
type MockUserController struct {
	ctrl     *gomock.Controller
	recorder *MockUserControllerMockRecorder
}

// MockUserControllerMockRecorder is the mock recorder for MockUserController.
type MockUserControllerMockRecorder struct {
	mock *MockUserController
}

// NewMockUserController creates a new mock instance.
func NewMockUserController(ctrl *gomock.Controller) *MockUserController {
	mock := &MockUserController{ctrl: ctrl}
	mock.recorder = &MockUserControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUserController) EXPECT() *MockUserControllerMockRecorder {
	return m.recorder
}

// FreeUser mocks base method.
func (m *MockUserController) FreeUser(user *security.User) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FreeUser", user)
	ret0, _ := ret[0].(error)
	return ret0
}

// FreeUser indicates an expected call of FreeUser.
func (mr *MockUserControllerMockRecorder) FreeUser(user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FreeUser", reflect.TypeOf((*MockUserController)(nil).FreeUser), user)
}

// GetAvailableUser mocks base method.
func (m *MockUserController) GetAvailableUser() (*security.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAvailableUser")
	ret0, _ := ret[0].(*security.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAvailableUser indicates an expected call of GetAvailableUser.
func (mr *MockUserControllerMockRecorder) GetAvailableUser() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAvailableUser", reflect.TypeOf((*MockUserController)(nil).GetAvailableUser))
}
