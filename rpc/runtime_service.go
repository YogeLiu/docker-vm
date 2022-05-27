package rpc

import (
	"io"
	"sync"

	"go.uber.org/atomic"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	runtimeServiceOnce     sync.Once
	runtimeServiceInstance *RuntimeService
)

type RuntimeService struct {
	streamCounter atomic.Uint64
	chainId       string
	lock          sync.RWMutex
	logger        protocol.Logger
	//stream           protogo.DockerVMRpc_DockerVMCommunicateServer
	sandboxMsgNotify map[string]func(msg *protogo.DockerVMMessage, sendMsg func(msg *protogo.DockerVMMessage))
	//responseChanMap  map[uint64]chan *protogo.DockerVMMessage
	responseChanMap sync.Map
}

func NewRuntimeService(chainId string, logger protocol.Logger) *RuntimeService {
	runtimeServiceOnce.Do(func() {
		runtimeServiceInstance = &RuntimeService{
			streamCounter: atomic.Uint64{},
			chainId:       chainId,
			lock:          sync.RWMutex{},
			logger:        logger,
			sandboxMsgNotify: make(
				map[string]func(
					msg *protogo.DockerVMMessage,
					sendMsg func(msg *protogo.DockerVMMessage),
				),
				1000,
			),
			responseChanMap: sync.Map{},
		}
	})
	return runtimeServiceInstance
}

func (s *RuntimeService) getStreamId() uint64 {
	s.streamCounter.Add(1)
	return s.streamCounter.Load()
}

func (s *RuntimeService) registerStreamSendCh(streamId uint64, sendCh chan *protogo.DockerVMMessage) bool {
	s.logger.Debugf("register send chan for stream[%d]", streamId)
	if _, ok := s.responseChanMap.Load(streamId); ok {
		s.logger.Debugf("[%d] fail to register receive chan cause chan already registered", streamId)
		return false
	}
	s.responseChanMap.Store(streamId, sendCh)
	return true
}

// nolint: unused
func (s *RuntimeService) getStreamSendCh(streamId uint64) chan *protogo.DockerVMMessage {
	s.logger.Debugf("get send chan for stream[%d]", streamId)
	ch, ok := s.responseChanMap.Load(streamId)
	if !ok {
		return nil
	}

	return ch.(chan *protogo.DockerVMMessage)
}

func (s *RuntimeService) deleteStreamSendCh(streamId uint64) {
	s.logger.Debugf("delete send chan for stream[%d]", streamId)
	s.responseChanMap.Delete(streamId)
}

type serviceStream struct {
	logger         protocol.Logger
	streamId       uint64
	stream         protogo.DockerVMRpc_DockerVMCommunicateServer
	sendResponseCh chan *protogo.DockerVMMessage
	stopSend       chan struct{}
	stopReceive    chan struct{}
	wg             *sync.WaitGroup
}

func (ss *serviceStream) putResp(msg *protogo.DockerVMMessage) {
	ss.logger.Debugf("put sys_call response to send chan, txId [%s], type [%s]", msg.TxId, msg.Type)
	ss.sendResponseCh <- msg
}

func (s *RuntimeService) DockerVMCommunicate(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error {
	ss := &serviceStream{
		logger:         s.logger,
		streamId:       s.getStreamId(),
		stream:         stream,
		sendResponseCh: make(chan *protogo.DockerVMMessage, 1),
		stopSend:       make(chan struct{}, 1),
		stopReceive:    make(chan struct{}, 1),
		wg:             &sync.WaitGroup{},
	}
	defer s.deleteStreamSendCh(ss.streamId)

	s.registerStreamSendCh(ss.streamId, ss.sendResponseCh)

	ss.wg.Add(2)

	go s.recvRoutine(ss)
	go s.sendRoutine(ss)

	ss.wg.Wait()
	return nil
}

func (s *RuntimeService) recvRoutine(ss *serviceStream) {
	s.logger.Infof("start receiving sandbox message")

	for {
		select {
		case <-ss.stopReceive:
			s.logger.Debugf("stop runtime server receive goroutine")
			ss.wg.Done()
			return
		default:
			receivedMsg, recvErr := ss.stream.Recv()

			// 客户端断开连接时会接收到该错误
			if recvErr == io.EOF {
				s.logger.Debugf("runtime service eof and exit receive goroutine")
				close(ss.stopSend)
				ss.wg.Done()
				return
			}

			if recvErr != nil {
				s.logger.Debugf("runtime service err and exit receive goroutine %s", recvErr)
				close(ss.stopSend)
				ss.wg.Done()
				return
			}

			s.logger.Debugf("runtime server recveive msg, txId [%s], type [%s]", receivedMsg.TxId, receivedMsg.Type)

			switch receivedMsg.Type {
			case protogo.DockerVMType_TX_RESPONSE,
				protogo.DockerVMType_CALL_CONTRACT_REQUEST,
				protogo.DockerVMType_GET_STATE_REQUEST,
				protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST,
				protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST,
				protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST,
				protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST,
				protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
				notify := s.getNotify(s.chainId, receivedMsg.TxId)
				if notify == nil {
					s.logger.Debugf("get receive notify[%s] failed, please check your key", receivedMsg.TxId)
					break
				}
				notify(receivedMsg, ss.putResp)
			}
		}
	}

}

func (s *RuntimeService) sendRoutine(ss *serviceStream) {
	s.logger.Debugf("start sending sys_call response")
	for {
		select {
		case msg := <-ss.sendResponseCh:
			s.logger.Debugf("get sys_call response from send chan, send to sandbox, txId [%s], type [%s]", msg.TxId, msg.Type)
			if err := ss.stream.Send(msg); err != nil {
				errStatus, _ := status.FromError(err)
				s.logger.Errorf("fail to send msg: err: %s, err message: %s, err code: %s",
					err, errStatus.Message(), errStatus.Code())
				if errStatus.Code() != codes.ResourceExhausted {
					close(ss.stopReceive)
					ss.wg.Done()
					return
				}
			}
		case <-ss.stopSend:
			ss.wg.Done()
			s.logger.Debugf("stop runtime server send goroutine")
			return
		}
	}
}

func (s *RuntimeService) RegisterSandboxMsgNotify(chainId, txKey string,
	respNotify func(msg *protogo.DockerVMMessage, sendF func(*protogo.DockerVMMessage))) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txKey)
	s.logger.Debugf("register receive respNotify for [%s]", notifyKey)
	_, ok := s.sandboxMsgNotify[notifyKey]
	if ok {
		s.logger.Errorf("[%s] fail to register respNotify cause ")
	}
	s.sandboxMsgNotify[notifyKey] = respNotify
	return nil
}

func (s *RuntimeService) getNotify(chainId, txId string) func(msg *protogo.DockerVMMessage, f func(msg *protogo.DockerVMMessage)) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txId)
	s.logger.Debugf("get notify for [%s]", notifyKey)
	return s.sandboxMsgNotify[notifyKey]
}

func (s *RuntimeService) DeleteSandboxMsgNotify(chainId, txId string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	notifyKey := utils.ConstructNotifyMapKey(chainId, txId)
	s.logger.Debugf("[%s] delete notify", txId)
	_, ok := s.sandboxMsgNotify[notifyKey]
	if !ok {
		s.logger.Debugf("[%s] delete notify fail, notify is already deleted", notifyKey)
		return false
	}
	delete(s.sandboxMsgNotify, notifyKey)
	return true
}
