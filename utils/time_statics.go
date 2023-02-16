package utils

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/config"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
)

var sysCallPool = sync.Pool{
	New: func() interface{} {
		return &SysCallDuration{}
	},
}

//// TxStepPool is the tx step sync pool
//var TxStepPool = sync.Pool{
//	New: func() interface{} {
//		return &protogo.StepDuration{}
//	},
//}

// SysCallDuration .
type SysCallDuration struct {
	OpType          protogo.DockerVMType
	StartTime       int64
	TotalDuration   int64
	StorageDuration int64
}

// NewSysCallDuration construct a SysCallDuration
func NewSysCallDuration(opType protogo.DockerVMType, startTime int64, totalTime int64,
	storageTime int64) *SysCallDuration {
	return &SysCallDuration{
		OpType:          opType,
		StartTime:       startTime,
		TotalDuration:   totalTime,
		StorageDuration: storageTime,
	}
}

// ToString .
func (s *SysCallDuration) ToString() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%s start: %v, spend: %dμs, r/w store: %dμs; ",
		s.OpType.String(), time.Unix(s.StartTime/1e9, s.StartTime%1e9), s.TotalDuration/1000, s.StorageDuration/1000,
	)
}

// TxDuration .
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
	Sealed                    bool
}

// NewTxDuration .
func NewTxDuration(originalTxId, txId string, startTime int64) *TxDuration {
	return &TxDuration{
		OriginalTxId: originalTxId,
		TxId:         txId,
		StartTime:    startTime,
		SysCallList:  make([]*SysCallDuration, 0, 8),
	}
}

// ToString .
func (e *TxDuration) ToString() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s spend time: %dμs, syscall: %dμs(count: %d), r/w store: %dμs, "+
		"possible syscall: %dμs(count: %d), cross contract: %dμs(count: %d)",
		e.TxId, e.TotalDuration/1000, e.SysCallDuration/1000, e.SysCallCnt, e.StorageDuration/1000,
		e.ContingentSysCallDuration/1000, e.ContingentSysCallCnt, e.CrossCallDuration/1000, e.CrossCallCnt,
	)
}

// PrintSysCallList print tx duration
func (e *TxDuration) PrintSysCallList() string {
	if e.SysCallList == nil {
		return "no syscalls"
	}
	var sb strings.Builder
	crossCnt := len(e.CrossCallList)
	crossIndex := 0
	for _, sysCallTime := range e.SysCallList {
		if sysCallTime.OpType == protogo.DockerVMType_CALL_CONTRACT_REQUEST {
			if crossIndex < crossCnt {
				sb.WriteString(sysCallTime.ToString())
				sb.WriteString("< details of the cross call from [" + e.TxId + "]: ")
				sb.WriteString(e.CrossCallList[crossIndex].PrintSysCallList())
				sb.WriteString("> ")
				crossIndex++
			}
		} else {
			sb.WriteString(sysCallTime.ToString())
		}
	}
	return sb.String()
}

