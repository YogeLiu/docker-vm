// nolint:unused, structcheck

package test

import (
	"io/ioutil"
	"sync"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	vmPb "chainmaker.org/chainmaker/pb-go/v2/vm"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
	"chainmaker.org/chainmaker/vm/v2"

	"github.com/docker/distribution/uuid"
)

var testOrgId = "wx-org1.chainmaker.org"

// CertFilePath is the test cert file path
var CertFilePath = "./testdata/admin1.sing.crt"

var file []byte

// TxContextMockTest mock tx context test
// nolint: unused, struct check
type TxContextMockTest struct {
	gasRemaining  uint64
	lock          *sync.Mutex
	vmManager     protocol.VmManager
	currentDepth  int
	currentResult []byte
	hisResult     []*callContractResult

	sender   *acPb.Member
	creator  *acPb.Member
	CacheMap map[string][]byte
}

// SubtractGas subtract gas from txSimContext
func (s *TxContextMockTest) SubtractGas(gasUsed uint64) error {
	s.gasRemaining -= gasUsed
	return nil
}

// GetGasRemaining return gas remaining in txSimContext
func (s *TxContextMockTest) GetGasRemaining() uint64 {
	return s.gasRemaining
}

// GetSnapshot implement me
func (s *TxContextMockTest) GetSnapshot() protocol.Snapshot {
	//TODO implement me
	panic("implement me")
}

// GetBlockFingerprint returns unique id for block
func (s *TxContextMockTest) GetBlockFingerprint() string {
	return s.GetTx().GetPayload().GetTxId()
}

// GetStrAddrFromPbMember calculate string address from pb Member
func (s *TxContextMockTest) GetStrAddrFromPbMember(pbMember *acPb.Member) (string, error) {
	//TODO implement me
	panic("implement me")
}

// GetNoRecord read data from state, but not record into read set, only used for framework
func (s *TxContextMockTest) GetNoRecord(contractName string, key []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

// GetTxRWMapByContractName get the read-write map of the specified contract of the current transaction
func (s *TxContextMockTest) GetTxRWMapByContractName(
	contractName string,
) (map[string]*commonPb.TxRead, map[string]*commonPb.TxWrite) {
	//TODO implement me
	panic("implement me")
}

// GetCrossInfo get contract call link information
func (s *TxContextMockTest) GetCrossInfo() uint64 {
	crossInfo := vm.NewCallContractContext(0)
	crossInfo.AddLayer(commonPb.RuntimeType_GO)
	return crossInfo.GetCtxBitmap()
}

// HasUsed judge whether the specified common.RuntimeType has appeared in the previous depth
// in the current cross-link
func (s *TxContextMockTest) HasUsed(runtimeType commonPb.RuntimeType) bool {
	//TODO implement me
	panic("implement me")
}

// RecordRuntimeTypeIntoCrossInfo add new vm runtime
func (s *TxContextMockTest) RecordRuntimeTypeIntoCrossInfo(runtimeType commonPb.RuntimeType) {
	//TODO implement me
	panic("implement me")
}

// RemoveRuntimeTypeFromCrossInfo remove runtime from cross info
func (s *TxContextMockTest) RemoveRuntimeTypeFromCrossInfo() {
	//TODO implement me
	panic("implement me")
}

// GetBlockTimestamp returns block timestamp
func (s *TxContextMockTest) GetBlockTimestamp() int64 {
	//TODO implement me
	panic("implement me")
}

// PutRecord put record
func (s *TxContextMockTest) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
	//TODO implement me
	panic("implement me")
}

// PutIntoReadSet put into read set
func (s *TxContextMockTest) PutIntoReadSet(contractName string, key []byte, value []byte) {
	panic("implement me")
}

// GetHistoryIterForKey returns history iter for key
func (s *TxContextMockTest) GetHistoryIterForKey(contractName string, key []byte) (protocol.KeyHistoryIterator,
	error) {
	panic("implement me")
}

// CallContract cross contract call, return (contract result, gas used)
func (s *TxContextMockTest) CallContract(caller, contract *commonPb.Contract, method string, byteCode []byte,
	parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult,
	protocol.ExecOrderTxType, commonPb.TxStatusCode) {
	panic("implement me")
}

// SetIterHandle returns iter
func (s *TxContextMockTest) SetIterHandle(index int32, iter interface{}) {
	panic("implement me")
}

// GetIterHandle returns iter
func (s *TxContextMockTest) GetIterHandle(index int32) (interface{}, bool) {
	panic("implement me")
}

// GetKeys  GetKeys
func (s *TxContextMockTest) GetKeys(keys []*vmPb.BatchKey) ([]*vmPb.BatchKey, error) {
	panic("implement me")
}

// InitContextTest initialize TxContext and Contract
func InitContextTest() *TxContextMockTest {

	if file == nil {
		var err error
		file, err = ioutil.ReadFile(CertFilePath)
		if err != nil {
			panic("file is nil" + err.Error())
		}
	}
	sender := &acPb.Member{
		OrgId:      testOrgId,
		MemberInfo: file,
		//IsFullCert: true,
	}

	txContext := TxContextMockTest{
		lock:         &sync.Mutex{},
		vmManager:    nil,
		hisResult:    make([]*callContractResult, 0),
		creator:      sender,
		sender:       sender,
		CacheMap:     make(map[string][]byte),
		gasRemaining: uint64(10_000_000),
	}

	return &txContext
}

