///*
//Copyright (C) BABEC. All rights reserved.
//
//SPDX-License-Identifier: Apache-2.0
//*/
package dockercontroller

//
//import (
//	"os"
//	"path/filepath"
//	"testing"
//
//	"github.com/spf13/viper"
//
//	"chainmaker.org/chainmaker/localconf/v2"
//)
//
//var (
//	path    = "./testData/"
//	ymlPath = "./wx-org1-solo-dockervm/test-chainmaker-docker-manager.yml"
//)
//
//func loadConfigFile(cmViper *viper.Viper) error {
//	ymlFile := path + ymlPath
//	if !filepath.IsAbs(ymlFile) {
//		ymlFile, _ = filepath.Abs(ymlFile)
//	}
//	cmViper.SetConfigFile(ymlFile)
//	if err := cmViper.ReadInConfig(); err != nil {
//		return err
//	}
//	logConfigFile := cmViper.GetString("log.config_file")
//	if logConfigFile != "" {
//		cmViper.SetConfigFile(logConfigFile)
//		if err := cmViper.MergeInConfig(); err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func InitConfig() error {
//	// 0. load env
//	cmViper := viper.New()
//	// 1. load the config file
//	if err := loadConfigFile(cmViper); err != nil {
//		return err
//	}
//	// 2. create new CMConfig instance
//	newCmConfig := &localconf.CMConfig{}
//	if err := cmViper.Unmarshal(newCmConfig); err != nil {
//		return err
//	}
//	localconf.ChainMakerConfig = newCmConfig
//	return nil
//}
//
//func TestInitConfig(t *testing.T) {
//	err := InitConfig()
//	if err != nil {
//		t.Fatalf("init config failed")
//	}
//}
//
//func TestNewDockerManagerAndGetContainer(t *testing.T) {
//	t.Helper()
//	t.Run("test1", func(t *testing.T) {
//		err := InitConfig()
//		if err != nil {
//			t.Fatalf("init config failed")
//		}
//		dm := NewDockerManager("chain1")
//		_, err = dm.getContainer(false)
//		if err != nil {
//			t.Errorf("getContainer failed!")
//		}
//		dm.StartCDMClient()
//	})
//}
//
//func TestStartContainer(t *testing.T) {
//	m := NewDockerManager("chain1")
//	t.Helper()
//	t.Run("testStartContainer", func(t *testing.T) {
//		err := m.StartContainer()
//		if err != nil {
//			t.Fatalf("Start container failed")
//		}
//	})
//
//	t.Run("test Stop And Remove VM", func(t *testing.T) {
//		err := m.StopAndRemoveVM()
//		if err != nil {
//			t.Fatalf("stop and remove vm failed")
//		}
//	})
//}
//
//func TestStopAndRemoveContainer(t *testing.T) {
//	m := NewDockerManager("chain1")
//	t.Run("test stop running container", func(t *testing.T) {
//		isRunning, err := m.getContainer(false)
//		if err != nil {
//			t.Fatalf("get Container failed")
//		}
//		if isRunning {
//			err = m.stopContainer()
//			if err != nil {
//				t.Fatalf("stop Container failed")
//			}
//		}
//	})
//
//	t.Run("test remove container", func(t *testing.T) {
//		containerExist, err := m.getContainer(true)
//		if err != nil {
//			t.Fatalf("get Contaniner failed")
//		}
//
//		if containerExist {
//			err = m.removeContainer()
//			if err != nil {
//				t.Fatalf("remove container failed")
//			}
//		}
//	})
//
//}
//
//func TestBuildAndRemoveImage(t *testing.T) {
//	m := NewDockerManager("chain1")
//	t.Run("test build and remove image", func(t *testing.T) {
//
//		imageExisted, err := m.imageExist()
//		if err != nil {
//			t.Fatalf("check image exist failed")
//		}
//
//		if imageExisted {
//			err = m.removeImage()
//			if err != nil {
//				t.Fatalf("remove image failed")
//			}
//		}
//
//		err = m.buildImage(m.dockerDir)
//		if err != nil {
//			t.Fatalf("build image failed")
//		}
//	})
//}
//
//func TestMain(m *testing.M) {
//	err := InitConfig()
//	if err != nil {
//		os.Exit(-1)
//	}
//	code := m.Run()
//	os.Exit(code)
//}
//
//func TestAll(t *testing.T) {
//	t.Run("TestInitConfig", TestInitConfig)
//	t.Run("TestNewDockerManagerAndGetContainer", TestNewDockerManagerAndGetContainer)
//	t.Run("TestStartContainer", TestStartContainer)
//	t.Run("TestStopAndRemoveContainer", TestStopAndRemoveContainer)
//	t.Run("TestBuildAndRemoveImage", TestBuildAndRemoveImage)
//}
