package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
	"time"

	//"chainmaker.org/chainmaker-go/logger"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	vmdockergo "chainmaker.org/chainmaker/vm-docker-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

/*
1 get chainmaker configuration setting from mock file
2 generate a new docker manager
3 start a docker container
4 mock TxSimContext(interaction with chain)
5 mock docker-go RuntimeInstace
6 create a user contract
7 invoke method
*/

const (
	chainId = "chain1"
)

func getMockedCMConfig() (map[string]interface{}, error) {
	// 0. load env
	cmd := &cobra.Command{}
	cmViper := viper.New()
	err := cmViper.BindPFlags(cmd.PersistentFlags())
	if err != nil {
		return nil, err
	}
	// 1. load the path of the config files
	mockedConfigration := "./testdata/chainmaker.yml"
	ymlFile := mockedConfigration
	if !filepath.IsAbs(ymlFile) {
		ymlFile, _ = filepath.Abs(ymlFile)
		mockedConfigration = ymlFile
	}

	// 2. load the config file
	cmViper.SetConfigFile(ymlFile)
	if err := cmViper.ReadInConfig(); err != nil {
		return nil, err
	}

	logConfigFile := cmViper.GetString("log.config_file")
	if logConfigFile != "" {
		cmViper.SetConfigFile(logConfigFile)
		if err := cmViper.MergeInConfig(); err != nil {
			return nil, err
		}
	}
	flagSets := make([]*pflag.FlagSet, 0)
	for _, command := range cmd.Commands() {
		flagSets = append(flagSets, command.PersistentFlags())
		err := cmViper.BindPFlags(command.PersistentFlags())
		if err != nil {
			return nil, err
		}
	}

	// 3. create new CMConfig instance
	cmConfig := &localconf.CMConfig{}
	if err := cmViper.Unmarshal(cmConfig); err != nil {
		return nil, err
	}
	return cmConfig.VMConfig, nil
}

func TestDockerGoInit(t *testing.T) {
	//step1: get chainmaker configuration setting from mocked data
	fmt.Printf("=== step 1 load mocked chainmaker configuration file ===\n")
	cmConfig, err := getMockedCMConfig()
	if err != nil {
		t.Errorf("get the mocked chainmaker configuration failed %v\n", err)
	}

	//step2: generate a docker manager instance
	fmt.Printf("=== step 2 Create docker instance ===\n")

	dockerMangerInstance := vmdockergo.NewDockerManager(chainId, cmConfig)

	//step3: start docker VM
	fmt.Printf("=== step 3 start Docker VM ===\n")
	dockerContainErr := dockerMangerInstance.StartVM()
	if dockerContainErr != nil {
		t.Errorf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	// step4: stop docker vm
	time.Sleep(10 * time.Second)
	dockerContainErr = dockerMangerInstance.StopVM()
	if dockerContainErr != nil {
		t.Errorf("stop docmer manager instance failed %v\n", dockerContainErr)
	}
}

func TestDockerGoContractPipeline(t *testing.T) {
	//step1: get chainmaker configuration setting from mocked data
	fmt.Printf("=== step 1 load mocked chainmaker configuration file ===\n")
	cmConfig, err := getMockedCMConfig()
	if err != nil {
		t.Errorf("get the mocked chainmaker configuration failed %v\n", err)
	}

	//step2: generate a docker manager instance
	fmt.Printf("=== step 2 Create docker instance ===\n")
	dockerMangerInstance := vmdockergo.NewDockerManager(chainId, cmConfig)

	//step3: start docker VM
	fmt.Printf("=== step 3 start Docker VM ===\n")
	dockerContainErr := dockerMangerInstance.StartVM()
	if dockerContainErr != nil {
		t.Errorf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	//step4: mock contractId, txContext and contractBin
	fmt.Printf("======step4 mock contractId and txContext=======\n")
	contractId, txContext := InitContextTest(commonPb.RuntimeType_DOCKER_GO)
	fmt.Printf("the contractId %v\n", contractId.RuntimeType)

	contractBin, contractFileErr := ioutil.ReadFile("./testdata/DockerGoCal.7z")
	if contractFileErr != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", contractFileErr))
	}

	//step5: create new NewRuntimeInstance -- for create user contract
	fmt.Printf("=== step 5 create new runtime instance ===\n")
	loger := logger.GetLogger(logger.MODULE_VM)
	mockedRuntimeInstance, err := dockerMangerInstance.NewRuntimeInstance(nil, chainId, "", "", nil, nil, loger)
	if err != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", err))
	}

	//step6: invoke user contract --- create user contract
	fmt.Printf("=== step 6 init user contract ===\n")
	parameters := make(map[string][]byte)
	parameters["__sender_pk__"] = []byte("c8c088cd9e333950f47cd5f5e3d6ebdadf522459553da90c138ca8ce16549480")
	parameters["__creator_org_id__"] = []byte("wx-org1.chainmaker.org")
	parameters["__creator_role__"] = []byte("CLIENT")
	parameters["__block_height__"] = []byte("1")
	parameters["__sender_org_id__"] = []byte("wx-org1.chainmaker.org")
	parameters["__creator_pk__"] = []byte("c8c088cd9e333950f47cd5f5e3d6ebdadf522459553da90c138ca8ce16549480")
	parameters["__tx_id__"] = []byte("a4f108f00005492bb608b7237b23fac0adfe521fb08b4f86aefb774843a4fc1e")
	parameters["__sender_role__"] = []byte("CLIENT")
	result := mockedRuntimeInstance.Invoke(contractId, "init_contract", contractBin, parameters, txContext, uint64(123))

	if result.GetCode() != 0 {
		t.Errorf("init customer contract failed %v", result.GetMessage())
	}
	if result.GetGasUsed() == 0 {
		t.Errorf("get gas failed %v", result.GetGasUsed())
	}
	fmt.Printf("init customer contract success: %v, and gas used: %v\n", string(result.GetResult()), result.GetGasUsed())

	//step7: create new NewRuntimeInstance -- for invoke method from user contract

	//step8: invoke add method
	fmt.Printf("=== step 8 invoke method--add  ===\n")
	parameters["arg1"] = []byte("10")
	parameters["arg2"] = []byte("20")
	parameters["method"] = []byte("add")
	result = mockedRuntimeInstance.Invoke(contractId, "invoke_contract", contractBin, parameters, txContext, uint64(123))
	if result.GetCode() != 0 {
		t.Errorf("init customer contract failed %v", result.GetMessage())
	}
	if result.GetGasUsed() == 0 {
		t.Errorf("get gas failed %v", result.GetGasUsed())
	}
	fmt.Printf("result as expect: %v, the gas used: %v\n", string(result.GetResult()), result.GetGasUsed())
}
