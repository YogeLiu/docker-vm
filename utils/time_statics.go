package utils

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
)

type SysCallDuration struct {
	OpType          protogo.DockerVMType
	StartTime       int64
	TotalDuration   int64
	StorageDuration int64
}

func NewSysCallDuration(opType protogo.DockerVMType, startTime int64, totalTime int64,
	storageTime int64) *SysCallDuration {
	return &SysCallDuration{
		OpType:          opType,
		StartTime:       startTime,
		TotalDuration:   totalTime,
		StorageDuration: storageTime,
	}
}

func (s *SysCallDuration) ToString() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%s start: %v, spend: %dμs, r/w store: %dμs; ",
		s.OpType.String(), time.Unix(s.StartTime/1e9, s.StartTime%1e9), s.TotalDuration/1000, s.StorageDuration/1000,
	)
}

type TxDuration struct {
	OriginalTxId              string
	TxId                      string
	StartTime                 int64
	EndTime                   int64
	TotalDuration             int64
	SysCallCnt                int32
	SysCallDuration           int64
	StorageDuration           int64
	ContingentSysCallCnt      int32
	ContingentSysCallDuration int64
	CrossCallCnt              int32
	CrossCallDuration         int64
	SysCallList               []*SysCallDuration
	CrossCallList             []*TxDuration
}

func NewTxDuration(originalTxId, txId string, startTime int64) *TxDuration {
	return &TxDuration{
		OriginalTxId: originalTxId,
		TxId:         txId,
		StartTime:    startTime,
	}
}

func (e *TxDuration) ToString() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s spend time: %dμs, syscall: %dμs(%d), r/w store: %dμs, possible syscall: %dμs(%d)"+
		"cross contract: %dμs(%d)",
		e.TxId, e.TotalDuration/1000, e.SysCallDuration/1000, e.SysCallCnt, e.StorageDuration/1000,
		e.ContingentSysCallDuration/1000, e.ContingentSysCallCnt, e.CrossCallDuration/1000, e.CrossCallCnt,
	)
}

func (e *TxDuration) PrintSysCallList() string {
	if e.SysCallList == nil {
		return "no syscalls"
	}
	var sb strings.Builder
	for _, sysCallTime := range e.SysCallList {
		sb.WriteString(sysCallTime.ToString())
	}
	return sb.String()
}

// StartSysCall start new sys call
func (e *TxDuration) StartSysCall(msgType protogo.DockerVMType) {
	duration := &SysCallDuration{
		OpType:    msgType,
		StartTime: time.Now().UnixNano(),
	}
	e.SysCallList = append(e.SysCallList, duration)
}

// AddLatestStorageDuration add storage time to latest sys call
func (e *TxDuration) AddLatestStorageDuration(duration int64) error {
	latestSysCall, err := e.GetLatestSysCall()
	if err != nil {
		return fmt.Errorf("failed to get latest sys call, %v", err)
	}
	latestSysCall.StorageDuration += duration
	return nil
}

// EndSysCall close new sys call
func (e *TxDuration) EndSysCall(msg *protogo.DockerVMMessage) error {
	latestSysCall, err := e.GetLatestSysCall()
	if err != nil {
		return fmt.Errorf("failed to get latest sys call, %v", err)
	}
	latestSysCall.TotalDuration = time.Since(time.Unix(0, latestSysCall.StartTime)).Nanoseconds()
	e.addSysCallDuration(latestSysCall)
	if latestSysCall.OpType == protogo.DockerVMType_TX_RESPONSE {
		e.CrossCallCnt = msg.Response.TxDuration.CrossCallCnt
		e.CrossCallDuration = msg.Response.TxDuration.CrossCallTime
	}
	return nil
}

// GetLatestSysCall returns latest sys call
func (e *TxDuration) GetLatestSysCall() (*SysCallDuration, error) {
	if len(e.SysCallList) == 0 {
		return nil, errors.New("sys call list length == 0")
	}
	return e.SysCallList[len(e.SysCallList)-1], nil
}

