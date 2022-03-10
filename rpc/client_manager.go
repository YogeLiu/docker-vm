package rpc

import (
	"strconv"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"

	"chainmaker.org/chainmaker/logger/v2"

	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"go.uber.org/atomic"
)

type EventType int

type Event struct {
	id        uint64
	eventType EventType
}

const (
	connectionStopped EventType = iota
	connectionIncrease
)

const (
	maxClientNum = 2
	clientDelta  = 10
	txSize       = 15000
	eventSize    = 100
)

type ClientManager struct {
	chainId               string
	logger                *logger.CMLogger
	count                 *atomic.Uint64 // tx count
	index                 uint64         // client index
	receiveChanLock       sync.RWMutex
	config                *config.DockerVMConfig
	aliveClientReachLimit bool
	aliveClientMap        map[uint64]*CDMClient               // used to restore alive client
	txSendCh              chan *protogo.CDMMessage            // used to send tx to docker-go instance
	sysCallRespSendCh     chan *protogo.CDMMessage            // used to receive message from docker-go
	receiveChMap          map[string]chan *protogo.CDMMessage // used to receive tx response from docker-go
	eventCh               chan *Event                         // used to receive event
	stopAllSend           chan struct{}
	stopAllReceive        chan struct{}
}

func NewClientManager(chainId string, vmConfig *config.DockerVMConfig) *ClientManager {
	return &ClientManager{
		chainId:               chainId,
		logger:                logger.GetLoggerByChain(logger.MODULE_VM, chainId),
		count:                 atomic.NewUint64(0),
		index:                 1,
		config:                vmConfig,
		aliveClientReachLimit: false,
		aliveClientMap:        make(map[uint64]*CDMClient),
		txSendCh:              make(chan *protogo.CDMMessage, txSize),
		sysCallRespSendCh:     make(chan *protogo.CDMMessage, txSize*8),
		receiveChMap:          make(map[string]chan *protogo.CDMMessage),
		eventCh:               make(chan *Event, eventSize),
		stopAllSend:           make(chan struct{}),
		stopAllReceive:        make(chan struct{}),
	}
}

func (cm *ClientManager) Start() error {
	cm.logger.Infof("start client manager")
	// 1. start one client
	if err := cm.increaseConnection(); err != nil {
		cm.logger.Errorf("fail to create first client: %s", err)
		return err
	}
	// 2. start event listen
	go cm.listen()

	return nil
}

func (cm *ClientManager) Stop() error {
	// 1. stop all client
	// 2. clean alive client map
	// 3. stop event listen
	return nil
}

func (cm *ClientManager) HasConnection() bool {
	if cm.aliveClientMap != nil && len(cm.aliveClientMap) > 0 {
		return true
	}
	return false
}

func (cm *ClientManager) GetUniqueTxKey(txId string) string {
	var sb strings.Builder
	nextCount := cm.count.Add(1)
	sb.WriteString(txId)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(nextCount, 10))
	return sb.String()
}

func (cm *ClientManager) GetTxSendCh() chan *protogo.CDMMessage {
	return cm.txSendCh
}

func (cm *ClientManager) GetSysCallRespSendCh() chan *protogo.CDMMessage {
	return cm.sysCallRespSendCh
}

func (cm *ClientManager) GetVMConfig() *config.DockerVMConfig {
	return cm.config
}

func (cm *ClientManager) PutEvent(event *Event) {
	cm.eventCh <- event
}

func (cm *ClientManager) PutTxRequest(txRequest *protogo.CDMMessage) {
	cm.logger.Debugf("[%s] put tx in send chan with length [%d]", txRequest.TxId, len(cm.txSendCh))
	cm.txSendCh <- txRequest
	if !cm.aliveClientReachLimit && len(cm.txSendCh) > clientDelta {
		cm.eventCh <- &Event{
			id:        0,
			eventType: connectionIncrease,
		}
	}
}

func (cm *ClientManager) PutSysCallResponse(sysCallResp *protogo.CDMMessage) {
	cm.sysCallRespSendCh <- sysCallResp
}

func (cm *ClientManager) RegisterReceiveChan(txId string, receiveCh chan *protogo.CDMMessage) error {
	cm.receiveChanLock.Lock()
	defer cm.receiveChanLock.Unlock()
	cm.logger.Debugf("register receive chan for [%s]", txId)

	_, ok := cm.receiveChMap[txId]
	if ok {
		cm.logger.Errorf("[%s] fail to register receive chan cause chan already registered", txId)
		return utils.ErrDuplicateTxId
	}

	cm.receiveChMap[txId] = receiveCh
	return nil
}

func (cm *ClientManager) GetReceiveChan(txId string) chan *protogo.CDMMessage {
	cm.receiveChanLock.RLock()
	defer cm.receiveChanLock.RUnlock()
	cm.logger.Debugf("get receive chan for [%s]", txId)
	return cm.receiveChMap[txId]
}

func (cm *ClientManager) GetAndDeleteReceiveChan(txId string) chan *protogo.CDMMessage {
	cm.receiveChanLock.Lock()
	defer cm.receiveChanLock.Unlock()
	cm.logger.Debugf("get receive chan for [%s] and delete", txId)
	receiveChan, ok := cm.receiveChMap[txId]
	if ok {
		delete(cm.receiveChMap, txId)
		return receiveChan
	}
	cm.logger.Warnf("cannot find receive chan for [%s] and return nil", txId)
	return nil
}

func (cm *ClientManager) DeleteReceiveChan(txId string) bool {
	cm.receiveChanLock.Lock()
	defer cm.receiveChanLock.Unlock()
	cm.logger.Debugf("[%s] delete receive chan", txId)
	_, ok := cm.receiveChMap[txId]
	if ok {
		delete(cm.receiveChMap, txId)
		return true
	}
	cm.logger.Debugf("[%s] delete receive chan fail, receive chan is already deleted", txId)
	return false
}

func (cm *ClientManager) listen() {
	cm.logger.Infof("client manager begin listen event")
	for {
		select {
		case <-cm.stopAllSend:
		case <-cm.stopAllReceive:
		case event := <-cm.eventCh:
			switch event.eventType {
			case connectionIncrease:
				err := cm.increaseConnection()
				if err == utils.ErrClientReachLimit {
					cm.logger.Warnf("client already reach limit")
					continue
				}
				if err != nil {
					// todo: stop all
					cm.logger.Errorf("fail to increase new client: %s", err)
				}
			case connectionStopped:
				cm.dropConnection(event)
			default:
				cm.logger.Warnf("unknown event: %s", event)
			}

		}
	}
}

func (cm *ClientManager) increaseConnection() error {
	cm.logger.Debugf("increase new connection")
	if len(cm.aliveClientMap) >= maxClientNum {
		return utils.ErrClientReachLimit
	}
	newIndex := cm.getNextIndex()
	newClient := NewCDMClient(newIndex, cm.chainId, cm.logger, cm)
	if err := newClient.StartClient(); err != nil {
		return err
	}
	cm.aliveClientMap[newIndex] = newClient
	if len(cm.aliveClientMap) == maxClientNum {
		cm.aliveClientReachLimit = true
	}
	return nil
}

func (cm *ClientManager) dropConnection(event *Event) {
	cm.logger.Debugf("drop connection: %d", event.id)
	_, ok := cm.aliveClientMap[event.id]
	if ok {
		delete(cm.aliveClientMap, event.id)
		return
	}
}

func (cm *ClientManager) getNextIndex() uint64 {
	curIndex := cm.index
	cm.index++
	return curIndex
}
