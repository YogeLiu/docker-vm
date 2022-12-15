/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/utils"
	"go.uber.org/zap"
)

func TestNewUsersManager(t *testing.T) {
	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}

	log := logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, config.DockerLogDir)
	userQueue := utils.NewFixedFIFO(userNum)
	usersManager := &UsersManager{
		userQueue: userQueue,
		logger:    log,
		userNum:   userNum,
	}

	tests := []struct {
		name string
		want *UsersManager
	}{
		{
			name: "testNewUsersManager",
			want: usersManager,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewUsersManager()
			got.logger = log
			got.userQueue = userQueue
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUsersManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsersManager_CreateNewUsers(t *testing.T) {
	currentPath, _ := os.Getwd()
	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}

	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}

	log := logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, currentPath)
	userQueue := utils.NewFixedFIFO(userNum)
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "testCreateNewUsers",
			fields: fields{
				userQueue: userQueue,
				logger:    log,
				userNum:   userNum,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UsersManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}
			if err := u.CreateNewUsers(); (err != nil) != tt.wantErr {
				t.Errorf("CreateNewUsers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUsersManager_FreeUser(t *testing.T) {
	currentPath, _ := os.Getwd()

	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}

	log := logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, currentPath)
	userQueue := utils.NewFixedFIFO(userNum)

	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}

	type args struct {
		user *security.User
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testFreeUser",
			fields: fields{
				userQueue: userQueue,
				logger:    log,
				userNum:   userNum,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UsersManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}
			if err := u.FreeUser(tt.args.user); (err != nil) != tt.wantErr {
				t.Errorf("FreeUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUsersManager_GetAvailableUser(t *testing.T) {

	currentPath, _ := os.Getwd()

	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}

	log := logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, currentPath)
	userQueue := utils.NewFixedFIFO(userNum)

	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}
	tests := []struct {
		name    string
		fields  fields
		want    *security.User
		wantErr bool
	}{
		{
			name: "testGetAvailableUser",
			fields: fields{
				userQueue: userQueue,
				logger:    log,
				userNum:   userNum,
			},
			want: &security.User{
				Uid: userNum,
				Gid: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UsersManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}

			go func() {
				for {
					_ = u.userQueue.Enqueue(&security.User{
						Uid: userNum,
					})
				}
			}()

			got, err := u.GetAvailableUser()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAvailableUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAvailableUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsersManager_constructNewUser(t *testing.T) {
	userId := 666
	userName := fmt.Sprintf("u-%d", userId)
	sockPath := filepath.Join(config.DMSDir, config.DMSSockPath)

	currentPath, _ := os.Getwd()
	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}
	log := logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, currentPath)
	userQueue := utils.NewFixedFIFO(userNum)

	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}
	type args struct {
		userId int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *security.User
	}{
		{
			name: "testconstructNewUser",
			fields: fields{
				userQueue: userQueue,
				logger:    log,
				userNum:   userNum,
			},
			args: args{userId: userId},
			want: &security.User{
				Uid:      userId,
				Gid:      userId,
				UserName: userName,
				SockPath: sockPath,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UsersManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}
			if got := u.constructNewUser(tt.args.userId); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructNewUser() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsersManager_generateNewUser(t *testing.T) {
	userId := 666
	currentPath, _ := os.Getwd()
	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}
	log := logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, currentPath)
	userQueue := utils.NewFixedFIFO(userNum)

	type fields struct {
		userQueue *utils.FixedFIFO
		logger    *zap.SugaredLogger
		userNum   int
	}

	type args struct {
		newUserId int
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testGenerateNewUserBad",
			fields: fields{
				userQueue: userQueue,
				logger:    log,
				userNum:   userNum,
			},
			args:    args{newUserId: userId},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UsersManager{
				userQueue: tt.fields.userQueue,
				logger:    tt.fields.logger,
				userNum:   tt.fields.userNum,
			}

			if err := u.generateNewUser(tt.args.newUserId); (err != nil) != tt.wantErr {
				t.Errorf("generateNewUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
