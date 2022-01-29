/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	context "context"
	reflect "reflect"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// MockCDMRpcClient is a mock of CDMRpcClient interface.
type MockCDMRpcClient struct {
	ctrl     *gomock.Controller
	recorder *MockCDMRpcClientMockRecorder
}

// MockCDMRpcClientMockRecorder is the mock recorder for MockCDMRpcClient.
type MockCDMRpcClientMockRecorder struct {
	mock *MockCDMRpcClient
}

// NewMockCDMRpcClient creates a new mock instance.
func NewMockCDMRpcClient(ctrl *gomock.Controller) *MockCDMRpcClient {
	mock := &MockCDMRpcClient{ctrl: ctrl}
	mock.recorder = &MockCDMRpcClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCDMRpcClient) EXPECT() *MockCDMRpcClientMockRecorder {
	return m.recorder
}

// CDMCommunicate mocks base method.
func (m *MockCDMRpcClient) CDMCommunicate(ctx context.Context, opts ...grpc.CallOption) (protogo.CDMRpc_CDMCommunicateClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "CDMCommunicate", varargs...)
	ret0, _ := ret[0].(protogo.CDMRpc_CDMCommunicateClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CDMCommunicate indicates an expected call of CDMCommunicate.
func (mr *MockCDMRpcClientMockRecorder) CDMCommunicate(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CDMCommunicate", reflect.TypeOf((*MockCDMRpcClient)(nil).CDMCommunicate), varargs...)
}

// MockCDMRpc_CDMCommunicateClient is a mock of CDMRpc_CDMCommunicateClient interface.
type MockCDMRpc_CDMCommunicateClient struct {
	ctrl     *gomock.Controller
	recorder *MockCDMRpc_CDMCommunicateClientMockRecorder
}

// MockCDMRpc_CDMCommunicateClientMockRecorder is the mock recorder for MockCDMRpc_CDMCommunicateClient.
type MockCDMRpc_CDMCommunicateClientMockRecorder struct {
	mock *MockCDMRpc_CDMCommunicateClient
}

// NewMockCDMRpc_CDMCommunicateClient creates a new mock instance.
func NewMockCDMRpc_CDMCommunicateClient(ctrl *gomock.Controller) *MockCDMRpc_CDMCommunicateClient {
	mock := &MockCDMRpc_CDMCommunicateClient{ctrl: ctrl}
	mock.recorder = &MockCDMRpc_CDMCommunicateClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCDMRpc_CDMCommunicateClient) EXPECT() *MockCDMRpc_CDMCommunicateClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method.
func (m *MockCDMRpc_CDMCommunicateClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).CloseSend))
}

// Context mocks base method.
func (m *MockCDMRpc_CDMCommunicateClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).Context))
}

// Header mocks base method.
func (m *MockCDMRpc_CDMCommunicateClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).Header))
}

// Recv mocks base method.
func (m *MockCDMRpc_CDMCommunicateClient) Recv() (*protogo.CDMMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*protogo.CDMMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockCDMRpc_CDMCommunicateClient) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockCDMRpc_CDMCommunicateClient) Send(arg0 *protogo.CDMMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).Send), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockCDMRpc_CDMCommunicateClient) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).SendMsg), m)
}

// Trailer mocks base method.
func (m *MockCDMRpc_CDMCommunicateClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer.
func (mr *MockCDMRpc_CDMCommunicateClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockCDMRpc_CDMCommunicateClient)(nil).Trailer))
}

// MockCDMRpcServer is a mock of CDMRpcServer interface.
type MockCDMRpcServer struct {
	ctrl     *gomock.Controller
	recorder *MockCDMRpcServerMockRecorder
}

// MockCDMRpcServerMockRecorder is the mock recorder for MockCDMRpcServer.
type MockCDMRpcServerMockRecorder struct {
	mock *MockCDMRpcServer
}

// NewMockCDMRpcServer creates a new mock instance.
func NewMockCDMRpcServer(ctrl *gomock.Controller) *MockCDMRpcServer {
	mock := &MockCDMRpcServer{ctrl: ctrl}
	mock.recorder = &MockCDMRpcServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCDMRpcServer) EXPECT() *MockCDMRpcServerMockRecorder {
	return m.recorder
}

// CDMCommunicate mocks base method.
func (m *MockCDMRpcServer) CDMCommunicate(arg0 protogo.CDMRpc_CDMCommunicateServer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CDMCommunicate", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// CDMCommunicate indicates an expected call of CDMCommunicate.
func (mr *MockCDMRpcServerMockRecorder) CDMCommunicate(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CDMCommunicate", reflect.TypeOf((*MockCDMRpcServer)(nil).CDMCommunicate), arg0)
}

// MockCDMRpc_CDMCommunicateServer is a mock of CDMRpc_CDMCommunicateServer interface.
type MockCDMRpc_CDMCommunicateServer struct {
	ctrl     *gomock.Controller
	recorder *MockCDMRpc_CDMCommunicateServerMockRecorder
}

// MockCDMRpc_CDMCommunicateServerMockRecorder is the mock recorder for MockCDMRpc_CDMCommunicateServer.
type MockCDMRpc_CDMCommunicateServerMockRecorder struct {
	mock *MockCDMRpc_CDMCommunicateServer
}

// NewMockCDMRpc_CDMCommunicateServer creates a new mock instance.
func NewMockCDMRpc_CDMCommunicateServer(ctrl *gomock.Controller) *MockCDMRpc_CDMCommunicateServer {
	mock := &MockCDMRpc_CDMCommunicateServer{ctrl: ctrl}
	mock.recorder = &MockCDMRpc_CDMCommunicateServerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCDMRpc_CDMCommunicateServer) EXPECT() *MockCDMRpc_CDMCommunicateServerMockRecorder {
	return m.recorder
}

// Context mocks base method.
func (m *MockCDMRpc_CDMCommunicateServer) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).Context))
}

// Recv mocks base method.
func (m *MockCDMRpc_CDMCommunicateServer) Recv() (*protogo.CDMMessage, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*protogo.CDMMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).Recv))
}

// RecvMsg mocks base method.
func (m_2 *MockCDMRpc_CDMCommunicateServer) RecvMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "RecvMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) RecvMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).RecvMsg), m)
}

// Send mocks base method.
func (m *MockCDMRpc_CDMCommunicateServer) Send(arg0 *protogo.CDMMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).Send), arg0)
}

// SendHeader mocks base method.
func (m *MockCDMRpc_CDMCommunicateServer) SendHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendHeader indicates an expected call of SendHeader.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) SendHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHeader", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).SendHeader), arg0)
}

// SendMsg mocks base method.
func (m_2 *MockCDMRpc_CDMCommunicateServer) SendMsg(m interface{}) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "SendMsg", m)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) SendMsg(m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).SendMsg), m)
}

// SetHeader mocks base method.
func (m *MockCDMRpc_CDMCommunicateServer) SetHeader(arg0 metadata.MD) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetHeader", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetHeader indicates an expected call of SetHeader.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) SetHeader(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetHeader", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).SetHeader), arg0)
}

// SetTrailer mocks base method.
func (m *MockCDMRpc_CDMCommunicateServer) SetTrailer(arg0 metadata.MD) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetTrailer", arg0)
}

// SetTrailer indicates an expected call of SetTrailer.
func (mr *MockCDMRpc_CDMCommunicateServerMockRecorder) SetTrailer(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetTrailer", reflect.TypeOf((*MockCDMRpc_CDMCommunicateServer)(nil).SetTrailer), arg0)
}
