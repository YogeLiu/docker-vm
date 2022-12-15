/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"context"
	"reflect"

	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/pb_sdk/protogo"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MockDMSRpcClient is a mock of DMSRpcClient interface.
type MockDMSRpcClient struct {
	ctrl     *gomock.Controller
	recorder *MockDMSRpcClientMockRecorder
}

// MockDMSRpcClientMockRecorder is the mock recorder for MockDMSRpcClient.
type MockDMSRpcClientMockRecorder struct {
	mock *MockDMSRpcClient
}

// NewMockDMSRpcClient creates a new mock instance.
func NewMockDMSRpcClient(ctrl *gomock.Controller) *MockDMSRpcClient {
	mock := &MockDMSRpcClient{ctrl: ctrl}
	mock.recorder = &MockDMSRpcClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDMSRpcClient) EXPECT() *MockDMSRpcClientMockRecorder {
	return m.recorder
}

// DMSCommunicate mocks base method.
func (m *MockDMSRpcClient) DMSCommunicate(ctx context.Context, opts ...grpc.CallOption) (SDKProtogo.DMSRpc_DMSCommunicateClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DMSCommunicate", varargs...)
	ret0, _ := ret[0].(SDKProtogo.DMSRpc_DMSCommunicateClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DMSCommunicate indicates an expected call of DMSCommunicate.
func (mr *MockDMSRpcClientMockRecorder) DMSCommunicate(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DMSCommunicate", reflect.TypeOf((*MockDMSRpcClient)(nil).DMSCommunicate), varargs...)
}

// MockDMSRpc_DMSCommunicateClient is a mock of DMSRpc_DMSCommunicateClient interface.
type MockDMSRpc_DMSCommunicateClient struct {
	ctrl     *gomock.Controller
	recorder *MockDMSRpc_DMSCommunicateClientMockRecorder
}

// MockDMSRpc_DMSCommunicateClientMockRecorder is the mock recorder for MockDMSRpc_DMSCommunicateClient.
type MockDMSRpc_DMSCommunicateClientMockRecorder struct {
	mock *MockDMSRpc_DMSCommunicateClient
}

// NewMockDMSRpc_DMSCommunicateClient creates a new mock instance.
func NewMockDMSRpc_DMSCommunicateClient(ctrl *gomock.Controller) *MockDMSRpc_DMSCommunicateClient {
	mock := &MockDMSRpc_DMSCommunicateClient{ctrl: ctrl}
	mock.recorder = &MockDMSRpc_DMSCommunicateClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDMSRpc_DMSCommunicateClient) EXPECT() *MockDMSRpc_DMSCommunicateClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockDMSRpc_DMSCommunicateClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockDMSRpc_DMSCommunicateClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).Context))
}

// Header mocks base method.
func (m *MockDMSRpc_DMSCommunicateClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockDMSRpc_DMSCommunicateClient) Recv() (*SDKProtogo.DMSMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*SDKProtogo.DMSMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockDMSRpc_DMSCommunicateClient) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockDMSRpc_DMSCommunicateClient) Send(arg0 *SDKProtogo.DMSMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).Send), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockDMSRpc_DMSCommunicateClient) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).SendMsg), m)
}

// Trailer mocks base method.
func (m *MockDMSRpc_DMSCommunicateClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockDMSRpc_DMSCommunicateClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockDMSRpc_DMSCommunicateClient)(nil).Trailer))
}

// MockDMSRpcServer is a mock of DMSRpcServer interface.
type MockDMSRpcServer struct {
	ctrl     *gomock.Controller
	recorder *MockDMSRpcServerMockRecorder
}

// MockDMSRpcServerMockRecorder is the mock recorder for MockDMSRpcServer.
type MockDMSRpcServerMockRecorder struct {
	mock *MockDMSRpcServer
}

// NewMockDMSRpcServer creates a new mock instance.
func NewMockDMSRpcServer(ctrl *gomock.Controller) *MockDMSRpcServer {
	mock := &MockDMSRpcServer{ctrl: ctrl}
	mock.recorder = &MockDMSRpcServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDMSRpcServer) EXPECT() *MockDMSRpcServerMockRecorder {
	return m.recorder
}

