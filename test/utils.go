package test

import (
	"fmt"
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker/common/v2/sortedmap"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"github.com/docker/distribution/uuid"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	initMethod   = "init_contract"
	invokeMethod = "invoke_contract"

	ContractNameTest    = "contract_fvt"
	ContractVersionTest = "v1.0.0"

	constructKeySeparator = "#"

	chainId = "chain1"

	txType = commonPb.TxType_INVOKE_CONTRACT

	boolTrue  int32 = 1
	boolFalse int32 = 0
)

var (
	iteratorWSets map[string]*common.TxWrite
	kvIndex       int32 = 0 // nolint: deadcode, unused
	kvRowCache          = make(map[int32]protocol.StateIterator)
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

// ========== Mock Kv Iterator ==========

func makeStringKeyMap() (map[string]*common.TxWrite, []*store.KV) {
	stringKeyMap := make(map[string]*common.TxWrite)
	kvs := []*store.KV{
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key1", "field1"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key1", "field2"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key1", "field23"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key1", "field3"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key2", "field1"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key3", "field2"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key33", "field2"),
			Value:        []byte("val"),
		},
		{
			ContractName: ContractNameTest,
			Key:          protocol.GetKeyStr("key4", "field3"),
			Value:        []byte("val"),
		},
	}

	for _, kv := range kvs {
		stringKeyMap[constructKey(kv.ContractName, kv.Key)] = &common.TxWrite{
			Key:          kv.Key,
			Value:        kv.Value,
			ContractName: kv.ContractName,
		}
	}
	return stringKeyMap, kvs
}

func constructKey(contractName string, key []byte) string {
	return contractName + constructKeySeparator + string(key)
}

func mockGetStateKvHandle(simContext *mock.MockTxSimContext, iteratorIndex int32) {
	simContext.EXPECT().GetStateKvHandle(gomock.Eq(iteratorIndex)).DoAndReturn(
		func(iteratorIndex int32) (protocol.StateIterator, bool) {
			iterator, ok := kvRowCache[iteratorIndex]
			if ok {
				return iterator, true
			}
			return nil, false
		},
	).AnyTimes()
}

func mockSelect(simContext *mock.MockTxSimContext, name string, key, value []byte) {
	simContext.EXPECT().Select(name, key, value).DoAndReturn(
		mockTxSimContextSelect,
	).AnyTimes()
}

func mockTxSimContextSelect(contractName string, startKey, limit []byte) (protocol.StateIterator, error) {
	wsetsMap := make(map[string]interface{})

	for _, txWrite := range iteratorWSets {
		if string(txWrite.Key) >= string(startKey) && string(txWrite.Key) < string(limit) {
			wsetsMap[string(txWrite.Key)] = &store.KV{
				Key:          txWrite.Key,
				Value:        txWrite.Value,
				ContractName: contractName,
			}
		}
	}
	wsetIterator := mockNewSimContextIterator(wsetsMap)

	return wsetIterator, nil
}

func mockNewSimContextIterator(wsets map[string]interface{}) protocol.StateIterator {
	return &mockStateIterator{
		stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(wsets),
	}
}

type mockStateIterator struct {
	stringKeySortedMap *sortedmap.StringKeySortedMap
}

func (iter *mockStateIterator) Next() bool {
	return iter.stringKeySortedMap.Length() > 0
}

func (iter *mockStateIterator) Value() (*store.KV, error) {
	var kv *store.KV
	var keyStr string
	ok := true
	// get the first row
	iter.stringKeySortedMap.Range(func(key string, val interface{}) (isContinue bool) {
		keyStr = key
		kv, ok = val.(*store.KV)
		return false
	})
	if !ok {
		return nil, fmt.Errorf("get value from wsetIterator failed, value type error")
	}
	iter.stringKeySortedMap.Remove(keyStr)
	return kv, nil
}

func (iter *mockStateIterator) Release() {}
