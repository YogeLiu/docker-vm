/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"testing"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	docker_go "chainmaker.org/chainmaker/vm-engine/v2"
)

const (
	methodSave = "save"
)

func TestDockerGoMemory(t *testing.T) {

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
		newMockHoleLogger(nil, testVMLogName),
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
	mockContractId = &commonPb.Contract{
		Name:        ContractNameTest,
		Version:     ContractVersionTest,
		RuntimeType: commonPb.RuntimeType_GO,
	}
	mockTxContext = initMockSimContext(t)
	mockGetLastChainConfig(mockTxContext)
	mockNormalGetDepth(mockTxContext)
	mockNormalGetrossInfo(mockTxContext)

	filePath := fmt.Sprintf("./testdata/%s", ContractNameTest)
	contractBin, contractFileErr := ioutil.ReadFile(filePath)
	if contractFileErr != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", contractFileErr))
	}

	//step5: create new NewRuntimeInstance -- for create user contract
	fmt.Printf("=== step 5 create new runtime instance ===\n")
	mockLogger := newMockHoleLogger(nil, testVMLogName)
	mockRuntimeInstance, err = mockDockerManager.NewRuntimeInstance(nil, chainId, "",
		"", nil, nil, mockLogger)
	if err != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", err))
	}

	//step6: invoke user contract --- create user contract
	fmt.Printf("=== step 6 init user contract ===\n")
	parameters := generateInitParams()
	result, _ := mockRuntimeInstance.Invoke(mockContractId, initMethod, contractBin, parameters,
		mockTxContext, uint64(123))
	if result.Code == 0 {
		fmt.Printf("deploy user contract successfully\n")
	}

	//testMultipleTxs(mockLogger)

	fmt.Println("tear down")
	tearDownTest()
}

func testMultipleTxs(mockLogger protocol.Logger) {
	fmt.Println("--------- Ready to analysis --------------")
	time.Sleep(20 * time.Second)
	fmt.Println("---------- Start -------------------------")

	mockTxContext.EXPECT().Put(ContractNameTest, []byte("key"), []byte("name")).Return(nil).AnyTimes()

	loopNum := 2000
	threadNum := 300

	for loopIndex := 0; loopIndex < loopNum; loopIndex++ {

		wg := sync.WaitGroup{}

		for threadIndex := 0; threadIndex < threadNum; threadIndex++ {

			wg.Add(1)

			go func(i int) {

				newRuntimeInstance, _ := mockDockerManager.NewRuntimeInstance(nil, chainId, "",
					"", nil, nil, mockLogger)

				newContractId := &commonPb.Contract{
					Name:        ContractNameTest,
					Version:     ContractVersionTest,
					RuntimeType: commonPb.RuntimeType_GO,
				}

				parameters := generateInitParams()
				parameters["file_key"] = []byte("key")
				parameters["file_name"] = []byte("name")
				method := methodSave

				newRuntimeInstance.Invoke(newContractId, method, nil, parameters,
					mockTxContext, uint64(123))

				wg.Done()

			}(threadIndex)
		}

		wg.Wait()
		fmt.Printf("finished %d loop, each loop has %d txs\n", loopIndex, threadNum)
	}

	fmt.Println("--------- Finished analysis --------------")
	time.Sleep(30 * time.Second)
}
