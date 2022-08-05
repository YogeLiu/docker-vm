/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/sortedmap"
	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/protocol/v2/mock"
	"chainmaker.org/chainmaker/protocol/v2/test"
	"chainmaker.org/chainmaker/utils/v2"
	dockergo "chainmaker.org/chainmaker/vm-engine/v2"
	"chainmaker.org/chainmaker/vm/v2"
	"github.com/docker/distribution/uuid"
	"github.com/golang/mock/gomock"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	testVMLogName = "vm"

	initMethod   = "init_contract"
	invokeMethod = "invoke_contract"

	// ContractNameTest is test contract name
	// ContractNameTest = "contract_test09"
	ContractNameTest = "contract_test100"
	// ContractNameAddr contract addr
	ContractNameAddr = "xxxxxxaddressfehis"

	// ContractVersionTest is test contract version
	ContractVersionTest = "v1.0.0"

	constructKeySeparator = "#"

	chainId = "chain1"

	txType = commonPb.TxType_INVOKE_CONTRACT

	keyHistoryPrefix = "k"
	splitChar        = "#"

	pkPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoEcz1UBgi0DQgAESkwkzwN7DHoCfmNLmUpf280PqnGM
6QU+P3X8uahlUjpgWv+Stfmeco9RqSTU8Y1YGcQvm2Jr327qkRlG7+dELQ==
-----END PUBLIC KEY-----`
	zxlPKAddress = "ZXaaa6f45415493ffb832ca28faa14bef5c357f5f0"
	cmPKAddress  = "4cd0b5e8f6d6df38ecdc06c7431a48dd0265cb1e"

	certPEM = `-----BEGIN CERTIFICATE-----
