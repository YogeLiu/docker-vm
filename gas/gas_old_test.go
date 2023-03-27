/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gas

import (
	"testing"

	"chainmaker.org/chainmaker/pb-go/v2/common"
)

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
			got, err := EmitEventGasUsedLt2312(tt.args.gasUsed, tt.args.contractEvent)
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
			got, err := GetStateGasUsedLt2312(tt.args.gasUsed, tt.args.value)
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
			got, err := PutStateGasUsedLt2312(tt.args.gasUsed, tt.args.contractName, tt.args.key, tt.args.field, tt.args.value)
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

func TestInitFuncGasUsed(t *testing.T) {
	type args struct {
		gasUsed          uint64
		configDefaultGas uint64
	}
	tests := []struct {
		name    string
		args    args
		want    uint64
		wantErr bool
	}{
		{
			name: "t1",
			args: args{
				gasUsed:          0,
				configDefaultGas: defaultInvokeBaseGas,
			},
			want:    defaultInvokeBaseGas + initFuncGas,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := InitFuncGasUsedLt2312(tt.args.gasUsed, tt.args.configDefaultGas)
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