// StartSysCall start new sys call
func (e *TxDuration) StartSysCall(msgType protogo.DockerVMType) {
	duration, _ := sysCallPool.Get().(*SysCallDuration)
	duration.OpType = msgType
	duration.StartTime = time.Now().UnixNano()
	duration.StorageDuration = 0
	duration.TotalDuration = 0
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
// addSysCallDuration add the count of system calls and the duration of system calls to the total record
func (e *TxDuration) addSysCallDuration(duration *SysCallDuration) {
	if duration == nil {
		return
	}
	switch duration.OpType {
	case protogo.DockerVMType_GET_BYTECODE_REQUEST:
		// get bytecode are recorded separately, which is different from syscall
		e.ContingentSysCallCnt++
		e.ContingentSysCallDuration += duration.TotalDuration
		e.StorageDuration += duration.StorageDuration
	case protogo.DockerVMType_GET_STATE_REQUEST, protogo.DockerVMType_GET_BATCH_STATE_REQUEST,
		protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST, protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST,
		protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST, protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST,
		protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
		// record all syscalls except cross contract calls and get bytecode
		e.SysCallCnt++
		e.SysCallDuration += duration.TotalDuration
		e.StorageDuration += duration.StorageDuration
	case protogo.DockerVMType_CALL_CONTRACT_REQUEST:
		// cross contract calls are recorded separately, which is different from syscall
		e.CrossCallCnt++
		e.CrossCallDuration += duration.TotalDuration
	default:
		return
	}
}

// Add the param txDuration to self
// just add the root duration,
// the duration of child nodes will be synchronized to the root node at the end of statistics
func (e *TxDuration) Add(txDuration *TxDuration) {
	if txDuration == nil {
		return
	}

	e.TotalDuration += txDuration.TotalDuration

	e.SysCallCnt += txDuration.SysCallCnt
	e.SysCallDuration += txDuration.SysCallDuration
	e.StorageDuration += txDuration.StorageDuration

	e.ContingentSysCallCnt += txDuration.ContingentSysCallCnt
	e.ContingentSysCallDuration += txDuration.ContingentSysCallDuration

	e.CrossCallCnt += txDuration.CrossCallCnt
	e.CrossCallDuration += txDuration.CrossCallDuration
}

// AddContingentSysCall add contingent syscall
func (e *TxDuration) AddContingentSysCall(spend int64) {
	e.ContingentSysCallCnt++
	e.ContingentSysCallDuration += spend
}

// Seal the txDuration, The sealed node will no longer update its state
// and will not generate new cross contract calls at this node and its child nodes
func (e *TxDuration) Seal() {
	e.Sealed = true
}

// BlockTxsDuration record the duration of all transactions in a block
type BlockTxsDuration struct {
	txs map[string]*TxDuration
}

// AddTxDuration add a TxDuration
func (b *BlockTxsDuration) AddTxDuration(t *TxDuration) {
	// todo add lock
	if b.txs == nil {
		b.txs = make(map[string]*TxDuration)
	}

	// the original txDuration is recorded in BlockTxsDuration as the root of the txDuration tree
	txDuration, ok := b.txs[t.OriginalTxId]
	if !ok {
		// original tx
		b.txs[t.OriginalTxId] = t
		return
	}

	// cross call tx
	// cross contract call finds its own parent node through the root
	callerNode := txDuration.getCallerNode()
	callerNode.CrossCallList = append(callerNode.CrossCallList, t)
}

// getCallerNode return the only node in the tree that can be called across contracts
func (e *TxDuration) getCallerNode() *TxDuration {
	for _, item := range e.CrossCallList {

		// sealed nodes cannot be called across contracts
		if item.Sealed {
			continue
		}

		// unsealed nodes may be called across contracts
		return item.getCallerNode()
	}

	// when the CrossCallList is empty or all nodes in CrossCallList are sealed,
	// it is the only node that can initiate cross contract call
	return e
}

// FinishTxDuration end time statistics
func (b *BlockTxsDuration) FinishTxDuration(t *TxDuration) {
	if b.txs == nil {
		return
	}

	txDuration, ok := b.txs[t.OriginalTxId]
	if !ok {
		return
	}

	// add the following information to the root node when the cross call node finishes statistics
	txDuration.SysCallCnt += t.SysCallCnt
	txDuration.SysCallDuration += t.SysCallDuration

	txDuration.ContingentSysCallCnt += t.ContingentSysCallCnt
	txDuration.ContingentSysCallDuration += t.ContingentSysCallDuration

	txDuration.CrossCallCnt += t.CrossCallCnt
	txDuration.CrossCallDuration += t.CrossCallDuration

	for _, v := range txDuration.SysCallList {
		sysCallPool.Put(v)
	}
}

// ToString .
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

// BlockTxsDurationMgr record the txDuration of each block
type BlockTxsDurationMgr struct {
	blockDurations map[string]*BlockTxsDuration
	lock           sync.RWMutex
	logger         *logger.CMLogger
}

// NewBlockTxsDurationMgr construct a BlockTxsDurationMgr
func NewBlockTxsDurationMgr() *BlockTxsDurationMgr {
	return &BlockTxsDurationMgr{
		blockDurations: make(map[string]*BlockTxsDuration),
		logger:         logger.GetLogger(logger.MODULE_RPC),
	}
}

// PrintBlockTxsDuration returns the duration of the specified block
func (r *BlockTxsDurationMgr) PrintBlockTxsDuration(id string) string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	durations := r.blockDurations[id]
	return durations.ToString()
}