MIICzjCCAi+gAwIBAgIDCzLUMAoGCCqGSM49BAMCMGoxCzAJBgNVBAYTAkNOMRAw
DgYDVQQIEwdCZWlqaW5nMRAwDgYDVQQHEwdCZWlqaW5nMRAwDgYDVQQKEwd3eC1v
cmcxMRAwDgYDVQQLEwdyb290LWNhMRMwEQYDVQQDEwp3eC1vcmcxLWNhMB4XDTIw
MTAyOTEzMzgxMFoXDTMwMTAyNzEzMzgxMFowcDELMAkGA1UEBhMCQ04xEDAOBgNV
BAgTB0JlaWppbmcxEDAOBgNVBAcTB0JlaWppbmcxEDAOBgNVBAoTB3d4LW9yZzEx
EzARBgNVBAsTCkNoYWluTWFrZXIxFjAUBgNVBAMTDXVzZXIxLnd4LW9yZzEwgZsw
EAYHKoZIzj0CAQYFK4EEACMDgYYABAGLEJZriYzK9Se/vMGfkwjhU55eEZsM2iKM
emSZICh/HY37uR0BFAVUjMYEj84tJBzEEzlpD+AUAe44/b11b+GCMwDXPKcsjHK0
jsAPrN5LH7uptXsjMFpN2bbOqvj6sAIDfTV9chuF91LxCjYnh+Lya0ikextGkpbp
HOvi5eQ/yUHSQaN7MHkwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
KQYDVR0OBCIEIAp+6tWmoiE0KmdtpLFBZpBj1Ni7JH8g2XPgoQwhQS8qMCsGA1Ud
IwQkMCKAIMsnP+UWEyGuyEHBn7JkJzb+tfBqsRCBUIPyMZH4h1HPMAoGCCqGSM49
BAMCA4GMADCBiAJCAIENc8ip2BP4yJpj9SdR9pvZc4/qbBzKucZQaD/GT2sj0FxH
hp8YLjSflgw1+uWlMb/WCY60MyxZr/RRsTYpHu7FAkIBSMAVxw5RYySsf4J3bpM0
CpIO2ZrxkJ1Nm/FKZzMLQjp7Dm//xEMkpCbqqC6koOkRP2MKGSnEGXGfRr1QgBvr
8H8=
-----END CERTIFICATE-----`
	zxlCertAddressFromCert = "ZX0787b8affa4cbdb9994548010c80d9741113ae78"
	cmCertAddressFromCert  = "305f98514f3c2f6fcaeb8247ed147bacf99990f8"

	blockFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

var (
	mockDockerManager *dockergo.InstancesManager
	iteratorWSets     map[string]*commonPb.TxWrite
	keyHistoryData    map[string]*store.KeyModification
	kvSetIndex        int32
	//kvGetIndex    int32
	kvRowCache = make(map[int32]interface{})

	//blockVersionCounter int32
	senderCounter      int32
	chainConfigCounter int32
)

var tmpSimContextMap map[string][]byte

func resetIterCacheAndIndex() {
	kvSetIndex = 0
	kvRowCache = make(map[int32]interface{})
}

func initContractId(runtimeType commonPb.RuntimeType) *commonPb.Contract {
	return &commonPb.Contract{
		Name:        ContractNameTest,
		Version:     ContractVersionTest,
		RuntimeType: runtimeType,
	}
}

func newMockHoleLogger(ctrl *gomock.Controller, name string) protocol.Logger {
	return &test.HoleLogger{}
}

// nolint: deadcode,unused
func newMockTestLogger(ctrl *gomock.Controller, name string) protocol.Logger {
	return &GoLogger{}
}

func initMockSimContext(t *testing.T) (*mock.MockTxSimContext, *gomock.Controller) {
	ctrl := gomock.NewController(t)
	simContext := mock.NewMockTxSimContext(ctrl)

	tmpSimContextMap = make(map[string][]byte)
	txId := uuid.Generate().String()

	simContext.EXPECT().GetTx().DoAndReturn(
		func() *commonPb.Transaction {
			tx := &commonPb.Transaction{
				Payload: &commonPb.Payload{
					ChainId:        chainId,
					TxType:         txType,
					TxId:           txId,
					Timestamp:      0,
					ExpirationTime: 0,
				},
				Result: nil,
			}
			return tx
		}).AnyTimes()

	mockPutIntoReadSet(simContext)
	simContext.EXPECT().GetBlockFingerprint().Return(blockFingerprint).AnyTimes()

	return simContext, ctrl

}

var (
	normalCrossInfoOnce sync.Once
	normalCrossInfo     = vm.NewCallContractContext(0)
	crossCallCrossInfo  = vm.NewCallContractContext(0)
)

func mockNormalGetrossInfo(simContext *mock.MockTxSimContext) {
	normalCrossInfoOnce.Do(
		func() {
			normalCrossInfo.AddLayer(commonPb.RuntimeType_GO)
		},
	)

	simContext.EXPECT().GetCrossInfo().DoAndReturn(
		func() uint64 {
			return normalCrossInfo.GetCtxBitmap()
		},
	).AnyTimes()
}

func mockCrossCallGetCrossInfo(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().GetCrossInfo().DoAndReturn(
		func() uint64 {
			crossCallCrossInfo.AddLayer(commonPb.RuntimeType_GO)
			return crossCallCrossInfo.GetCtxBitmap()
		},
	).AnyTimes()
}

func mockNormalGetDepth(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().GetDepth().DoAndReturn(
		func() int {
			return 0
		},
	).AnyTimes()
}

func mockCrossCallGetDepth(simContext *mock.MockTxSimContext) {
	var depth int
	simContext.EXPECT().GetDepth().DoAndReturn(
		func() int {
			defer func() { depth++ }()
			return depth
		},
	).AnyTimes()
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

//func mockPutIntoReadSet(simContext *mock.MockTxSimContext, name string, key, value []byte) {
func mockPutIntoReadSet(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().PutIntoReadSet(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(name string, key, value []byte) {
			final := name + "::" + string(key)
			tmpSimContextMap[final] = value
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
	return cmConfig.VMConfig.Go, nil
}

// ========== Mock Kv Iterator ==========

/*
	| Key   | Field   | Value |
	| ---   | ---     | ---   |
	| key1  | field1  | val   |
	| key1  | field2  | val   |
	| key1  | field23 | val   |
	| ey1   | field3  | val   |
	| key2  | field1  | val   |
	| key3  | field2  | val   |
	| key33 | field2  | val   |
	| key4  | field3  | val   |
*/
func makeStringKeyMap() (map[string]*commonPb.TxWrite, []*store.KV) {
	stringKeyMap := make(map[string]*commonPb.TxWrite)
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
		stringKeyMap[constructKey(kv.ContractName, kv.Key)] = &commonPb.TxWrite{
			Key:          kv.Key,
			Value:        kv.Value,
			ContractName: kv.ContractName,
		}
	}
	return stringKeyMap, kvs
}

// TxIds is a list of tx ids
var TxIds = []string{
	uuid.Generate().String(),
	uuid.Generate().String(),
	uuid.Generate().String(),
	uuid.Generate().String(),
	uuid.Generate().String(),
	uuid.Generate().String(),
	uuid.Generate().String(),
}

// k+ContractName+StateKey+BlockHeight+TxId
func constructHistoryKey(contractName string, key []byte, blockHeight uint64, txId string) []byte {
	dbkey := fmt.Sprintf(keyHistoryPrefix+"%s"+splitChar+"%s"+splitChar+"%d"+splitChar+"%s",
		contractName, key, blockHeight, txId)
	return []byte(dbkey)
}
func constructHistoryKeyPrefix(contractName string, key []byte) []byte {
	dbkey := fmt.Sprintf(keyHistoryPrefix+"%s"+splitChar+"%s"+splitChar, contractName, key)
	return []byte(dbkey)
}

func makeKeyModificationMap() map[string]*store.KeyModification {
	const testKey = "key1"
	const testField = "field1"
	key := protocol.GetKeyStr(testKey, testField)

	timestamp := time.Now().Unix()
	keyModifications := []*store.KeyModification{
		{
			TxId:        TxIds[0],
			Value:       []byte("value1-1"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 2,
		},
		{
			TxId:        TxIds[1],
			Value:       []byte("value1-2"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 3,
		},
		{
			TxId:        TxIds[2],
			Value:       []byte("value1-3"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 4,
		},
		{
			TxId:        TxIds[3],
			Value:       []byte("value1-4"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 5,
		},
		{
			TxId:        TxIds[4],
			Value:       []byte("value1-5"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 6,
		},
		{
			TxId:        TxIds[5],
			Value:       []byte("value1-6"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 7,
		},
		{
			TxId:        TxIds[6],
			Value:       []byte("value1-7"),
			Timestamp:   timestamp,
			IsDelete:    false,
			BlockHeight: 8,
		},
	}

	keyModificationMap := map[string]*store.KeyModification{
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[0].BlockHeight,
				keyModifications[0].TxId),
		): keyModifications[0],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[1].BlockHeight,
				keyModifications[1].TxId),
		): keyModifications[1],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[2].BlockHeight,
				keyModifications[2].TxId),
		): keyModifications[2],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[3].BlockHeight,
				keyModifications[3].TxId),
		): keyModifications[3],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[4].BlockHeight,
				keyModifications[4].TxId),
		): keyModifications[4],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[5].BlockHeight,
				keyModifications[5].TxId),
		): keyModifications[5],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[5].BlockHeight,
				keyModifications[5].TxId),
		): keyModifications[5],
		string(
			constructHistoryKey(
				ContractNameTest,
				key,
				keyModifications[6].BlockHeight,
				keyModifications[6].TxId),
		): keyModifications[6],
	}
	return keyModificationMap
}

func constructKey(contractName string, key []byte) string {
	return contractName + constructKeySeparator + string(key)
}

func mockGetStateKvHandle(simContext *mock.MockTxSimContext, iteratorIndex int32) {
	simContext.EXPECT().GetIterHandle(gomock.Eq(iteratorIndex)).DoAndReturn(
		func(iteratorIndex int32) (protocol.StateIterator, bool) {
			iterator, ok := kvRowCache[iteratorIndex]
			if !ok {
				return nil, false
			}

			kvIterator, ok := iterator.(protocol.StateIterator)
			if !ok {
				return nil, false
			}
			return kvIterator, true
		},
	).AnyTimes()
}

func mockGetKeyHistoryKVHandle(simContext *mock.MockTxSimContext, iteratorIndex int32) {
	simContext.EXPECT().GetIterHandle(gomock.Eq(iteratorIndex)).DoAndReturn(
		func(iteratorIndex int32) (protocol.KeyHistoryIterator, bool) {
			iterator, ok := kvRowCache[iteratorIndex]
			if !ok {
				return nil, false
			}

			keyHistoryKvIter, ok := iterator.(protocol.KeyHistoryIterator)
			if !ok {
				return nil, false
			}

			return keyHistoryKvIter, true
		},
	).AnyTimes()
}

func mockGetHistoryIterForKey(simContext *mock.MockTxSimContext, contractName string, key []byte) {
	simContext.EXPECT().GetHistoryIterForKey(contractName, key).DoAndReturn(
		mockTxSimContextGetHistoryIterForKey,
	).AnyTimes()
}

func mockTxSimContextGetHistoryIterForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	// 1. 构造迭代器
	historyMap := make(map[string]interface{})
	prefixKey := constructHistoryKeyPrefix(contractName, key)

	for historyDataKey, historyData := range keyHistoryData {
		splitHistoryKey := strings.Split(historyDataKey, splitChar)
		historyContractName := splitHistoryKey[0]
		historyKey := splitHistoryKey[1]
		historyField := splitHistoryKey[2]
		if strings.EqualFold(
			string(prefixKey),
			string(constructHistoryKeyPrefix(historyContractName, protocol.GetKeyStr(historyKey, historyField))),
		) {
			historyMap[historyDataKey] = historyData
		}
	}

	iter := mockNewHistoryIteratorForKey(historyMap)
	return iter, nil
}

func mockNewHistoryIteratorForKey(historyMap map[string]interface{}) protocol.KeyHistoryIterator {
	return &mockHistoryKeyIterator{
		stringKeySortedMap: sortedmap.NewStringKeySortedMapWithInterfaceData(historyMap),
	}
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

type mockHistoryKeyIterator struct {
	stringKeySortedMap *sortedmap.StringKeySortedMap
}

func (iter *mockHistoryKeyIterator) Next() bool {
	return iter.stringKeySortedMap.Length() > 0
}

func (iter *mockHistoryKeyIterator) Value() (*store.KeyModification, error) {
	var km *store.KeyModification
	var keyStr string
	ok := true
	iter.stringKeySortedMap.Range(func(key string, val interface{}) (isContinue bool) {
		keyStr = key
		km, ok = val.(*store.KeyModification)
		return false

	})

	if !ok {
		return nil, errors.New("get value from historyIterator failed, value type error")
	}

	iter.stringKeySortedMap.Remove(keyStr)
	return km, nil
}

func (iter *mockHistoryKeyIterator) Release() {}

func mockGetBlockVersion(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().GetBlockVersion().DoAndReturn(
		GetBlockVersion,
	).AnyTimes()
}

// GetBlockVersion get block version
func GetBlockVersion() uint32 {
	return uint32(2300)
}

// 获取sender公钥
func mockGetSender(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().GetSender().DoAndReturn(
		mockTxSimContextGetSender,
	).AnyTimes()
}

func mockTxSimContextGetSender() *accesscontrol.Member {
	atomic.AddInt32(&senderCounter, 1)
	switch senderCounter % 2 {
	case 1:
		return &accesscontrol.Member{
			OrgId:      chainId,
			MemberType: accesscontrol.MemberType_CERT,
			MemberInfo: []byte(certPEM),
		}
	// case 2:
	// 	return &accesscontrol.Member{
	// 		OrgId:      chainId,
	// 		MemberType: accesscontrol.MemberType_CERT_HASH,
	// 		MemberInfo: nil,
	// 	}
	case 0:
		return &accesscontrol.Member{
			OrgId:      chainId,
			MemberType: accesscontrol.MemberType_PUBLIC_KEY,
			MemberInfo: []byte(pkPEM),
		}
	// case 0:
	// 	return &accesscontrol.Member{
	// 		OrgId:      chainId,
	// 		MemberType: accesscontrol.MemberType_ALIAS,
	// 		MemberInfo: nil,
	// 	}
	default:
		return nil
	}
}

func mockTxQueryCertFromChain(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().Get(syscontract.SystemContract_CERT_MANAGE.String(), gomock.Any()).DoAndReturn(
		mockQueryCert,
	).AnyTimes()
}

func mockQueryCert(name string, nothing interface{}) ([]byte, error) {
	return []byte(certPEM), nil
}

//func mockGetDepth(simContext *mock.MockTxSimContext) {
//	simContext.EXPECT().GetDepth().DoAndReturn(
//		getDepth,
//	).AnyTimes()
//}
//
//func getDepth() int {
//	return 0
//}

func mockGetLastChainConfig(simContext *mock.MockTxSimContext, ctrl *gomock.Controller) {

	blockchainStore := mock.NewMockBlockchainStore(ctrl)
	blockchainStore.EXPECT().GetLastChainConfig().DoAndReturn(mockLastChainConfig).AnyTimes()

	simContext.EXPECT().GetBlockchainStore().Return(blockchainStore).AnyTimes()
}

func mockLastChainConfig() (*configPb.ChainConfig, error) {
	switch chainConfigCounter % 4 {
	case 1, 2:
		chainConfig := &configPb.ChainConfig{
			AccountConfig: &configPb.GasAccountConfig{
				DefaultGas: 10000,
			},
			Vm: &configPb.Vm{
				AddrType: configPb.AddrType_ZXL,
			},
		}
		return chainConfig, nil
	case 3, 0:
		chainConfig := &configPb.ChainConfig{
			AccountConfig: &configPb.GasAccountConfig{
				DefaultGas: 10000,
			},
			Vm: &configPb.Vm{
				AddrType: configPb.AddrType_CHAINMAKER,
			},
		}
		return chainConfig, nil
	}

	return nil, nil
}

// 获取链配置，读取地址格式
// nolint: deadcode, unused
func mockTxGetChainConf_back(simContext *mock.MockTxSimContext) {
	simContext.EXPECT().Get(
		syscontract.SystemContract_CHAIN_CONFIG.String(),
		[]byte(syscontract.SystemContract_CHAIN_CONFIG.String()),
	).DoAndReturn(
		mockGetChainConf,
	).AnyTimes()
}

// nolint: unused
func mockGetChainConf(name string, key []byte) ([]byte, error) {
	atomic.AddInt32(&chainConfigCounter, 1)

	switch chainConfigCounter % 8 {
	case 1, 2, 3, 4:
		zxConfig := configPb.ChainConfig{
			Vm: &configPb.Vm{
				AddrType: configPb.AddrType_ZXL,
			},
			Crypto: &configPb.CryptoConfig{
				Hash: "SHA256",
			},
		}

		bytes, err := zxConfig.Marshal()
		if err != nil {
			return nil, err
		}
		return bytes, nil
	case 5, 6, 7, 0:
		ethConfig := configPb.ChainConfig{
			Vm: &configPb.Vm{
				AddrType: configPb.AddrType_CHAINMAKER,
			},
			Crypto: &configPb.CryptoConfig{
				Hash: "SHA256",
			},
		}
		bytes, err := ethConfig.Marshal()
		if err != nil {
			return nil, err
		}
		return bytes, nil
	default:
		return nil, nil
	}
}

func mockGetStrAddrFromPbMember(s *mock.MockTxSimContext) {
	s.EXPECT().GetStrAddrFromPbMember(gomock.Any()).DoAndReturn(mockGetStrAddr).AnyTimes()
}

func mockGetStrAddr(pbMember *accesscontrol.Member) (string, error) {
	atomic.AddInt32(&chainConfigCounter, 1)

	switch chainConfigCounter % 4 {
	case 1, 2:
		return utils.GetStrAddrFromPbMember(pbMember, configPb.AddrType_ZXL, crypto.HASH_TYPE_SHA256)
	case 3, 0:
		return utils.GetStrAddrFromPbMember(pbMember, configPb.AddrType_CHAINMAKER, crypto.HASH_TYPE_SHA256)
	}

	return "", nil
}

func mockCallContract(simContext *mock.MockTxSimContext, param map[string][]byte) {
	simContext.EXPECT().CallContract(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).DoAndReturn(
		func(contract *commonPb.Contract,
			method string, byteCode []byte,
			parameter map[string][]byte,
			gasUsed uint64,
			refTxType commonPb.TxType) (*commonPb.ContractResult, protocol.ExecOrderTxType, commonPb.TxStatusCode) {

			mockLogger := newMockHoleLogger(nil, testVMLogName)
			callContractRuntimeInstance, _ := mockDockerManager.NewRuntimeInstance(nil, chainId, "",
				"", nil, nil, mockLogger)

			param["method"] = param["contract_method"]
			runtimeContractResult, specialTxType := callContractRuntimeInstance.Invoke(
				&commonPb.Contract{
					Name:        ContractNameTest,
					Version:     ContractVersionTest,
					RuntimeType: commonPb.RuntimeType_GO,
				},
				method,
				nil,
				param,
				simContext,
				uint64(123),
			)
			code := commonPb.TxStatusCode_CONTRACT_FAIL
			if runtimeContractResult.Code == 0 {
				code = commonPb.TxStatusCode_SUCCESS
			}

			return runtimeContractResult, specialTxType, code
		},
	).AnyTimes()
}

// nolint: deadcode,unused
func callContract(
	simContext *mock.MockTxSimContext,
	param map[string][]byte,
) (*commonPb.ContractResult, protocol.ExecOrderTxType, commonPb.TxStatusCode) {

	mockLogger := newMockHoleLogger(nil, testVMLogName)
	callContractRuntimeInstance, _ := mockDockerManager.NewRuntimeInstance(nil, chainId, "",
		"", nil, nil, mockLogger)

	runtimeContractResult, specialTxType := callContractRuntimeInstance.Invoke(
		&commonPb.Contract{
			Name:        ContractNameTest,
			Version:     ContractVersionTest,
			RuntimeType: commonPb.RuntimeType_GO,
		},
		invokeMethod,
		nil,
		param,
		simContext,
		uint64(123),
	)
	code := commonPb.TxStatusCode_CONTRACT_FAIL
	if runtimeContractResult.Code == 0 {
		code = commonPb.TxStatusCode_SUCCESS
	}

	return runtimeContractResult, specialTxType, code
}

// GoLogger is a golang system log implementation of protocol.Logger, it's for unit test
type GoLogger struct{}

// Debug is the debug log
func (GoLogger) Debug(args ...interface{}) {
	log.Printf("DEBUG: %v", args)
}

// Debugf is the debugf log
func (GoLogger) Debugf(format string, args ...interface{}) {
	log.Printf("DEBUG: "+format, args...)
}

// Debugw is the debugw log
func (GoLogger) Debugw(msg string, keysAndValues ...interface{}) {
	log.Printf("DEBUG: "+msg+" %v", keysAndValues...)
}

// Error is the error log
func (GoLogger) Error(args ...interface{}) {
	log.Printf("ERROR: %v", args)
}

// Errorf is the errorf log
func (GoLogger) Errorf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	log.Printf("ERROR: " + str + "")
}

// Errorw is the errorw log
func (GoLogger) Errorw(msg string, keysAndValues ...interface{}) {
	log.Printf("ERROR: "+msg+" %v", keysAndValues...)
}

// Fatal is the fatal log
func (GoLogger) Fatal(args ...interface{}) {
	log.Fatal(args...)
}

// Fatalf is the fatalf log
func (GoLogger) Fatalf(format string, args ...interface{}) {
	//log.Fatalf(format, args...)
}

// Fatalw is the fatalw log
func (GoLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	//log.Fatalf(msg+" %v", keysAndValues...)
}

// Info is the info log
func (GoLogger) Info(args ...interface{}) {
	log.Printf("INFO: %v", args)
}

// Infof is the infof log
func (GoLogger) Infof(format string, args ...interface{}) {
	log.Printf("INFO: "+format, args...)
}

// Infow is the infow log
func (GoLogger) Infow(msg string, keysAndValues ...interface{}) {
	//log.Printf("INFO: "+msg+" %v", keysAndValues...)
}

// Panic is the panic log
func (GoLogger) Panic(args ...interface{}) {
	log.Panic(args...)
}

// Panicf is the panicf log
func (GoLogger) Panicf(format string, args ...interface{}) {
	log.Panicf(format, args...)
}

// Panicw is the panicw log
func (GoLogger) Panicw(msg string, keysAndValues ...interface{}) {
	log.Panicf(msg+" %v", keysAndValues...)
}

// Warn is the warn log
func (GoLogger) Warn(args ...interface{}) {
	log.Printf("WARN: %v", args)
}

// Warnf is the warnf log
func (GoLogger) Warnf(format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	log.Printf("WARN: " + str + "")
}

// Warnw is the warn log
func (GoLogger) Warnw(msg string, keysAndValues ...interface{}) {
	log.Printf("WARN: "+msg+" %v", keysAndValues...)
}

// DebugDynamic is the dynamic debug log
func (GoLogger) DebugDynamic(l func() string) {
	log.Print("DEBUG:", l())
}

// InfoDynamic is the dynamic info log
func (GoLogger) InfoDynamic(l func() string) {
	log.Print("INFO:", l())
}
