/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	configFileName = "../../../test/testdata/config/vm.yml"
)

func TestNewUser(t *testing.T) {

	SetConfig()

	type args struct {
		id int
	}
	tests := []struct {
		name string
		args args
		want *User
	}{
		{
			name: "TestNewUser",
			args: args{id: 0},
			want: &User{
				Uid:      0,
				Gid:      0,
				UserName: "u-0",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUser(tt.args.id)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUser() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestUser_GetGid(t *testing.T) {

	SetConfig()

	type fields struct {
		Uid      int
		Gid      int
		UserName string
		SockPath string
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "TestUser_GetGid_1",
			fields: fields{
				Uid:      0,
				Gid:      0,
				UserName: "u-0",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: 0,
		},
		{
			name: "TestUser_GetGid_2",
			fields: fields{
				Uid:      999999,
				Gid:      999999,
				UserName: "u-999999",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: 999999,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Uid:      tt.fields.Uid,
				Gid:      tt.fields.Gid,
				UserName: tt.fields.UserName,
				SockPath: tt.fields.SockPath,
			}
			if got := u.GetGid(); got != tt.want {
				t.Errorf("GetGid() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestUser_GetSockPath(t *testing.T) {

	SetConfig()

	type fields struct {
		Uid      int
		Gid      int
		UserName string
		SockPath string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestUser_GetSockPath_1",
			fields: fields{
				Uid:      0,
				Gid:      0,
				UserName: "u-0",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
		},
		{
			name: "TestUser_GetSockPath_2",
			fields: fields{
				Uid:      999999,
				Gid:      999999,
				UserName: "u-999999",
				SockPath: "/",
			},
			want: "/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Uid:      tt.fields.Uid,
				Gid:      tt.fields.Gid,
				UserName: tt.fields.UserName,
				SockPath: tt.fields.SockPath,
			}
			if got := u.GetSockPath(); got != tt.want {
				t.Errorf("GetGid() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestUser_GetUid(t *testing.T) {

	SetConfig()

	type fields struct {
		Uid      int
		Gid      int
		UserName string
		SockPath string
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name: "TestUser_GetUid_1",
			fields: fields{
				Uid:      0,
				Gid:      0,
				UserName: "u-0",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: 0,
		},
		{
			name: "TestUser_GetUid_2",
			fields: fields{
				Uid:      999999,
				Gid:      999999,
				UserName: "u-999999",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: 999999,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Uid:      tt.fields.Uid,
				Gid:      tt.fields.Gid,
				UserName: tt.fields.UserName,
				SockPath: tt.fields.SockPath,
			}
			if got := u.GetUid(); got != tt.want {
				t.Errorf("GetUid() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestUser_GetUserName(t *testing.T) {

	SetConfig()

	type fields struct {
		Uid      int
		Gid      int
		UserName string
		SockPath string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestUser_GetUserName_1",
			fields: fields{
				Uid:      0,
				Gid:      0,
				UserName: "u-0",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: "u-0",
		},
		{
			name: "TestUser_GetUserName_2",
			fields: fields{
				Uid:      999999,
				Gid:      999999,
				UserName: "u-999999",
				SockPath: filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName),
			},
			want: "u-999999",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &User{
				Uid:      tt.fields.Uid,
				Gid:      tt.fields.Gid,
				UserName: tt.fields.UserName,
				SockPath: tt.fields.SockPath,
			}
			if got := u.GetUserName(); got != tt.want {
				t.Errorf("GetUserName() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func SetConfig() {
	_ = config.InitConfig(configFileName)
}