// AddBlockTxsDuration .
func (r *BlockTxsDurationMgr) AddBlockTxsDuration(id string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.blockDurations[id] == nil {
		r.blockDurations[id] = &BlockTxsDuration{}
		return
	}
	r.logger.Warnf("receive duplicated block, fingerprint: %s", id)
}

// RemoveBlockTxsDuration .
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

// FinishTx .
func (r *BlockTxsDurationMgr) FinishTx(id string, txTime *TxDuration) {
	txTime.Seal()
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.blockDurations[id] == nil {
		return
	}
	r.blockDurations[id].FinishTxDuration(txTime)
}

// EnterNextStep enter next duration tx step
func EnterNextStep(msg *protogo.DockerVMMessage, stepType protogo.StepType, getStr func() string) {
	if config.VMConfig.Slow.Disable {
		return
	}
	if stepType != protogo.StepType_RUNTIME_PREPARE_TX_REQUEST {
		endTxStep(msg)
	}
	addTxStep(msg, stepType, getStr())
	if stepType == protogo.StepType_RUNTIME_HANDLE_TX_RESPONSE {
		endTxStep(msg)
	}
}

func addTxStep(msg *protogo.DockerVMMessage, stepType protogo.StepType, log string) {
	//stepDur, _ := TxStepPool.Get().(*protogo.StepDuration)
	//stepDur.Type = stepType
	//stepDur.StartTime = time.Now().UnixNano()
	//stepDur.Msg = log
	//stepDur.StepDuration = 0
	//stepDur.UntilDuration = 0
	msg.StepDurations = append(msg.StepDurations, &protogo.StepDuration{
		Type:      stepType,
		StartTime: time.Now().UnixNano(),
		Msg:       log,
	})
}

func endTxStep(msg *protogo.DockerVMMessage) {
	if len(msg.StepDurations) == 0 {
		return
	}
	stepLen := len(msg.StepDurations)
	currStep := msg.StepDurations[stepLen-1]
	firstStep := msg.StepDurations[0]
	currStep.UntilDuration = time.Since(time.Unix(0, firstStep.StartTime)).Nanoseconds()
	currStep.StepDuration = time.Since(time.Unix(0, currStep.StartTime)).Nanoseconds()
}

// PrintTxSteps print all duration tx steps
func PrintTxSteps(msg *protogo.DockerVMMessage) string {
	var sb strings.Builder
	for _, step := range msg.StepDurations {
		sb.WriteString(fmt.Sprintf("<step: %q, start time: %v, step cost: %vms, until cost: %vms, msg: %s> ",
			step.Type, time.Unix(0, step.StartTime),
			time.Duration(step.StepDuration).Seconds()*1000,
			time.Duration(step.UntilDuration).Seconds()*1000,
			step.Msg))
	}
	return sb.String()
}

// PrintTxStepsWithTime print all duration tx steps with time limt
func PrintTxStepsWithTime(msg *protogo.DockerVMMessage) (string, bool) {
	if len(msg.StepDurations) == 0 {
		return "", false
	}
	lastStep := msg.StepDurations[len(msg.StepDurations)-1]
	var sb strings.Builder
	if lastStep.UntilDuration > time.Second.Nanoseconds()*int64(config.VMConfig.Slow.TxTime) {
		sb.WriteString("slow tx overall: ")
		sb.WriteString(PrintTxSteps(msg))
		return sb.String(), true
	}
	for _, step := range msg.StepDurations {
		if step.StepDuration > time.Second.Nanoseconds()*int64(config.VMConfig.Slow.StepTime) {
			sb.WriteString(fmt.Sprintf("slow tx at step %q, step cost: %vms: ",
				step.Type, time.Duration(step.StepDuration).Seconds()*1000))
			sb.WriteString(PrintTxSteps(msg))
			return sb.String(), true
		}
	}
	return "", false
}