// todo add lock (maybe do not need)
func (e *TxDuration) addSysCallDuration(duration *SysCallDuration) {
	if duration == nil {
		return
	}
	switch duration.OpType {
	case protogo.DockerVMType_GET_BYTECODE_REQUEST:
		e.ContingentSysCallCnt++
		e.ContingentSysCallDuration += duration.TotalDuration
		e.StorageDuration += duration.StorageDuration
	case protogo.DockerVMType_GET_STATE_REQUEST, protogo.DockerVMType_GET_BATCH_STATE_REQUEST,
		protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST, protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST,
		protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST, protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST,
		protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
		e.SysCallCnt++
		e.SysCallDuration += duration.TotalDuration
		e.StorageDuration += duration.StorageDuration
	default:
		return
	}
}

func (e *TxDuration) Add(txDuration *TxDuration) {
	if txDuration == nil {
		return
	}

	// 跨合约调用时直接更新跟节点的以下属性，以便最终统计
	e.TotalDuration += txDuration.TotalDuration

	e.SysCallCnt += txDuration.SysCallCnt
	e.SysCallDuration += txDuration.SysCallDuration
	e.StorageDuration += txDuration.StorageDuration

	e.ContingentSysCallCnt += txDuration.ContingentSysCallCnt
	e.ContingentSysCallDuration += txDuration.ContingentSysCallDuration

	e.CrossCallCnt += txDuration.CrossCallCnt
	e.CrossCallDuration += txDuration.CrossCallDuration
}

func (e *TxDuration) AddContingentSysCall(spend int64) {
	e.ContingentSysCallCnt++
	e.ContingentSysCallDuration += spend
}

type BlockTxsDuration struct {
	//txs []*TxDuration
	txs map[string]*TxDuration
}

// todo add lock
func (b *BlockTxsDuration) AddTxDuration(t *TxDuration) {
	txDuration, ok := b.txs[t.OriginalTxId]
	if !ok {
		// original tx
		b.txs[t.OriginalTxId] = t
	}

	// cross call tx
	callerNode := txDuration.getCallerNode()
	callerNode.CrossCallList = append(callerNode.CrossCallList, t)
}

func (e *TxDuration) getCallerNode() *TxDuration {
	if len(e.CrossCallList) == 0 {
		return e
	}

	return e.CrossCallList[len(e.CrossCallList)-1].getCallerNode()
}

func (b *BlockTxsDuration) FinishTxDuration(t *TxDuration) {
	txDuration, ok := b.txs[t.OriginalTxId]
	if !ok {
		// original tx
		b.txs[t.OriginalTxId] = t
	}

	// update root txDuration data to facilitate statistics in blocks
	txDuration.SysCallCnt += t.SysCallCnt
	txDuration.ContingentSysCallCnt += t.ContingentSysCallCnt
	txDuration.CrossCallCnt += t.CrossCallCnt
}

func (b *BlockTxsDuration) ToString() string {
	if b == nil {
		return ""
	}
	txTotal := NewTxDuration("", "", 0)
	for _, tx := range b.txs {
		txTotal.Add(tx)
	}
	return txTotal.ToString()
}

type BlockTxsDurationMgr struct {
	blockDurations map[string]*BlockTxsDuration
	lock           sync.Mutex
	logger         *logger.CMLogger
}

func NewBlockTxsDurationMgr() *BlockTxsDurationMgr {
	return &BlockTxsDurationMgr{
		blockDurations: make(map[string]*BlockTxsDuration),
		logger:         logger.GetLogger(logger.MODULE_RPC),
	}
}

func (r *BlockTxsDurationMgr) PrintBlockTxsDuration(id string) string {
	durations := r.blockDurations[id]
	return durations.ToString()
}

func (r *BlockTxsDurationMgr) AddBlockTxsDuration(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.blockDurations[id] == nil {
		r.blockDurations[id] = &BlockTxsDuration{}
		return
	}
	r.logger.Warnf("receive duplicated block, fingerprint: %s", id)
}

func (r *BlockTxsDurationMgr) RemoveBlockTxsDuration(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.blockDurations, id)
}

// AddTx if add tx to block map need lock
func (r *BlockTxsDurationMgr) AddTx(id string, txTime *TxDuration) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.blockDurations[id] == nil {
		return
	}
	r.blockDurations[id].AddTxDuration(txTime)
}

func (r *BlockTxsDurationMgr) FinishTx(id string, txTime *TxDuration) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.blockDurations[id] == nil {
		return
	}
	r.blockDurations[id].FinishTxDuration(txTime)
}
