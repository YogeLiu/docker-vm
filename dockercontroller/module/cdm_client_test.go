/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package module

//import (
//	"fmt"
//	"io/ioutil"
//	"os"
//	"path/filepath"
//	"testing"
//
//	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
//	"chainmaker.org/chainmaker-go/localconf"
//	"chainmaker.org/chainmaker/common/v2/random/uuid"
//	"github.com/gogo/protobuf/proto"
//	"github.com/spf13/viper"
//)
//
///*
//	运行次测试需先启动chainmaker-go
//*/
//var (
//	path         = "../testData/"
//	ymlPath      = "./wx-org1-solo-dockervm/test-chainmaker-cdm-client.yml"
//	ContractPath = "../../../../../test/wasm/docker-go-contract20_big"
//	ContractName = "contract001"
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
//	fmt.Println("init")
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
//func TestNewClientConn(t *testing.T) {
//	c, err := NewClientConn()
//	if c == nil || err != nil {
//		t.Fatalf("NewClientConn failed!")
//	}
//}
//
//func TestStartClient(t *testing.T) {
//	t.Helper()
//	t.Run("testStartClient", func(t *testing.T) {
//		client := NewCDMClient("chain1")
//		res := client.StartClient()
//		if res != true {
//			t.Fatalf("Start Client failed")
//		}
//		txId := GetRandTxId()
//		cdmMsg := contractCreateMsg(txId)
//		err := client.sendCDMMsg(cdmMsg)
//		if err != nil {
//			t.Fatalf("sendCDMMsg failed")
//		}
//		client.closeConnection()
//	})
//}
//
//func TestCDMClient_RegisterAndDeleteRecvChan(t *testing.T) {
//	c := NewCDMClient("chain1")
//	t.Helper()
//	t.Run("testRegisterAndDeleteRecvChan", func(t *testing.T) {
//		response := make(chan *protogo.CDMMessage)
//		txId := GetRandTxId()
//		c.RegisterRecvChan(txId, response)
//		if c.recvChMap[txId] != response {
//			t.Errorf("CDMClient_RegisterRecvChan failed!")
//		}
//		c.deleteRecvChan(txId)
//		if _, ok := c.recvChMap[txId]; ok {
//			t.Errorf("CDMClient_deleteRecvChan failed!")
//		}
//	})
//}
//
//func GetRandTxId() string {
//	return uuid.GetUUID() + uuid.GetUUID()
//}
//
//func contractCreateMsg(txId string) *protogo.CDMMessage {
//	// construct cdm message
//	params := make(map[string]string)
//	ContractBin, _ := ioutil.ReadFile(ContractPath)
//
//	txRequest := &protogo.TxRequest{
//		TxId:            txId,
//		ContractName:    ContractName,
//		ContractVersion: "1.0.0",
//		Method:          "init_contract",
//		ByteCode:        ContractBin,
//		Parameters:      params,
//	}
//	txPayload, _ := proto.Marshal(txRequest)
//
//	cdmMessage := &protogo.CDMMessage{
//		TxId:    txId,
//		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
//		Payload: txPayload,
//	}
//	return cdmMessage
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
//	t.Run("TestNewClientConn", TestNewClientConn)
//	t.Run("TestStartClient", TestStartClient)
//	t.Run("TestCDMClient_RegisterAndDeleteRecvChan", TestCDMClient_RegisterAndDeleteRecvChan)
//}
