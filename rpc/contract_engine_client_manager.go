package rpc

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"
	"go.uber.org/atomic"
)

var clientMgrOnce sync.Once

const (
	txSize               = 15000
	eventSize            = 100
	retryConnectDuration = 2 * time.Second
)

type ContractEngineClientManager struct {
	chainId        string
	startOnce      sync.Once
	logger         *logger.CMLogger
	count          *atomic.Uint64 // tx count
	index          uint64         // client index
	config         *config.DockerVMConfig
	notifyLock     sync.RWMutex
	clientLock     sync.Mutex
	aliveClientMap map[uint64]interfaces.ContractEngineClient    // used to restore alive client
	txSendCh       chan *protogo.DockerVMMessage                 // used to send tx to docker-go instance
	byteCodeRespCh chan *protogo.DockerVMMessage                 // used to receive GetByteCode response from docker-go
	notify         map[string]func(msg *protogo.DockerVMMessage) // used to receive tx response from docker-go
	eventCh        chan *interfaces.Event                        // used to receive event
	stop           bool
}

func NewClientManager(chainId string, vmConfig *config.DockerVMConfig) interfaces.ContractEngineClientMgr {
	mgrCh := make(chan *ContractEngineClientManager, 1)
	clientMgrOnce.Do(func() {
		mgrCh <- &ContractEngineClientManager{
			chainId:        chainId,
			startOnce:      sync.Once{},
			logger:         logger.GetLoggerByChain(logger.MODULE_VM, chainId),
			count:          atomic.NewUint64(0),
			config:         vmConfig,
			notifyLock:     sync.RWMutex{},
			clientLock:     sync.Mutex{},
			aliveClientMap: make(map[uint64]interfaces.ContractEngineClient),
			txSendCh:       make(chan *protogo.DockerVMMessage, txSize),
			byteCodeRespCh: make(chan *protogo.DockerVMMessage, txSize*8),
			notify:         make(map[string]func(msg *protogo.DockerVMMessage)),
			eventCh:        make(chan *interfaces.Event, eventSize),
			stop:           false,
		}
	})

	return <-mgrCh
}

func (cm *ContractEngineClientManager) Start() error {
	cm.logger.Infof("start client manager")

	startErrCh := make(chan error, 1)
	defer close(startErrCh)

	cm.startOnce.Do(func() {
		// 1. start all clients
		if err := cm.establishConnections(); err != nil {
			cm.logger.Errorf("fail to create client: %s", err)
			startErrCh <- err
			return
		}

		startErrCh <- nil

		// 2. start event listen
		go cm.listen()
	})

	return <-startErrCh
}

func (cm *ContractEngineClientManager) Stop() error {
	return nil
}

// === ForRuntimeInstance ===

func (cm *ContractEngineClientManager) PutTxRequestWithNotify(txRequest *protogo.DockerVMMessage, chainId string, notify func(msg *protogo.DockerVMMessage)) error {
	if err := cm.registerNotify(chainId, txRequest.TxId, notify); err != nil {
		return err
	}

	cm.txSendCh <- txRequest

	return nil
}

func (cm *ContractEngineClientManager) PutByteCodeResp(getByteCodeResp *protogo.DockerVMMessage) {
	cm.byteCodeRespCh <- getByteCodeResp
}

func (cm *ContractEngineClientManager) registerNotify(chainId, txId string, notify func(msg *protogo.DockerVMMessage)) error {
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

func (cm *ContractEngineClientManager) DeleteNotify(chainId, txId string) bool {
	cm.notifyLock.Lock()
	defer cm.notifyLock.Unlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txId)
	cm.logger.Debugf("[%s] delete notify", notifyKey)
	_, ok := cm.notify[notifyKey]
	if ok {
		delete(cm.notify, notifyKey)
		return true
	}

	cm.logger.Debugf("[%s] delete notify fail, notify is already deleted", notifyKey)
	return false
}

func (cm *ContractEngineClientManager) GetUniqueTxKey(txId string) string {
	var sb strings.Builder
	nextCount := cm.count.Add(1)
	sb.WriteString(txId)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(nextCount, 10))
	return sb.String()
}

func (cm *ContractEngineClientManager) NeedSendContractByteCode() bool {
	// TODO:
	//return !cm.config.DockerVMUDSOpen
	return false
}

func (cm *ContractEngineClientManager) HasActiveConnections() bool {
	return len(cm.aliveClientMap) > 0
}

func (cm *ContractEngineClientManager) GetVMConfig() *config.DockerVMConfig {
	return cm.config
}

func (cm *ContractEngineClientManager) GetTxSendChLen() int {
	return len(cm.txSendCh)
}

func (cm *ContractEngineClientManager) GetByteCodeRespChLen() int {
	return len(cm.byteCodeRespCh)
}

// === forClient ===

func (cm *ContractEngineClientManager) GetTxSendCh() chan *protogo.DockerVMMessage {
	return cm.txSendCh
}

func (cm *ContractEngineClientManager) PutEvent(event *interfaces.Event) {
	cm.eventCh <- event
}

func (cm *ContractEngineClientManager) GetByteCodeRespSendCh() chan *protogo.DockerVMMessage {
	return cm.byteCodeRespCh
}

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
				err := newClient.StartClient()
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

		if err := newClient.StartClient(); err != nil {
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
