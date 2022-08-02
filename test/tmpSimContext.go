// nolint:unused, structcheck

package test

import (
	"io/ioutil"
	"sync"

	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
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
// nolint: unused, structcheck
type TxContextMockTest struct {
	lock          *sync.Mutex
	vmManager     protocol.VmManager
	gasUsed       uint64 // only for callContract
	currentDepth  int
	currentResult []byte
	hisResult     []*callContractResult

	sender   *acPb.Member
	creator  *acPb.Member
	CacheMap map[string][]byte
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

//GetNoRecord read data from state, but not record into read set, only used for framework
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
func (s *TxContextMockTest) CallContract(contract *commonPb.Contract, method string, byteCode []byte,
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
		lock:      &sync.Mutex{},
		vmManager: nil,
		hisResult: make([]*callContractResult, 0),
		creator:   sender,
		sender:    sender,
		CacheMap:  make(map[string][]byte),
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

// GetBlockVersion returns block version
func (s *TxContextMockTest) GetBlockVersion() uint32 {
	panic("implement me")
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
	return &mockBlockchainStore{}
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

// BaseParam is base params for test
func BaseParam(parameters map[string][]byte) {
	parameters[protocol.ContractTxIdParam] = []byte("TX_ID")
	parameters[protocol.ContractCreatorOrgIdParam] = []byte("org_a")
	parameters[protocol.ContractCreatorRoleParam] = []byte("admin")
	parameters[protocol.ContractCreatorPkParam] = []byte("1234567890abcdef1234567890abcdef")
	parameters[protocol.ContractSenderOrgIdParam] = []byte("org_b")
	parameters[protocol.ContractSenderRoleParam] = []byte("user")
	parameters[protocol.ContractSenderPkParam] = []byte("11223344556677889900aabbccddeeff")
	parameters[protocol.ContractBlockHeightParam] = []byte("1")
}

type mockBlockchainStore struct {
}

func (m mockBlockchainStore) CreateDatabase(contractName string) error {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) DropDatabase(contractName string) error {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) GetContractDbName(contractName string) string {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) GetMemberExtraData(member *acPb.Member) (*acPb.MemberExtraData, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) GetTxWithRWSet(txId string) (*commonPb.TransactionWithRWSet, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) GetTxInfoWithRWSet(txId string) (*commonPb.TransactionInfoWithRWSet, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) GetTxWithInfo(txId string) (*commonPb.TransactionInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) TxExistsInFullDB(txId string) (bool, uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) TxExistsInIncrementDB(txId string, startHeight uint64) (bool, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) GetTxInfoOnly(txId string) (*commonPb.TransactionInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (m mockBlockchainStore) ReadObjects(contractName string, keys [][]byte) ([][]byte, error) {
	//TODO implement me
	panic("implement me")
}

// GetContractByName
// @param name
// @return *commonPb.Contract
// @return error
func (m mockBlockchainStore) GetContractByName(name string) (*commonPb.Contract, error) {
	panic("implement me")
}

// GetContractBytecode
// @param name
// @return []byte
// @return error
func (m mockBlockchainStore) GetContractBytecode(name string) ([]byte, error) {
	panic("implement me")
}

// GetHeightByHash
// @param blockHash
// @return uint64
// @return error
func (m mockBlockchainStore) GetHeightByHash(blockHash []byte) (uint64, error) {
	panic("implement me")
}

// GetBlockHeaderByHeight
// @param height
// @return *commonPb.BlockHeader
// @return error
func (m mockBlockchainStore) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
	panic("implement me")
}

// GetLastChainConfig
// @return *configPb.ChainConfig
// @return error
func (m mockBlockchainStore) GetLastChainConfig() (*configPb.ChainConfig, error) {
	return &configPb.ChainConfig{
		AccountConfig: nil,
	}, nil
}

// GetTxHeight
// @param txId
// @return uint64
// @return error
func (m mockBlockchainStore) GetTxHeight(txId string) (uint64, error) {
	panic("implement me")
}

// GetArchivedPivot
// @return uint64
func (m mockBlockchainStore) GetArchivedPivot() uint64 {
	panic("implement me")
}

// ArchiveBlock
// @param archiveHeight
// @return error
func (m mockBlockchainStore) ArchiveBlock(archiveHeight uint64) error {
	panic("implement me")
}

// RestoreBlocks
// @param serializedBlocks
// @return error
func (m mockBlockchainStore) RestoreBlocks(serializedBlocks [][]byte) error {
	panic("implement me")
}

// QuerySingle
// @param contractName
// @param sql
// @param values
// @return protocol.SqlRow
// @return error
func (m mockBlockchainStore) QuerySingle(contractName, sql string, values ...interface{}) (protocol.SqlRow, error) {
	panic("implement me")
}

// QueryMulti
// @param contractName
// @param sql
// @param values
// @return protocol.SqlRows
// @return error
func (m mockBlockchainStore) QueryMulti(contractName, sql string, values ...interface{}) (protocol.SqlRows, error) {
	panic("implement me")
}

// ExecDdlSql
// @param contractName
// @param sql
// @param version
// @return error
func (m mockBlockchainStore) ExecDdlSql(contractName, sql string, version string) error {
	panic("implement me")
}

// BeginDbTransaction
// @param txName
// @return protocol.SqlDBTransaction
// @return error
func (m mockBlockchainStore) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic("implement me")
}

// GetDbTransaction
// @param txName
// @return protocol.SqlDBTransaction
// @return error
func (m mockBlockchainStore) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
	panic("implement me")
}

// CommitDbTransaction
// @param txName
// @return error
func (m mockBlockchainStore) CommitDbTransaction(txName string) error {
	panic("implement me")
}

// RollbackDbTransaction
// @param txName
// @return error
func (m mockBlockchainStore) RollbackDbTransaction(txName string) error {
	panic("implement me")
}

// InitGenesis
// @param genesisBlock
// @return error
func (m mockBlockchainStore) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
	panic("implement me")
}

// PutBlock
// @param block
// @param txRWSets
// @return error
func (m mockBlockchainStore) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
	panic("implement me")
}

