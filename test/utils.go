package test

import (
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/docker/distribution/uuid"
	"github.com/golang/mock/gomock"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/localconf/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	initMethod   = "init_contract"
	invokeMethod = "invoke_contract"

	ContractNameTest    = "contract_test"
	ContractVersionTest = "v1.0.0"

	chainId = "chain1"

	txType = commonPb.TxType_INVOKE_CONTRACT
)

var tmpSimContextMap map[string][]byte

func initContractId(runtimeType commonPb.RuntimeType) *commonPb.Contract {
	return &commonPb.Contract{
		Name:        ContractNameTest,
		Version:     ContractVersionTest,
		RuntimeType: runtimeType,
	}
}

func initMockSimContext(t *testing.T) *mock.MockTxSimContext {
	ctrl := gomock.NewController(t)
	simContext := mock.NewMockTxSimContext(ctrl)

	tmpSimContextMap = make(map[string][]byte)

	// getTx
	tx := &commonPb.Transaction{
		Payload: &commonPb.Payload{
			ChainId:        chainId,
			TxType:         txType,
			TxId:           uuid.Generate().String(),
			Timestamp:      0,
			ExpirationTime: 0,
		},
		Result: nil,
	}

	simContext.EXPECT().GetTx().Return(tx).AnyTimes()
	return simContext

}

func mockPut(simContext *mock.MockTxSimContext, name string, key, value []byte) {
	simContext.EXPECT().Put(name, key, value).DoAndReturn(
		func(name string, key, value []byte) error {
			final := name + "::" + string(key)
			tmpSimContextMap[final] = value
			return nil
		},
	).AnyTimes()
}

func generateInitParams() map[string][]byte {
	parameters := make(map[string][]byte)
	parameters["__sender_pk__"] = []byte("c8c088cd9e333950f47cd5f5e3d6ebdadf522459553da90c138ca8ce16549480")
	parameters["__creator_org_id__"] = []byte("wx-org1.chainmaker.org")
	parameters["__creator_role__"] = []byte("CLIENT")
	parameters["__block_height__"] = []byte("1")
	parameters["__sender_org_id__"] = []byte("wx-org1.chainmaker.org")
	parameters["__creator_pk__"] = []byte("c8c088cd9e333950f47cd5f5e3d6ebdadf522459553da90c138ca8ce16549480")
	parameters["__tx_id__"] = []byte("a4f108f00005492bb608b7237b23fac0adfe521fb08b4f86aefb774843a4fc1e")
	parameters["__sender_role__"] = []byte("CLIENT")
	parameters["__tx_time_stamp__"] = []byte("12345")
	return parameters
}

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
	cmViper.SetConfigFile(mockedConfigration)
	if err = cmViper.ReadInConfig(); err != nil {
		return nil, err
	}

	logConfigFile := cmViper.GetString("log.config_file")
	if logConfigFile != "" {
		cmViper.SetConfigFile(logConfigFile)
		if err = cmViper.MergeInConfig(); err != nil {
			return nil, err
		}
	}
	//flagSets := make([]*pflag.FlagSet, 0)
	for _, command := range cmd.Commands() {
		//flagSets = append(flagSets, command.PersistentFlags())
		err = cmViper.BindPFlags(command.PersistentFlags())
		if err != nil {
			return nil, err
		}
	}

	// 3. create new CMConfig instance
	cmConfig := &localconf.CMConfig{}
	if err = cmViper.Unmarshal(cmConfig); err != nil {
		return nil, err
	}
	return cmConfig.VMConfig, nil
}

//func execCommand(command string) {
//	cmd := exec.Command(command)
//
//	stdout, err := cmd.StdoutPipe()
//	if err != nil {
//		fmt.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
//		return
//	}
//
//	if err = cmd.Start(); err != nil {
//		fmt.Println("Error:The command is err,", err)
//		return
//	}
//
//	bytes, err := ioutil.ReadAll(stdout)
//	if err != nil {
//		fmt.Println("ReadAll Stdout:", err.Error())
//		return
//	}
//
//	if err = cmd.Wait(); err != nil {
//		fmt.Println("wait:", err.Error())
//		return
//	}
//
//	fmt.Printf("stdout:\n\n %s", bytes)
//}
