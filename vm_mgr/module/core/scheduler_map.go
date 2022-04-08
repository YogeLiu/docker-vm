package core

import (
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"go.uber.org/zap"
)

type SchedulerMap struct {
	logger         *zap.SugaredLogger
	processManager *ProcessManager

	schedulerLock     sync.Mutex
	chainSchedulerMap map[string]*DockerScheduler

	txReqCh chan *protogo.TxRequest
}

// NewDockerScheduler new docker scheduler
func NewSchedulerMap(processManager *ProcessManager) *SchedulerMap {
	scheduler := &SchedulerMap{
		logger:         logger.NewDockerLogger(logger.MODULE_SCHEDULER, config.DockerLogDir),
		processManager: processManager,

		chainSchedulerMap: make(map[string]*DockerScheduler),

		txReqCh: make(chan *protogo.TxRequest, ReqChanSize),
	}

	return scheduler
}

func (s *SchedulerMap) listenIncomingTxRequest() {
	s.logger.Debugf("start listen incoming tx request")

	for {
		txRequest := <-s.txReqCh
		go s.handleTx(txRequest)
	}
}

func (s *SchedulerMap) handleTx(txRequest *protogo.TxRequest) {
	s.logger.Debugf("[%s] docker scheduler handle tx", txRequest.TxId)
	err := s.processManager.AddTx(txRequest)
	if err == utils.ContractFileError {
		s.logger.Errorf("failed to add tx, err is :%s, txId: %s",
			err, txRequest.TxId)
		s.ReturnErrorResponse(txRequest.ChainId, txRequest.TxId, err.Error())
		return
	}
	if err != nil {
		s.logger.Warnf("add tx warning: err is :%s, txId: %s",
			err, txRequest.TxId)
		return
	}
}

// StartScheduler start docker scheduler
func (s *SchedulerMap) StartScheduler() {

	s.logger.Debugf("start docker scheduler")

	go s.listenIncomingTxRequest()

}

func (s *SchedulerMap) ReturnErrorResponse(chainId, txId string, errMsg string) {
	errTxResponse := s.constructErrorResponse(txId, errMsg)
	s.getSchedulerByChainId(chainId).txResponseCh <- errTxResponse
}

func (s *SchedulerMap) constructErrorResponse(txId string, errMsg string) *protogo.TxResponse {
	return &protogo.TxResponse{
		TxId:    txId,
		Code:    protogo.ContractResultCode_FAIL,
		Result:  nil,
		Message: errMsg,
	}
}

// GetTxReqCh get tx request chan
func (s *SchedulerMap) GetTxReqCh(chainId string) chan *protogo.TxRequest {
	return s.txReqCh
}

// GetTxResponseCh get tx response ch
func (s *SchedulerMap) GetTxResponseCh(chainId string) chan *protogo.TxResponse {
	return s.getSchedulerByChainId(chainId).txResponseCh
}

// GetGetStateReqCh retrieve get state request chan
func (s *SchedulerMap) GetGetStateReqCh(chainId string) chan *protogo.CDMMessage {
	return s.getSchedulerByChainId(chainId).getStateReqCh
}

// GetCrossContractReqCh get cross contract request chan
func (s *SchedulerMap) GetCrossContractReqCh(chainId string) chan *protogo.TxRequest {
	return s.getSchedulerByChainId(chainId).crossContractsCh
}

// GetByteCodeReqCh get bytecode request chan
func (s *SchedulerMap) GetByteCodeReqCh(chainId string) chan *protogo.CDMMessage {
	return s.getSchedulerByChainId(chainId).getByteCodeReqCh
}

// RegisterResponseCh register response chan
func (s *SchedulerMap) RegisterResponseCh(chainId string, responseId string, responseCh chan *protogo.CDMMessage) {
	s.getSchedulerByChainId(chainId).responseChMap.Store(responseId, responseCh)
}

// GetResponseChByTxId get response chan by tx id
func (s *SchedulerMap) GetResponseChByTxId(chainId string, txId string) chan *protogo.CDMMessage {

	responseCh, _ := s.getSchedulerByChainId(chainId).responseChMap.LoadAndDelete(txId)
	return responseCh.(chan *protogo.CDMMessage)
}

// RegisterCrossContractResponseCh register cross contract response chan
func (s *SchedulerMap) RegisterCrossContractResponseCh(chainId string, responseId string, responseCh chan *SDKProtogo.DMSMessage) {
	s.getSchedulerByChainId(chainId).responseChMap.Store(responseId, responseCh)
}

// GetCrossContractResponseCh get cross contract response chan
func (s *SchedulerMap) GetCrossContractResponseCh(chainId string, responseId string) chan *SDKProtogo.DMSMessage {
	responseCh, loaded := s.getSchedulerByChainId(chainId).responseChMap.LoadAndDelete(responseId)
	if !loaded {
		return nil
	}
	return responseCh.(chan *SDKProtogo.DMSMessage)
}

func (s *SchedulerMap) ReturnErrorCrossContractResponse(crossContractTx *protogo.TxRequest,
	errResponse *SDKProtogo.DMSMessage) {

	responseChId := crossContractChKey(crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
	responseCh := s.getSchedulerByChainId(crossContractTx.ChainId).GetCrossContractResponseCh(responseChId)
	if responseCh == nil {
		s.logger.Warnf("scheduler fail to get response chan and abandon cross err response [%s]",
			errResponse.TxId)
		return
	}
	responseCh <- errResponse
}

// GetCrossContractResponseCh get cross contract response chan
func (s *SchedulerMap) getSchedulerByChainId(chainId string) *DockerScheduler {

	scheduler, ok := s.chainSchedulerMap[chainId]
	if !ok {
		scheduler = s.createChainScheduler(chainId)
	}

	return scheduler
}

func (s *SchedulerMap) createChainScheduler(chainId string) *DockerScheduler {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	scheduler, ok := s.chainSchedulerMap[chainId]
	if !ok {
		scheduler = NewDockerScheduler()
		s.chainSchedulerMap[chainId] = scheduler
	}
	return scheduler
}
