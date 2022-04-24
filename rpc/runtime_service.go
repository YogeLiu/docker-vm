package rpc

import (
	"google.golang.org/grpc/status"
	"io"
	"sync"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"

	"google.golang.org/grpc/codes"
)

var runtimeServiceInstance = NewRuntimeService()

func GetRuntimeServiceInstance() *RuntimeService {
	return runtimeServiceInstance
}

type RuntimeService struct {
	lock               sync.RWMutex
	logger             *logger.CMLogger
	stream             protogo.DockerVMRpc_DockerVMCommunicateServer
	sandboxMsgNotifies map[string]func(msg *protogo.DockerVMMessage, sendMsg func(msg *protogo.DockerVMMessage))
	stopSend           chan struct{}
	stopReceive        chan struct{}
	wg                 *sync.WaitGroup
}

func NewRuntimeService() *RuntimeService {
	return &RuntimeService{
		lock:               sync.RWMutex{},
		sandboxMsgNotifies: make(map[string]func(msg *protogo.DockerVMMessage, sendMsg func(msg *protogo.DockerVMMessage)), 1000),
		stopSend:           make(chan struct{}, 1),
		stopReceive:        make(chan struct{}, 1),
		wg:                 &sync.WaitGroup{},
	}
}

func (s *RuntimeService) DockerVMCommunicate(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error {
	sendResponseCh := make(chan *protogo.DockerVMMessage, 1)
	sendMessage := func(msg *protogo.DockerVMMessage) {
		sendResponseCh <- msg
	}

	s.wg.Add(2)

	s.recvRoutine(stream, sendMessage)
	s.sendRoutine(stream, sendResponseCh)

	s.wg.Wait()
	return nil
}

func (s *RuntimeService) recvRoutine(stream protogo.DockerVMRpc_DockerVMCommunicateServer, sendF func(msg *protogo.DockerVMMessage)) {
	s.logger.Infof("start receiving sandbox message")

	for {
		select {
		case <-s.stopReceive:
			s.logger.Debugf("stop runtime server receive goroutine")
			s.wg.Done()
			return
		default:
			receivedMsg, recvErr := stream.Recv()

			// 客户端断开连接时会接收到该错误
			if recvErr == io.EOF {
				s.logger.Error("runtime service eof and exit receive goroutine")
				close(s.stopSend)
				s.wg.Done()
				return
			}

			if recvErr != nil {
				s.logger.Errorf("runtime service err and exit receive goroutine %s", recvErr)
				close(s.stopSend)
				s.wg.Done()
				return
			}

			s.logger.Debugf("runtime server recv msg [%s]", receivedMsg)

			switch receivedMsg.Type {
			case protogo.DockerVMType_TX_RESPONSE,
				protogo.DockerVMType_CALL_CONTRACT_RESPONSE,
				protogo.DockerVMType_CALL_CONTRACT_REQUEST,
				protogo.DockerVMType_GET_STATE_REQUEST,
				protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST,
				protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST,
				protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST,
				protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST,
				protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
				s.getNotify(receivedMsg.TxId)(receivedMsg, sendF)
			}
		}
	}

}

func (s *RuntimeService) sendRoutine(stream protogo.DockerVMRpc_DockerVMCommunicateServer, sendCh chan *protogo.DockerVMMessage) {
	for {
		select {
		case msg := <-sendCh:
			if err := stream.Send(msg); err != nil {
				errStatus, _ := status.FromError(err)
				s.logger.Errorf("fail to send msg: err: %s, err message: %s, err code: %s",
					err, errStatus.Message(), errStatus.Code())
				if errStatus.Code() != codes.ResourceExhausted {
					close(s.stopReceive)
					s.wg.Done()
					return
				}
			}
		case <-s.stopSend:
			s.wg.Done()
			s.logger.Debugf("stop runtime server send goroutine")
			return
		}
	}
}

func (s *RuntimeService) RegisterNotifyForSandbox(txKey string,
	respNotify func(msg *protogo.DockerVMMessage, sendF func(*protogo.DockerVMMessage))) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.logger.Debugf("register receive respNotify for [%s]", txKey)
	_, ok := s.sandboxMsgNotifies[txKey]
	if ok {
		s.logger.Errorf("[%s] fail to register respNotify cause ")
	}
	s.sandboxMsgNotifies[txKey] = respNotify
	return nil
}

func (s *RuntimeService) getNotify(txKey string) func(msg *protogo.DockerVMMessage, f func(msg *protogo.DockerVMMessage)) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	s.logger.Debugf("get notify for [%s]", txKey)
	return s.sandboxMsgNotifies[txKey]
}

func (s *RuntimeService) getAndDeleteNotify(txKey string) func(msg *protogo.DockerVMMessage, f func(msg *protogo.DockerVMMessage)) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.logger.Debugf("get respNotify for [%s] and delete", txKey)
	respNotify, ok := s.sandboxMsgNotifies[txKey]
	if !ok {
		s.logger.Warnf("cannot find respNotify for [%s] and return nil", txKey)
		return nil
	}
	delete(s.sandboxMsgNotifies, txKey)
	return respNotify
}
