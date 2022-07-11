package rpc

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/vm-engine/v2/config"
	"chainmaker.org/chainmaker/vm-engine/v2/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/utils"
	"go.uber.org/atomic"
)

var (
	clientMgrOnce sync.Once
	mgrInstance   *ContractEngineClientManager
)

const (
	txSize               = 15000
	eventSize            = 100
	retryConnectDuration = 2 * time.Second
)

// ContractEngineClientManager manager all contract engine clients
type ContractEngineClientManager struct {
	chainId        string
	startOnce      sync.Once
	logger         protocol.Logger
	count          *atomic.Uint64 // tx count
	index          uint64         // client index
	config         *config.DockerVMConfig
	notifyLock     sync.RWMutex
	clientLock     sync.Mutex
	aliveClientMap map[uint64]*ContractEngineClient              // used to restore alive client
	txSendCh       chan *protogo.DockerVMMessage                 // used to send tx to docker-go instance
	byteCodeRespCh chan *protogo.DockerVMMessage                 // used to receive GetByteCode response from docker-go
	notify         map[string]func(msg *protogo.DockerVMMessage) // used to receive tx response from docker-go
	eventCh        chan *interfaces.Event                        // used to receive event
	stop           bool
}

// NewClientManager returns new client manager
func NewClientManager(
	chainId string,
	logger protocol.Logger,
	vmConfig *config.DockerVMConfig,
) interfaces.ContractEngineClientMgr {

	clientMgrOnce.Do(func() {
		mgrInstance = &ContractEngineClientManager{
			chainId:        chainId,
			startOnce:      sync.Once{},
			logger:         logger,
			count:          atomic.NewUint64(0),
			config:         vmConfig,
			notifyLock:     sync.RWMutex{},
			clientLock:     sync.Mutex{},
			aliveClientMap: make(map[uint64]*ContractEngineClient),
			txSendCh:       make(chan *protogo.DockerVMMessage, txSize),
			byteCodeRespCh: make(chan *protogo.DockerVMMessage, txSize*8),
			notify:         make(map[string]func(msg *protogo.DockerVMMessage)),
			eventCh:        make(chan *interfaces.Event, eventSize),
			stop:           false,
		}
	})

	return mgrInstance
}

// Start establish connections
func (cm *ContractEngineClientManager) Start() error {
	cm.logger.Infof("start client manager")
	cm.logger.Infof("before start: alive conn %d", len(cm.aliveClientMap))

	var err error

	cm.startOnce.Do(func() {
		// 1. start all clients
		if err = cm.establishConnections(); err != nil {
			cm.logger.Errorf("fail to create client: %s", err)
			return
		}

		// 2. start event listen
		go cm.listen()
	})

	cm.logger.Infof("after start: alive conn %d", len(cm.aliveClientMap))
	return err
}

// Stop all connectiosn
func (cm *ContractEngineClientManager) Stop() error {
	cm.closeAllConnections()
	return nil
}

func (cm *ContractEngineClientManager) closeAllConnections() {
	cm.stop = true
	for _, client := range cm.aliveClientMap {
		client.Stop()
	}
}

// === ForRuntimeInstance ===

// PutTxRequestWithNotify put tx request with notify into tx send channel
func (cm *ContractEngineClientManager) PutTxRequestWithNotify(
	txRequest *protogo.DockerVMMessage,
	chainId string,
	notify func(msg *protogo.DockerVMMessage),
) error {

	if err := cm.registerNotify(chainId, txRequest.TxId, notify); err != nil {
		return err
	}

	cm.txSendCh <- txRequest

	return nil
}

// PutByteCodeResp put butecode resp into bytecode resp channel
func (cm *ContractEngineClientManager) PutByteCodeResp(getByteCodeResp *protogo.DockerVMMessage) {
	cm.byteCodeRespCh <- getByteCodeResp
}

func (cm *ContractEngineClientManager) registerNotify(
	chainId,
	txId string,
	notify func(msg *protogo.DockerVMMessage),
) error {

	cm.notifyLock.Lock()
	defer cm.notifyLock.Unlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txId)
	cm.logger.Debugf("register notify for [%s]", notifyKey)

	_, ok := cm.notify[notifyKey]
	if ok {
		cm.logger.Errorf("[%s] fail to register notify, cause notify already registered", txId)
	}

	cm.notify[notifyKey] = notify
	return nil
}

// DeleteNotify delete notify
func (cm *ContractEngineClientManager) DeleteNotify(chainId, txId string) bool {
	cm.notifyLock.Lock()
	defer cm.notifyLock.Unlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txId)
	cm.logger.Debugf("[%s] delete notify", notifyKey)
	if _, ok := cm.notify[notifyKey]; ok {
		delete(cm.notify, notifyKey)
		return true
	}

	cm.logger.Debugf("[%s] delete notify fail, notify is already deleted", notifyKey)
	return false
}