// GetContractByName returns contract name
func (s *TxContextMockTest) GetContractByName(name string) (*commonPb.Contract, error) {
	return utils.GetContractByName(s.Get, name)
}

// GetContractBytecode returns contract bytecode
func (s *TxContextMockTest) GetContractBytecode(name string) ([]byte, error) {
	return utils.GetContractBytecode(s.Get, name)
}

// GetLastChainConfig returns last chain config
func (s *TxContextMockTest) GetLastChainConfig() *configPb.ChainConfig {
	return &configPb.ChainConfig{
		AccountConfig: &configPb.GasAccountConfig{
			EnableGas:       true,
			DefaultGasPrice: float32(0.1),
			DefaultGas:      uint64(11),
			InstallGasPrice: float32(0.2),
			InstallBaseGas:  uint64(12),
		},
	}
}

// GetBlockVersion returns block version
func (s *TxContextMockTest) GetBlockVersion() uint32 {
	return uint32(2030102)
}

// SetStateKvHandle set state kv handle
func (s *TxContextMockTest) SetStateKvHandle(i int32, iterator protocol.StateIterator) {
	panic("implement me")
}

// GetStateKvHandle returns state kv handle
func (s *TxContextMockTest) GetStateKvHandle(i int32) (protocol.StateIterator, bool) {
	panic("implement me")
}

//func (s *TxContextMockTest) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
//	panic("implement me")
//}

// Select range query for key [start, limit)
func (s *TxContextMockTest) Select(name string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
	panic("implement me")
}

// GetBlockProposer returns block proposer
func (s *TxContextMockTest) GetBlockProposer() *acPb.Member {
	panic("implement me")
}

// SetStateSqlHandle set state sql
func (s *TxContextMockTest) SetStateSqlHandle(i int32, rows protocol.SqlRows) {
	panic("implement me")
}

// GetStateSqlHandle returns get state sql
func (s *TxContextMockTest) GetStateSqlHandle(i int32) (protocol.SqlRows, bool) {
	panic("implement me")
}

// nolint: unused, structcheck
type callContractResult struct {
	contractName string
	method       string
	param        map[string][]byte
	deep         int
	gasUsed      uint64
	result       []byte
}

// Get returns key from cache, record this operation to read set
func (s *TxContextMockTest) Get(name string, key []byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	k := string(key)
	if name != "" {
		k = name + "::" + k
	}

	value := s.CacheMap[k]
	//fmt.Printf("[get] key: %s, len of value is %d\n", k, len(value))
	return value, nil
}

// Put key into cache
func (s *TxContextMockTest) Put(name string, key []byte, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	k := string(key)
	//v := string(value)
	if name != "" {
		k = name + "::" + k
	}
	//fmt.Printf("[put] key is %s, len of value is: %d\n", key, len(value))
	s.CacheMap[k] = value
	return nil
}

// Del delete key from cache
func (s *TxContextMockTest) Del(name string, key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	k := string(key)
	//v := string(value)
	if name != "" {
		k = name + "::" + k
	}
	//println("【put】 key:" + k)
	s.CacheMap[k] = nil
	return nil
}

// GetCurrentResult returns current result
func (s *TxContextMockTest) GetCurrentResult() []byte {
	return s.currentResult
}

// GetTx returns tx
func (s *TxContextMockTest) GetTx() *commonPb.Transaction {
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
	return tx
}

// GetBlockHeight returns block height
func (*TxContextMockTest) GetBlockHeight() uint64 {
	return 0
}

// GetTxResult returns tx result
func (s *TxContextMockTest) GetTxResult() *commonPb.Result {
	panic("implement me")
}

// SetTxResult set tx result
func (s *TxContextMockTest) SetTxResult(txResult *commonPb.Result) {
	panic("implement me")
}

// GetTxRWSet returns tx rwset
func (TxContextMockTest) GetTxRWSet(runVmSuccess bool) *commonPb.TxRWSet {
	return &commonPb.TxRWSet{
		TxId:     "txId",
		TxReads:  nil,
		TxWrites: nil,
	}
}

// GetCreator returns creator
func (s *TxContextMockTest) GetCreator(namespace string) *acPb.Member {
	return s.creator
}

// GetSender returns sender
func (s *TxContextMockTest) GetSender() *acPb.Member {
	return s.sender
}

// GetBlockchainStore returns related blockchain store
func (*TxContextMockTest) GetBlockchainStore() protocol.BlockchainStore {
	panic("implement me")
}

// GetBlockchainStore returns blockchain store
//func (*TxContextMockTest) GetBlockchainStore() protocol.BlockchainStore {
//	return nil
//}

// GetAccessControl returns access control
func (*TxContextMockTest) GetAccessControl() (protocol.AccessControlProvider, error) {
	panic("implement me")
}

// GetChainNodesInfoProvider returns chain nodes info provider
func (s *TxContextMockTest) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
	panic("implement me")
}

// GetTxExecSeq returns tx exec seq
func (*TxContextMockTest) GetTxExecSeq() int {
	panic("implement me")
}

// SetTxExecSeq set tx exec seq
func (*TxContextMockTest) SetTxExecSeq(i int) {
	panic("implement me")
}

// GetDepth returns cross contract call depth
func (s *TxContextMockTest) GetDepth() int {
	return s.currentDepth
}
