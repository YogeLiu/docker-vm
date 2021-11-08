package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"testing"

	docker_go "chainmaker.org/chainmaker/vm-docker-go"
	"github.com/stretchr/testify/assert"

	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
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
	mockDockerManager = docker_go.NewDockerManager(chainId, cmConfig)

	//step3: start docker VM
	fmt.Printf("=== step 3 start Docker VM ===\n")
	dockerContainErr := mockDockerManager.StartVM()
	if dockerContainErr != nil {
		log.Fatalf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	//step4: mock contractId, contractBin
	fmt.Printf("======step4 mock contractId and txContext=======\n")
	mockContractId = initContractId(commonPb.RuntimeType_DOCKER_GO)
	mockTxContext = initMockSimContext(t)
	contractBin, contractFileErr := ioutil.ReadFile("./testdata/contract_test.7z")
	if contractFileErr != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", contractFileErr))
	}

	//step5: create new NewRuntimeInstance -- for create user contract
	fmt.Printf("=== step 5 create new runtime instance ===\n")
	mockLogger := logger.GetLogger(logger.MODULE_VM)
	mockRuntimeInstance, err = mockDockerManager.NewRuntimeInstance(nil, chainId, "",
		"", nil, nil, mockLogger)
	if err != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", err))
	}

	//step6: invoke user contract --- create user contract
	fmt.Printf("=== step 6 init user contract ===\n")
	parameters := generateInitParams()
	result := mockRuntimeInstance.Invoke(mockContractId, initMethod, contractBin, parameters,
		mockTxContext, uint64(123))
	if result.Code == 0 {
		fmt.Printf("deploy user contract successfully\n")
	}
}

func tearDownTest() {
	err := mockDockerManager.StopVM()
	if err != nil {
		log.Fatalf("stop docmer manager instance failed %v\n", err)
	}
}

func TestDockerGoBasicInvoke(t *testing.T) {
	setupTest(t)

	parameters := generateInitParams()
	parameters["method"] = []byte("display")
	result := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil, parameters,
		mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)

	parameters["method"] = []byte("not existed method")
	result = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil, parameters,
		mockTxContext, uint64(123))
	assert.Equal(t, uint32(1), result.Code)
	assert.Equal(t, []byte("unknown method"), result.Result)

	tearDownTest()
}