// GetUniqueTxKey returns unique tx key
func (cm *ContractEngineClientManager) GetUniqueTxKey(txId string) string {
	var sb strings.Builder
	nextCount := cm.count.Add(1)
	sb.WriteString(txId)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(nextCount, 10))
	return sb.String()
}

// NeedSendContractByteCode judge whether need to send contract bytecode
func (cm *ContractEngineClientManager) NeedSendContractByteCode() bool {
	return !cm.config.DockerVMUDSOpen
}

// HasActiveConnections returns the alive client map length
func (cm *ContractEngineClientManager) HasActiveConnections() bool {
	return len(cm.aliveClientMap) > 0
}

// GetVMConfig returns vm config
func (cm *ContractEngineClientManager) GetVMConfig() *config.DockerVMConfig {
	return cm.config
}

// GetTxSendChLen returns tx send chan length
func (cm *ContractEngineClientManager) GetTxSendChLen() int {
	return len(cm.txSendCh)
}

// GetByteCodeRespChLen returns bytecode resp chan length
func (cm *ContractEngineClientManager) GetByteCodeRespChLen() int {
	return len(cm.byteCodeRespCh)
}

// === forClient ===

// GetTxSendCh returns tx send channel
func (cm *ContractEngineClientManager) GetTxSendCh() chan *protogo.DockerVMMessage {
	return cm.txSendCh
}

// PutEvent put event into event channel
func (cm *ContractEngineClientManager) PutEvent(event *interfaces.Event) {
	cm.eventCh <- event
}

// GetByteCodeRespSendCh returns bytecode resp send ch
func (cm *ContractEngineClientManager) GetByteCodeRespSendCh() chan *protogo.DockerVMMessage {
	return cm.byteCodeRespCh
}

// GetReceiveNotify returns receive notify
func (cm *ContractEngineClientManager) GetReceiveNotify(chainId, txId string) func(msg *protogo.DockerVMMessage) {
	cm.notifyLock.RLock()
	defer cm.notifyLock.RUnlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txId)
	cm.logger.Debugf("get notify for [%s]", notifyKey)
	notify, ok := cm.notify[notifyKey]
	if !ok {
		cm.logger.Debugf("get receive notify[%s] failed, please check your key", notifyKey)
		return nil
	}

	return notify
}

func (cm *ContractEngineClientManager) listen() {
	cm.logger.Infof("client manager begin listen event")
	for {
		event := <-cm.eventCh
		switch event.EventType {
		case interfaces.EventType_ConnectionStopped:
			cm.dropConnection(event)
			go cm.reconnect()
		default:
			cm.logger.Warnf("unknown event: %s", event)
		}
	}
}

func (cm *ContractEngineClientManager) establishConnections() error {
	cm.logger.Debugf("establish new connections")
	totalConnections := int(utils.GetMaxConnectionFromConfig(cm.GetVMConfig()))
	var wg sync.WaitGroup
	for i := 0; i < totalConnections; i++ {
		wg.Add(1)
		go func() {
			newIndex := cm.getNextIndex()
			newClient := NewContractEngineClient(cm.chainId, newIndex, cm.logger, cm)

			for {
				if cm.stop {
					return
				}
				err := newClient.Start()
				if err == nil {
					break
				}
				cm.logger.Warnf("client[%d] connect fail, try reconnect...", newIndex)
				time.Sleep(retryConnectDuration)
			}
			cm.clientLock.Lock()
			cm.aliveClientMap[newIndex] = newClient
			cm.clientLock.Unlock()
			wg.Done()
		}()
	}

	wg.Wait()
	return nil
}

func (cm *ContractEngineClientManager) dropConnection(event *interfaces.Event) {
	cm.clientLock.Lock()
	defer cm.clientLock.Unlock()
	cm.logger.Debugf("drop connection: %d", event.Id)
	_, ok := cm.aliveClientMap[event.Id]
	if ok {
		delete(cm.aliveClientMap, event.Id)
	}
}

func (cm *ContractEngineClientManager) reconnect() {
	newIndex := cm.getNextIndex()
	newClient := NewContractEngineClient(cm.chainId, newIndex, cm.logger, cm)

	for {
		if cm.stop {
			return
		}

		if err := newClient.Start(); err != nil {
			break
		}
		cm.logger.Warnf("client[%d] connect fail, try reconnect...", newIndex)
		time.Sleep(retryConnectDuration)
	}
	cm.clientLock.Lock()
	cm.aliveClientMap[newIndex] = newClient
	cm.clientLock.Unlock()

}

func (cm *ContractEngineClientManager) getNextIndex() uint64 {
	curIndex := cm.index
	cm.index++
	return curIndex
}
