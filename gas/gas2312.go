package gas

import (
	"errors"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	gasutils "chainmaker.org/chainmaker/utils/v2/gas"
)

// PutStateGasUsed2312 returns put state gas used
func PutStateGasUsed2312(gasConfig *gasutils.GasConfig,
	gasUsed uint64, contractName, key, field string, value []byte) (uint64, error) {
	putStateGasPrice := float32(PutStateGasPrice)
	if gasConfig != nil {
		putStateGasPrice = gasConfig.GetGasPriceForInvoke()
	}

	dataSize := len(value) + len(contractName) + len(key) + len(field)
	gas, err := gasutils.MultiplyGasPrice(dataSize, putStateGasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

// GetStateGasUsed2312 returns get state gas used
func GetStateGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64,
	contractName, key, field string, value []byte) (uint64, error) {

	getStateGasPrice := float32(GetStateGasPrice)
	if gasConfig != nil {
		getStateGasPrice = gasConfig.GetGasPriceForInvoke()
	}

	dataSize := len(contractName) + len(key) + len(field)
	gas, err := gasutils.MultiplyGasPrice(dataSize, getStateGasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

// GetBatchStateGasUsed2312 returns get batch state gas used
func GetBatchStateGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64, payload []byte) (uint64, error) {
	getBatchStateGasPrice := float32(GetBatchStateGasPrice)
	if gasConfig != nil {
		getBatchStateGasPrice = gasConfig.GetGasPriceForInvoke()
	}

	gas, err := gasutils.MultiplyGasPrice(len(payload), getBatchStateGasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

// EmitEventGasUsed2312 returns emit event gas used
func EmitEventGasUsed2312(gasConfig *gasutils.GasConfig,
	gasUsed uint64, contractEvent *common.ContractEvent) (uint64, error) {

	emitEventGasPrice := float32(EmitEventGasPrice)
	if gasConfig != nil {
		emitEventGasPrice = gasConfig.GetGasPriceForInvoke()
	}

	dataSize := len(contractEvent.Topic) + len(contractEvent.ContractName) + len(contractEvent.ContractVersion)
	for _, event := range contractEvent.EventData {
		dataSize += len(event)
	}
	gas, err := gasutils.MultiplyGasPrice(dataSize, emitEventGasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func CreateKeyHistoryIterGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64) (uint64, error) {
	gasPrice := float32(KeyHistoryIterCreateGasPrice)
	if gasConfig != nil {
		gasPrice = gasConfig.GetGasPriceForInvoke()
	}

	gas, err := gasutils.MultiplyGasPrice(10, gasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func CreateKvIteratorGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64) (uint64, error) {
	gasPrice := float32(KvIteratorCreateGasPrice)
	if gasConfig != nil {
		gasPrice = gasConfig.GetGasPriceForInvoke()
	}

	gas, err := gasutils.MultiplyGasPrice(10, gasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func ConsumeKeyHistoryIterGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64) (uint64, error) {
	gasPrice := float32(KeyHistoryIterHasNextGasPrice)
	if gasConfig != nil {
		gasPrice = gasConfig.GetGasPriceForInvoke()
	}

	gas, err := gasutils.MultiplyGasPrice(10, gasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func ConsumeKvIteratorGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64) (uint64, error) {
	gasPrice := float32(KvIteratorNextGasPrice)
	if gasConfig != nil {
		gasPrice = gasConfig.GetGasPriceForInvoke()
	}

	gas, err := gasutils.MultiplyGasPrice(10, gasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func ConsumeKvIteratorNextGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64,
	key string, field string, value []byte) (uint64, error) {

	gasPrice := float32(0)
	if gasConfig != nil {
		gasPrice = gasConfig.GetGasPriceForInvoke()
	}

	dataSize := len(key) + len(field) + len(value)
	gas, err := gasutils.MultiplyGasPrice(dataSize, gasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func ConsumeKeyHistoryIterNextGasUsed2312(gasConfig *gasutils.GasConfig, gasUsed uint64,
	value []byte) (uint64, error) {

	gasPrice := float32(0)
	if gasConfig != nil {
		gasPrice = gasConfig.GetGasPriceForInvoke()
	}

	dataSize := len(value)
	gas, err := gasutils.MultiplyGasPrice(dataSize, gasPrice)
	if err != nil {
		return 0, err
	}

	gasUsed += gas
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}
