/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
	"go.uber.org/zap"
	"reflect"
	"testing"
)

func TestNewUsersManager(t *testing.T) {

	SetConfig()

	tests := []struct {
		name string
	}{
		{
			name: "TestNewUsersManager",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = NewUsersManager()
		})
	}
}

func TestUserManager_BatchCreateUsers(t *testing.T) {

	SetConfig()

	config.DockerVMConfig.Process.MaxOriginalProcessNum = 1

	mgr := NewUsersManager()

	type fields struct {
		userManger *UserManager
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "TestUserManager_BatchCreateUsers",
			fields:  fields{userManger: mgr},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := mgr
			if err := u.BatchCreateUsers(); (err != nil) != tt.wantErr {
				t.Errorf("BatchCreateUsers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUserManager_FreeUser(t *testing.T) {

	SetConfig()

	user := NewUser(10000)

	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}
	tests := []struct {
		name    string
		fields  fields
		want    interfaces.User
		wantErr bool
	}{
		{
			name: "TestUserManager_FreeUser",
			fields: fields{
				userQueue: utils.NewFixedFIFO(config.DockerVMConfig.GetMaxUserNum()),
				logger:    logger.NewTestDockerLogger(),
				userNum:   config.DockerVMConfig.GetMaxUserNum(),
			},
			want:    user,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}
			err := u.FreeUser(user)
			if (err != nil) != tt.wantErr {
				t.Errorf("FreeUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got, err := u.GetAvailableUser()
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FreeUser() got = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestUserManager_GetAvailableUser(t *testing.T) {

	SetConfig()

	user1 := NewUser(10000)
	user2 := NewUser(10001)

	userQueue := utils.NewFixedFIFO(config.DockerVMConfig.GetMaxUserNum())
	err := userQueue.Enqueue(user1)
	if err != nil {
		t.Error(err.Error())
		return
	}
	err = userQueue.Enqueue(user2)
	if err != nil {
		t.Error(err.Error())
		return
	}

	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}
	tests := []struct {
		name    string
		fields  fields
		want    interfaces.User
		wantErr bool
	}{
		{
			name: "TestUserManager_GetAvailableUser1",
			fields: fields{
				userQueue: userQueue,
				logger:    logger.NewTestDockerLogger(),
				userNum:   config.DockerVMConfig.GetMaxUserNum(),
			},
			want:    user1,
			wantErr: false,
		},
		{
			name: "TestUserManager_GetAvailableUser2",
			fields: fields{
				userQueue: userQueue,
				logger:    logger.NewTestDockerLogger(),
				userNum:   config.DockerVMConfig.GetMaxUserNum(),
			},
			want:    user2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}
			got, err := u.GetAvailableUser()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAvailableUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAvailableUser() got = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestUserManager_ReleaseUsers(t *testing.T) {

	SetConfig()

	config.DockerVMConfig.Process.MaxOriginalProcessNum = 1

	mgr := NewUsersManager()

	type fields struct {
		userManger *UserManager
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name:    "TestUserManager_ReleaseUsers",
			fields:  fields{userManger: mgr},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := tt.fields.userManger
			if err := u.ReleaseUsers(); (err != nil) != tt.wantErr {
				t.Errorf("ReleaseUsers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
