package test

//
//import (
//	"io/ioutil"
//	"sync"
//
//	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
//	"chainmaker.org/chainmaker/utils/v2"
//	"github.com/docker/distribution/uuid"
//
//	acPb "chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
//	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
//	storePb "chainmaker.org/chainmaker/pb-go/v2/store"
//	"chainmaker.org/chainmaker/protocol/v2"
//)
//
//var testOrgId = "wx-org1.chainmaker.org"
//
//var CertFilePath = "./testdata/admin1.sing.crt"
//
//var file []byte
//
//type TxContextMockTest struct {
//	lock          *sync.Mutex
//	vmManager     protocol.VmManager
//	gasUsed       uint64 // only for callContract
//	currentDepth  int
//	currentResult []byte
//	hisResult     []*callContractResult
//
//	sender   *acPb.Member
//	creator  *acPb.Member
//	CacheMap map[string][]byte
//}
//
//func (s *TxContextMockTest) GetBlockTimestamp() int64 {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
//	//TODO implement me
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) PutIntoReadSet(contractName string, key []byte, value []byte) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetHistoryIterForKey(contractName string, key []byte) (protocol.KeyHistoryIterator,
//	error) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) CallContract(contract *commonPb.Contract, method string, byteCode []byte,
//	parameter map[string][]byte, gasUsed uint64, refTxType commonPb.TxType) (*commonPb.ContractResult,
//	protocol.ExecOrderTxType, commonPb.TxStatusCode) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) SetIter(index int32, iter interface{}) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetIter(index int32) (interface{}, bool) {
//	panic("implement me")
//}
//
//// initialize TxContext and Contract
//func InitContextTest() *TxContextMockTest {
//
//	if file == nil {
//		var err error
//		file, err = ioutil.ReadFile(CertFilePath)
//		if err != nil {
//			panic("file is nil" + err.Error())
//		}
//	}
//	sender := &acPb.Member{
//		OrgId:      testOrgId,
//		MemberInfo: file,
//		//IsFullCert: true,
//	}
//
//	txContext := TxContextMockTest{
//		lock:      &sync.Mutex{},
//		vmManager: nil,
//		hisResult: make([]*callContractResult, 0),
//		creator:   sender,
//		sender:    sender,
//		CacheMap:  make(map[string][]byte),
//	}
//
//	return &txContext
//}
//func (s *TxContextMockTest) GetContractByName(name string) (*commonPb.Contract, error) {
//	return utils.GetContractByName(s.Get, name)
//}
//
//func (s *TxContextMockTest) GetContractBytecode(name string) ([]byte, error) {
//	return utils.GetContractBytecode(s.Get, name)
//}
//
//func (s *TxContextMockTest) GetBlockVersion() uint32 {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) SetStateKvHandle(i int32, iterator protocol.StateIterator) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetStateKvHandle(i int32) (protocol.StateIterator, bool) {
//	panic("implement me")
//}
//
////func (s *TxContextMockTest) PutRecord(contractName string, value []byte, sqlType protocol.SqlType) {
////	panic("implement me")
////}
//
//func (s *TxContextMockTest) Select(name string, startKey []byte, limit []byte) (protocol.StateIterator, error) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetBlockProposer() *acPb.Member {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) SetStateSqlHandle(i int32, rows protocol.SqlRows) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetStateSqlHandle(i int32) (protocol.SqlRows, bool) {
//	panic("implement me")
//}
//
//type callContractResult struct {
//	contractName string
//	method       string
//	param        map[string][]byte
//	deep         int
//	gasUsed      uint64
//	result       []byte
//}
//
//func (s *TxContextMockTest) Get(name string, key []byte) ([]byte, error) {
//	s.lock.Lock()
//	defer s.lock.Unlock()
//	k := string(key)
//	if name != "" {
//		k = name + "::" + k
//	}
//
//	value := s.CacheMap[k]
//	//fmt.Printf("[get] key: %s, len of value is %d\n", k, len(value))
//	return value, nil
//}
//
//func (s *TxContextMockTest) Put(name string, key []byte, value []byte) error {
//	s.lock.Lock()
//	defer s.lock.Unlock()
//	k := string(key)
//	//v := string(value)
//	if name != "" {
//		k = name + "::" + k
//	}
//	//fmt.Printf("[put] key is %s, len of value is: %d\n", key, len(value))
//	s.CacheMap[k] = value
//	return nil
//}
//
//func (s *TxContextMockTest) Del(name string, key []byte) error {
//	s.lock.Lock()
//	defer s.lock.Unlock()
//	k := string(key)
//	//v := string(value)
//	if name != "" {
//		k = name + "::" + k
//	}
//	//println("【put】 key:" + k)
//	s.CacheMap[k] = nil
//	return nil
//}
//
//func (s *TxContextMockTest) GetCurrentResult() []byte {
//	return s.currentResult
//}
//
//func (s *TxContextMockTest) GetTx() *commonPb.Transaction {
//	tx := &commonPb.Transaction{
//		Payload: &commonPb.Payload{
//			chainId:        chainId,
//			TxType:         txType,
//			TxId:           uuid.Generate().String(),
//			Timestamp:      0,
//			ExpirationTime: 0,
//		},
//		Result: nil,
//	}
//	return tx
//}
//
//func (*TxContextMockTest) GetBlockHeight() uint64 {
//	return 0
//}
//func (s *TxContextMockTest) GetTxResult() *commonPb.Result {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) SetTxResult(txResult *commonPb.Result) {
//	panic("implement me")
//}
//
//func (TxContextMockTest) GetTxRWSet(runVmSuccess bool) *commonPb.TxRWSet {
//	return &commonPb.TxRWSet{
//		TxId:     "txId",
//		TxReads:  nil,
//		TxWrites: nil,
//	}
//}
//
//func (s *TxContextMockTest) GetCreator(namespace string) *acPb.Member {
//	return s.creator
//}
//
//func (s *TxContextMockTest) GetSender() *acPb.Member {
//	return s.sender
//}
//
//// func (*TxContextMockTest) GetBlockchainStore() protocol.BlockchainStore {
//// 	return &mockBlockchainStore{}
////  }
//
//func (*TxContextMockTest) GetBlockchainStore() protocol.BlockchainStore {
//	return nil
//}
//
//func (*TxContextMockTest) GetAccessControl() (protocol.AccessControlProvider, error) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetChainNodesInfoProvider() (protocol.ChainNodesInfoProvider, error) {
//	panic("implement me")
//}
//
//func (*TxContextMockTest) GetTxExecSeq() int {
//	panic("implement me")
//}
//
//func (*TxContextMockTest) SetTxExecSeq(i int) {
//	panic("implement me")
//}
//
//func (s *TxContextMockTest) GetDepth() int {
//	return s.currentDepth
//}
//
//func BaseParam(parameters map[string][]byte) {
//	parameters[protocol.ContractTxIdParam] = []byte("TX_ID")
//	parameters[protocol.ContractCreatorOrgIdParam] = []byte("org_a")
//	parameters[protocol.ContractCreatorRoleParam] = []byte("admin")
//	parameters[protocol.ContractCreatorPkParam] = []byte("1234567890abcdef1234567890abcdef")
//	parameters[protocol.ContractSenderOrgIdParam] = []byte("org_b")
//	parameters[protocol.ContractSenderRoleParam] = []byte("user")
//	parameters[protocol.ContractSenderPkParam] = []byte("11223344556677889900aabbccddeeff")
//	parameters[protocol.ContractBlockHeightParam] = []byte("1")
//}
//
//type mockBlockchainStore struct {
//}
//
//func (m mockBlockchainStore) GetContractByName(name string) (*commonPb.Contract, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetContractBytecode(name string) ([]byte, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetHeightByHash(blockHash []byte) (uint64, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetBlockHeaderByHeight(height uint64) (*commonPb.BlockHeader, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetLastChainConfig() (*configPb.ChainConfig, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetTxHeight(txId string) (uint64, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetArchivedPivot() uint64 {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) ArchiveBlock(archiveHeight uint64) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) RestoreBlocks(serializedBlocks [][]byte) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) QuerySingle(contractName, sql string, values ...interfaces {}) (protocol.SqlRow, error) {
//panic("implement me")
//}
//
//func (m mockBlockchainStore) QueryMulti(contractName, sql string, values ...interfaces {}) (protocol.SqlRows, error) {
//panic("implement me")
//}
//
//func (m mockBlockchainStore) ExecDdlSql(contractName, sql string, version string) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) BeginDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetDbTransaction(txName string) (protocol.SqlDBTransaction, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) CommitDbTransaction(txName string) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) RollbackDbTransaction(txName string) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) InitGenesis(genesisBlock *storePb.BlockWithRWSet) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) PutBlock(block *commonPb.Block, txRWSets []*commonPb.TxRWSet) error {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) SelectObject(contractName string, startKey []byte, limit []byte) (protocol.StateIterator,
//	error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetHistoryForKey(contractName string, key []byte) (protocol.KeyHistoryIterator, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetAccountTxHistory(accountId []byte) (protocol.TxHistoryIterator, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetContractTxHistory(contractName string) (protocol.TxHistoryIterator, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetBlockByHash(blockHash []byte) (*commonPb.Block, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) BlockExists(blockHash []byte) (bool, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetBlock(height uint64) (*commonPb.Block, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetLastConfigBlock() (*commonPb.Block, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetBlockByTx(txId string) (*commonPb.Block, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetBlockWithRWSets(height uint64) (*storePb.BlockWithRWSet, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetTx(txId string) (*commonPb.Transaction, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) TxExists(txId string) (bool, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetTxConfirmedTime(txId string) (int64, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetLastBlock() (*commonPb.Block, error) {
//	return &commonPb.Block{
//		Header: &commonPb.BlockHeader{
//			chainId:        "",
//			BlockHeight:    0,
//			PreBlockHash:   nil,
//			BlockHash:      nil,
//			PreConfHeight:  0,
//			BlockVersion:   0,
//			DagHash:        nil,
//			RwSetRoot:      nil,
//			TxRoot:         nil,
//			BlockTimestamp: 0,
//			Proposer:       nil,
//			ConsensusArgs:  nil,
//			TxCount:        0,
//			Signature:      nil,
//		},
//		Dag:            nil,
//		Txs:            nil,
//		AdditionalData: nil,
//	}, nil
//}
//
//func (m mockBlockchainStore) ReadObject(contractName string, key []byte) ([]byte, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetTxRWSet(txId string) (*commonPb.TxRWSet, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetTxRWSetsByHeight(height uint64) ([]*commonPb.TxRWSet, error) {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) GetDBHandle(dbName string) protocol.DBHandle {
//	panic("implement me")
//}
//
//func (m mockBlockchainStore) Close() error {
//	panic("implement me")
//}
