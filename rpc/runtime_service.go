package rpc

import (
	"io"
	"sync"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var runtimeServiceOnce sync.Once

type RuntimeService struct {
	chainId          string
	lock             sync.RWMutex
	logger           protocol.Logger
	stream           protogo.DockerVMRpc_DockerVMCommunicateServer
	sandboxMsgNotify map[string]func(msg *protogo.DockerVMMessage, sendMsg func(msg *protogo.DockerVMMessage))
}

func NewRuntimeService(chainId string, logger protocol.Logger) *RuntimeService {
	instanceCh := make(chan *RuntimeService, 1)
	runtimeServiceOnce.Do(func() {
		instanceCh <- &RuntimeService{
			chainId:          chainId,
			lock:             sync.RWMutex{},
			logger:           logger,
			sandboxMsgNotify: make(map[string]func(msg *protogo.DockerVMMessage, sendMsg func(msg *protogo.DockerVMMessage)), 1000),
		}
	})
	return <-instanceCh
}

type serviceStream struct {
	stream      protogo.DockerVMRpc_DockerVMCommunicateServer
	stopSend    chan struct{}
	stopReceive chan struct{}
	wg          *sync.WaitGroup
}

func (s *RuntimeService) DockerVMCommunicate(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error {
	sendResponseCh := make(chan *protogo.DockerVMMessage, 1)
	putResp := func(msg *protogo.DockerVMMessage) {
		s.logger.Debugf("put sys_call response to send chan, txId [%s], type [%s]", msg.TxId, msg.Type)
		sendResponseCh <- msg
	}
	st := &serviceStream{
		stream:      stream,
		stopSend:    make(chan struct{}, 1),
		stopReceive: make(chan struct{}, 1),
		wg:          &sync.WaitGroup{},
	}
	st.wg.Add(2)

	go s.recvRoutine(st, putResp)
	go s.sendRoutine(st, sendResponseCh)

	st.wg.Wait()
	return nil
}

func (s *RuntimeService) recvRoutine(st *serviceStream, putResp func(msg *protogo.DockerVMMessage)) {
	s.logger.Infof("start receiving sandbox message")

	for {
		select {
		case <-st.stopReceive:
			s.logger.Debugf("stop runtime server receive goroutine")
			st.wg.Done()
			return
		default:
			receivedMsg, recvErr := st.stream.Recv()

			// 客户端断开连接时会接收到该错误
			if recvErr == io.EOF {
				s.logger.Error("runtime service eof and exit receive goroutine")
				close(st.stopSend)
				st.wg.Done()
				return
			}

			if recvErr != nil {
				s.logger.Errorf("runtime service err and exit receive goroutine %s", recvErr)
				close(st.stopSend)
				st.wg.Done()
				return
			}

			s.logger.Debugf("runtime server recv msg [%s]", receivedMsg)

			switch receivedMsg.Type {
			case protogo.DockerVMType_TX_RESPONSE,
				protogo.DockerVMType_CALL_CONTRACT_REQUEST,
				protogo.DockerVMType_GET_STATE_REQUEST,
				protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST,
				protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST,
				protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST,
				protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST,
				protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
				s.getNotify(s.chainId, receivedMsg.TxId)(receivedMsg, putResp)
			}
		}
	}

}

func (s *RuntimeService) sendRoutine(st *serviceStream, sendCh chan *protogo.DockerVMMessage) {
	s.logger.Debugf("start sending sys_call response")
	for {
		select {
		case msg := <-sendCh:
			s.logger.Debugf("get sys_call response from send chan, send to sandbox, txId [%s], type [%s]", msg.TxId, msg.Type)
			if err := st.stream.Send(msg); err != nil {
				errStatus, _ := status.FromError(err)
				s.logger.Errorf("fail to send msg: err: %s, err message: %s, err code: %s",
					err, errStatus.Message(), errStatus.Code())
				if errStatus.Code() != codes.ResourceExhausted {
					close(st.stopReceive)
					st.wg.Done()
					return
				}
			}
		case <-st.stopSend:
			st.wg.Done()
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
