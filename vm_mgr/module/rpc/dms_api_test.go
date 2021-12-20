/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"reflect"
	"testing"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
)

func TestDMSApi_DMSCommunicate(t *testing.T) {
	type fields struct {
		logger      *zap.SugaredLogger
		processPool ProcessPoolInterface
	}
	type args struct {
		stream protogo.DMSRpc_DMSCommunicateServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &DMSApi{
				logger:      tt.fields.logger,
				processPool: tt.fields.processPool,
			}
			if err := s.DMSCommunicate(tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("DMSCommunicate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewDMSApi(t *testing.T) {
	log := utils.GetLogHandler()
	c := gomock.NewController(t)
	defer c.Finish()

	type args struct {
		processPool ProcessPoolInterface
	}
	tests := []struct {
		name string
		args args
		want *DMSApi
	}{
		{
			name: "testNewDMSApi",
			args: args{
				processPool: nil,
			},
			want: &DMSApi{
				logger:      log,
				processPool: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDMSApi(tt.args.processPool)
			got.logger = log
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewDMSApi() = %v, want %v", got, tt.want)
			}
		})
	}
}
