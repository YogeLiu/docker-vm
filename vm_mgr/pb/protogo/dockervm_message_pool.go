// Code generated by protoc-gen-go-vtproto. DO NOT EDIT.
// protoc-gen-go-vtproto version: v0.3.0
// source: dockervm_message.proto

package protogo

import (
	sync "sync"
)

var protoPool_DockerVMMessage = sync.Pool{
	New: func() interface{} {
		return &DockerVMMessage{}
	},
}

func (m *DockerVMMessage) ResetMsg() {
	m.Reset()
}
func (m *DockerVMMessage) ReturnToPool() {
	if m != nil {
		m.ResetMsg()
		protoPool_DockerVMMessage.Put(m)
	}
}
func DockerVMMessageFromPool() *DockerVMMessage {
	return protoPool_DockerVMMessage.Get().(*DockerVMMessage)
}
