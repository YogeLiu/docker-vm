/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"os"
	"testing"
	"time"
)

const (
	configFileName = "../../test/testdata/vm.yml"
)

func TestInitConfig(t *testing.T) {
	type args struct {
		configFileName string
		dockerMountDir string
		configFileType string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"good case", args{configFileName: configFileName}, false},
		{"wrong path", args{configFileName: "test/testdata/vm_wrong.yml"}, true},
		{"empty path", args{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := InitConfig(tt.args.configFileName); (err != nil) != tt.wantErr {
				t.Errorf("InitConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_conf_GetBusyTimeout(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, DockerVMConfig.Process.ExecTxTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				RPC:      tt.fields.RPC,
				Process:  tt.fields.Process,
				Log:      tt.fields.Log,
				Pprof:    tt.fields.Pprof,
				Contract: tt.fields.Contract,
			}
			if got := c.Process.ExecTxTimeout; got != tt.want {
				t.Errorf("GetBusyTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetConnectionTimeout(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, DockerVMConfig.RPC.ConnectionTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				RPC:      tt.fields.RPC,
				Process:  tt.fields.Process,
				Log:      tt.fields.Log,
				Pprof:    tt.fields.Pprof,
				Contract: tt.fields.Contract,
			}
			if got := c.RPC.ConnectionTimeout; got != tt.want {
				t.Errorf("GetConnectionTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetMaxUserNum(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   int
	}{
		{"good case", DockerVMConfig, DockerVMConfig.GetMaxUserNum()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				RPC:      tt.fields.RPC,
				Process:  tt.fields.Process,
				Log:      tt.fields.Log,
				Pprof:    tt.fields.Pprof,
				Contract: tt.fields.Contract,
			}
			if got := c.GetMaxUserNum(); got != tt.want {
				t.Errorf("GetMaxUserNum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetReadyTimeout(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, DockerVMConfig.Process.WaitingTxTime},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				RPC:      tt.fields.RPC,
				Process:  tt.fields.Process,
				Log:      tt.fields.Log,
				Pprof:    tt.fields.Pprof,
				Contract: tt.fields.Contract,
			}
			if got := c.Process.WaitingTxTime; got != tt.want {
				t.Errorf("GetReadyTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

//func Test_conf_GetReleasePeriod(t *testing.T) {
//	initConfig()
//	tests := []struct {
//		name   string
//		fields *conf
//		want   time.Duration
//	}{
//		{"good case", DockerVMConfig, 10 * time.Second},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &conf{
//				RPC:      tt.fields.RPC,
//				Process:  tt.fields.Process,
//				Log:      tt.fields.Log,
//				Pprof:    tt.fields.Pprof,
//				Contract: tt.fields.Contract,
//			}
//			if got := c.GetReleasePeriod(); got != tt.want {
//				t.Errorf("GetReleasePeriod() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
func Test_conf_GetReleaseRate(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   float64
	}{
		{"good case", DockerVMConfig, DockerVMConfig.GetReleaseRate()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				RPC:      tt.fields.RPC,
				Process:  tt.fields.Process,
				Log:      tt.fields.Log,
				Pprof:    tt.fields.Pprof,
				Contract: tt.fields.Contract,
			}
			if got := c.GetReleaseRate(); got != tt.want {
				t.Errorf("GetReleaseRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_SetEnv(t *testing.T) {

	os.Setenv("CHAIN_RPC_PORT", "21215")
	os.Setenv("MAX_CONN_TIMEOUT", "1000")
	defer os.Unsetenv("CHAIN_RPC_PORT")
	defer os.Unsetenv("MAX_CONN_TIMEOUT")

	initConfig()

	tests := []struct {
		name     string
		fields   *conf
		wantPort int
		wantTime time.Duration
	}{
		{"good case", DockerVMConfig, 21215, 1000 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &conf{
				RPC:      tt.fields.RPC,
				Process:  tt.fields.Process,
				Log:      tt.fields.Log,
				Pprof:    tt.fields.Pprof,
				Contract: tt.fields.Contract,
			}
			if c.RPC.ChainRPCPort != tt.wantPort {
				t.Errorf("c.RPC.ChainRPCPort = %v, want %v", c.RPC.ChainRPCPort, tt.wantPort)
			}
			if c.RPC.ConnectionTimeout != tt.wantTime {
				t.Errorf("c.RPC.ConnectionTimeout = %v, want %v", c.RPC.ConnectionTimeout, tt.wantTime)
			}
		})
	}

}

//func Test_conf_GetServerKeepAliveTime(t *testing.T) {
//	initConfig()
//	tests := []struct {
//		name   string
//		fields *conf
//		want   time.Duration
//	}{
//		{"good case", DockerVMConfig, 60 * time.Second},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &conf{
//				RPC:      tt.fields.RPC,
//				Process:  tt.fields.Process,
//				Log:      tt.fields.Log,
//				Pprof:    tt.fields.Pprof,
//				Contract: tt.fields.Contract,
//			}
//			if got := c.GetServerKeepAliveTime(); got != tt.want {
//				t.Errorf("GetServerKeepAliveTime() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_conf_GetServerKeepAliveTimeout(t *testing.T) {
//	initConfig()
//	tests := []struct {
//		name   string
//		fields *conf
//		want   time.Duration
//	}{
//		{"good case", DockerVMConfig, 20 * time.Second},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &conf{
//				RPC:      tt.fields.RPC,
//				Process:  tt.fields.Process,
//				Log:      tt.fields.Log,
//				Pprof:    tt.fields.Pprof,
//				Contract: tt.fields.Contract,
//			}
//			if got := c.GetServerKeepAliveTimeout(); got != tt.want {
//				t.Errorf("GetServerKeepAliveTimeout() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func Test_conf_GetServerMinInterval(t *testing.T) {
//	initConfig()
//	tests := []struct {
//		name   string
//		fields *conf
//		want   time.Duration
//	}{
//		{"good case", DockerVMConfig, 60 * time.Second},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &conf{
//				RPC:      tt.fields.RPC,
//				Process:  tt.fields.Process,
//				Log:      tt.fields.Log,
//				Pprof:    tt.fields.Pprof,
//				Contract: tt.fields.Contract,
//			}
//			if got := c.GetServerMinInterval(); got != tt.want {
//				t.Errorf("GetServerMinInterval() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func initConfig() {
	InitConfig(configFileName)
}
