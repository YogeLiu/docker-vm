/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gas

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

func TestInitFuncGasUsed(t *testing.T) {
	type args struct {
		gasUsed    uint64
		parameters map[string][]byte
		keys       []string
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "testInitFuncGasUsed",
			args: args{
				gasUsed: 0,
				parameters: map[string][]byte{
					ContractParamCreatorOrgId: []byte("orgId1"),
					ContractParamCreatorRole:  []byte("creatorRole1"),
					ContractParamCreatorPk:    []byte("creatorPk1"),
					ContractParamSenderOrgId:  []byte("senderOrgId1"),
					ContractParamSenderRole:   []byte("senderRole11"),
					ContractParamSenderPk:     []byte("senderPk1"),
					ContractParamBlockHeight:  []byte("blockHeight1"),
					ContractParamTxId:         []byte("txId1"),
				},
				keys: []string{
					ContractParamCreatorOrgId,
					ContractParamCreatorRole,
					ContractParamCreatorPk,
					ContractParamSenderOrgId,
					ContractParamSenderRole,
					ContractParamSenderPk,
					ContractParamBlockHeight,
					ContractParamTxId,
				},
			},
			want:    10078,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitFuncGasUsed(tt.args.gasUsed, tt.args.parameters, tt.args.keys...)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitFuncGasUsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InitFuncGasUsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDelStateGasUsed(t *testing.T) {
	type args struct {
		gasUsed uint64
		value   []byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "delStateGasUsed",
			args: args{
				gasUsed: 0,
				value:   []byte("delStateGasUsed"),
			},
			want:    150,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DelStateGasUsed(tt.args.gasUsed, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("DelStateGasUsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DelStateGasUsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmitEventGasUsed(t *testing.T) {
	type args struct {
		gasUsed       uint64
		contractEvent *common.ContractEvent
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "emitEventGasUsed",
			args: args{
				gasUsed: 0,
				contractEvent: &common.ContractEvent{
					Topic:           "emitTopic",
					TxId:            "0x4ed18383472027c1fa035976cb57c5e6ea7e4de2569e6368c18ea31ff647d337",
					ContractName:    "contractName1",
					ContractVersion: "v1.0.0_beta",
					EventData:       []string{"eventData1", "eventData1"},
				},
			},
			want:    1020,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EmitEventGasUsed(tt.args.gasUsed, tt.args.contractEvent)
			if (err != nil) != tt.wantErr {
				t.Errorf("EmitEventGasUsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EmitEventGasUsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetStateGasUsed(t *testing.T) {
	type args struct {
		gasUsed uint64
		value   []byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "getStateGasUsed",
			args: args{
				gasUsed: 0,
				value:   []byte("getStateGasUsed"),
			},
			want:    15,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetStateGasUsed(tt.args.gasUsed, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetStateGasUsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetStateGasUsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPutStateGasUsed(t *testing.T) {
	type args struct {
		gasUsed      uint64
		contractName string
		key          string
		field        string
		value        []byte
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "putStateGasUsed",
			args: args{
				gasUsed:      0,
				contractName: "contractName1",
				key:          "key1",
				field:        "field1",
				value:        []byte("value1"),
			},
			want:    290,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PutStateGasUsed(tt.args.gasUsed, tt.args.contractName, tt.args.key, tt.args.field, tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutStateGasUsed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PutStateGasUsed() got = %v, want %v", got, tt.want)
			}
		})
	}
}
