/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	docker_go "chainmaker.org/chainmaker/vm-engine/v2"
	"github.com/stretchr/testify/assert"
)

/*
1 get chainmaker configuration setting from mock file
2 generate a new docker manager
3 start a docker container
4 mock TxSimContext(interaction with chain)
5 mock docker-go RuntimeInstace
6 create a user contract
7 deploy user contract
*/

func setupTest(t *testing.T) {

	//step1: get chainmaker configuration setting from mocked data
	fmt.Printf("=== step 1 load mocked chainmaker configuration file ===\n")
	cmConfig, err := getMockedCMConfig()
	if err != nil {
		log.Fatalf("get the mocked chainmaker configuration failed %v\n", err)
	}

	//step2: generate a docker manager instance
	fmt.Printf("=== step 2 Create docker instance ===\n")
	mockDockerManager = docker_go.NewInstancesManager(
		chainId,
		newMockTestLogger(nil, testVMLogName),
		cmConfig,
	).(*docker_go.InstancesManager)
	mockDockerManager.BlockDurationMgr.AddBlockTxsDuration(blockFingerprint)

	//step3: start docker VM
	fmt.Printf("=== step 3 start Docker VM ===\n")
	dockerContainErr := mockDockerManager.StartVM()
	if dockerContainErr != nil {
		log.Fatalf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	//step4: mock contractId, contractBin
	fmt.Printf("======step4 mock contractId and txContext=======\n")
	mockContractId = initContractId(commonPb.RuntimeType_GO)
	mockTxContext = initMockSimContext(t)
	mockNormalGetDepth(mockTxContext)
	mockNormalGetrossInfo(mockTxContext)

	filePath := fmt.Sprintf("./testdata/%s", ContractNameTest)
	contractBin, contractFileErr := ioutil.ReadFile(filePath)
	if contractFileErr != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", contractFileErr))
	}

	//step5: create new NewRuntimeInstance -- for create user contract
	fmt.Printf("=== step 5 create new runtime instance ===\n")
	mockLogger := newMockTestLogger(nil, testVMLogName)
	mockRuntimeInstance, err = mockDockerManager.NewRuntimeInstance(nil, chainId, "",
		"", nil, nil, mockLogger)
	if err != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", err))
	}

	mockTxContext.EXPECT().GetContractBytecode(gomock.Any()).Return(contractBin, nil).AnyTimes()
	mockGetLastChainConfig(mockTxContext)

	//step6: invoke user contract --- create user contract
	fmt.Printf("=== step 6 init user contract ===\n")
	parameters := generateInitParams()
	result, _ := mockRuntimeInstance.Invoke(mockContractId, initMethod, contractBin, parameters,
		mockTxContext, uint64(123))
	if result.Code == 0 {
		fmt.Printf("deploy user contract successfully\n")
	}
}

func tearDownTest() {
	//err := mockDockerManager.StopVM()
	//if err != nil {
	//	log.Fatalf("stop docmer manager instance failed %v\n", err)
	//}
	time.Sleep(1000 * time.Millisecond)
}

func TestDockerGoBasicInvoke(t *testing.T) {
	setupTest(t)

	parameters := generateInitParams()
	// parameters["method"] = []byte("display")
	method := "display"
	result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil, parameters,
		mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)

	parameters["method"] = []byte("not existed method")
	method = "not existed method"
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil, parameters,
		mockTxContext, uint64(123))
	assert.Equal(t, uint32(1), result.Code)
	assert.Equal(t, []byte("unknown method"), result.Result)
	fmt.Println(result)

	tearDownTest()
}