// SelectObject select object
func (m mockBlockchainStore) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator,
	error) {
	panic("implement me")
}

// GetHistoryForKey returns history for key
func (m mockBlockchainStore) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
	panic("implement me")
}

// GetAccountTxHistory returns account tx history
func (m mockBlockchainStore) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
	panic("implement me")
}

// GetContractTxHistory returns contract tx history
func (m mockBlockchainStore) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
	panic("implement me")
}

// GetBlockByHash return block by hash
func (m mockBlockchainStore) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
	panic("implement me")
}

// BlockExists judge block exist
func (m mockBlockchainStore) BlockExists(blockHash []byte) (bool, error) {
	panic("implement me")
}

// GetBlock returns block
func (m mockBlockchainStore) GetBlock(height uint64) (*commonPb.Block, error) {
	panic("implement me")
}

// GetLastConfigBlock returns last config block
func (m mockBlockchainStore) GetLastConfigBlock() (*commonPb.Block, error) {
	panic("implement me")
}

// GetBlockByTx returns block by tx
func (m mockBlockchainStore) GetBlockByTx(txId string) (*commonPb.Block, error) {
	panic("implement me")
}

// GetBlockWithRWSets returns block with rwsets
func (m mockBlockchainStore) GetBlockWithRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
	panic("implement me")
}

// GetTx returns tx
func (m mockBlockchainStore) GetTx(txId string) (*commonPb.Transaction, error) {
	panic("implement me")
}

// TxExists judge whether tx exists
func (m mockBlockchainStore) TxExists(txId string) (bool, error) {
	panic("implement me")
}

// GetTxConfirmedTime returns tx confirmed time
func (m mockBlockchainStore) GetTxConfirmedTime(txId string) (int64, error) {
	panic("implement me")
}

// GetLastBlock returns last block
func (m mockBlockchainStore) GetLastBlock() (*commonPb.Block, error) {
	return &commonPb.Block{
		Header: &commonPb.BlockHeader{
			ChainId:        "",
			BlockHeight:    0,
			PreBlockHash:   nil,
			BlockHash:      nil,
			PreConfHeight:  0,
			BlockVersion:   0,
			DagHash:        nil,
			RwSetRoot:      nil,
			TxRoot:         nil,
			BlockTimestamp: 0,
			Proposer:       nil,
			ConsensusArgs:  nil,
			TxCount:        0,
			Signature:      nil,
		},
		Dag:            nil,
		Txs:            nil,
		AdditionalData: nil,
	}, nil
}

// ReadObject returns object
func (m mockBlockchainStore) ReadObject(contractName string, key []byte) ([]byte, error) {
	panic("implement me")
}

// GetTxRWSet returns tx rwset
func (m mockBlockchainStore) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
	panic("implement me")
}

// GetTxRWSetsByHeight returns tx rwset by height
func (m mockBlockchainStore) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
	panic("implement me")
}

// GetDBHandle returns db handle
func (m mockBlockchainStore) GetDBHandle(dbName string) protocol.DBHandle {
	panic("implement me")
}

// Close the mockBlockchainStore
func (m mockBlockchainStore) Close() error {
	panic("implement me")
}