// DMSCommunicate mocks base method.
func (m *MockDMSRpcServer) DMSCommunicate(arg0 SDKProtogo.DMSRpc_DMSCommunicateServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DMSCommunicate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// DMSCommunicate indicates an expected call of DMSCommunicate.
func (mr *MockDMSRpcServerMockRecorder) DMSCommunicate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DMSCommunicate", reflect.TypeOf((*MockDMSRpcServer)(nil).DMSCommunicate), arg0)
}

// MockDMSRpc_DMSCommunicateServer is a mock of DMSRpc_DMSCommunicateServer interface.
type MockDMSRpc_DMSCommunicateServer struct {
	ctrl     *gomock.Controller
	recorder *MockDMSRpc_DMSCommunicateServerMockRecorder
}

// MockDMSRpc_DMSCommunicateServerMockRecorder is the mock recorder for MockDMSRpc_DMSCommunicateServer.
type MockDMSRpc_DMSCommunicateServerMockRecorder struct {
	mock *MockDMSRpc_DMSCommunicateServer
}

// NewMockDMSRpc_DMSCommunicateServer creates a new mock instance.
func NewMockDMSRpc_DMSCommunicateServer(ctrl *gomock.Controller) *MockDMSRpc_DMSCommunicateServer {
	mock := &MockDMSRpc_DMSCommunicateServer{ctrl: ctrl}
	mock.recorder = &MockDMSRpc_DMSCommunicateServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDMSRpc_DMSCommunicateServer) EXPECT() *MockDMSRpc_DMSCommunicateServerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockDMSRpc_DMSCommunicateServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).Context))
}

// Recv mocks base method.
func (m *MockDMSRpc_DMSCommunicateServer) Recv() (*SDKProtogo.DMSMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*SDKProtogo.DMSMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockDMSRpc_DMSCommunicateServer) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockDMSRpc_DMSCommunicateServer) Send(arg0 *SDKProtogo.DMSMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).Send), arg0)
}

// SendHeader mocks base method.
func (m *MockDMSRpc_DMSCommunicateServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockDMSRpc_DMSCommunicateServer) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).SendMsg), m)
}

// SetHeader mocks base method.
func (m *MockDMSRpc_DMSCommunicateServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockDMSRpc_DMSCommunicateServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockDMSRpc_DMSCommunicateServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockDMSRpc_DMSCommunicateServer)(nil).SetTrailer), arg0)
}

// MockProcessPoolInterface is a mock of ProcessPoolInterface interface.
type MockProcessPoolInterface struct {
	ctrl     *gomock.Controller
	recorder *MockProcessPoolInterfaceMockRecorder
}

// MockProcessPoolInterfaceMockRecorder is the mock recorder for MockProcessPoolInterface.
type MockProcessPoolInterfaceMockRecorder struct {
	mock *MockProcessPoolInterface
}

// NewMockProcessPoolInterface creates a new mock instance.
func NewMockProcessPoolInterface(ctrl *gomock.Controller) *MockProcessPoolInterface {
	mock := &MockProcessPoolInterface{ctrl: ctrl}
	mock.recorder = &MockProcessPoolInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProcessPoolInterface) EXPECT() *MockProcessPoolInterfaceMockRecorder {
	return m.recorder
}

// RetrieveProcessContext mocks base method.
//func (m *MockProcessPoolInterface) RetrieveProcessContext(initialProcessName string) *ProcessContext {
//	m.ctrl.T.Helper()
//	ret := m.ctrl.Call(m, "RetrieveProcessContext", initialProcessName)
//	ret0, _ := ret[0].(*ProcessContext)
//	return ret0
//}

//// RetrieveProcessContext indicates an expected call of RetrieveProcessContext.
//func (mr *MockProcessPoolInterfaceMockRecorder) RetrieveProcessContext(initialProcessName interface{}) *gomock.Call {
//	mr.mock.ctrl.T.Helper()
//	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RetrieveProcessContext", reflect.TypeOf((*MockProcessPoolInterface)(nil).RetrieveProcessContext), initialProcessName)
//}
