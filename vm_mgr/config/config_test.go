package config

import (
	"testing"
	"time"
)

const (
	configFileName = "./vm.yml"
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
		{"good case", DockerVMConfig, 2000 * time.Millisecond},
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
			if got := c.GetBusyTimeout(); got != tt.want {
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
		{"good case", DockerVMConfig, 5 * time.Second},
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
			if got := c.GetConnectionTimeout(); got != tt.want {
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
		{"good case", DockerVMConfig, 300},
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
		{"good case", DockerVMConfig, 200 * time.Millisecond},
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
			if got := c.GetReadyTimeout(); got != tt.want {
				t.Errorf("GetReadyTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetReleasePeriod(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, 10 * time.Second},
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
			if got := c.GetReleasePeriod(); got != tt.want {
				t.Errorf("GetReleasePeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetReleaseRate(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   float64
	}{
		{"good case", DockerVMConfig, float64(30) / 100.0},
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

func Test_conf_GetServerKeepAliveTime(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, 60 * time.Second},
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
			if got := c.GetServerKeepAliveTime(); got != tt.want {
				t.Errorf("GetServerKeepAliveTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetServerKeepAliveTimeout(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, 20 * time.Second},
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
			if got := c.GetServerKeepAliveTimeout(); got != tt.want {
				t.Errorf("GetServerKeepAliveTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_conf_GetServerMinInterval(t *testing.T) {
	initConfig()
	tests := []struct {
		name   string
		fields *conf
		want   time.Duration
	}{
		{"good case", DockerVMConfig, 60 * time.Second},
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
			if got := c.GetServerMinInterval(); got != tt.want {
				t.Errorf("GetServerMinInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func initConfig() {
	InitConfig(configFileName)
}
