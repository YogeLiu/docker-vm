package gas

import (
	"errors"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	gasutils "chainmaker.org/chainmaker/utils/v2/gas"
)

const (
	blockVersion2312 = uint32(2030102)
)

// GetSenderAddressGasUsed returns get sender address gas used
func GetSenderAddressGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * GetSenderAddressGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

// CreateKeyHistoryIterGasUsed returns create key history iter gas used
func CreateKeyHistoryIterGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KeyHistoryIterCreateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

// ConsumeKeyHistoryIterGasUsed returns consume key history iter gas used
func ConsumeKeyHistoryIterGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KeyHistoryIterHasNextGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

// CreateKvIteratorGasUsed create kv iter gas used
func CreateKvIteratorGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KvIteratorCreateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

// ConsumeKvIteratorGasUsed returns kv iter gas used
func ConsumeKvIteratorGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KvIteratorNextGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}

	return gasUsed, nil
}

// PutStateGasUsed returns put state gas used
func PutStateGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, contractName, key, field string, value []byte) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return PutStateGasUsedLt2312(gasUsed, contractName, key, field, value)
	}

	return PutStateGasUsed2312(gasConfig, gasUsed, contractName, key, field, value)
}

// GetStateGasUsed returns put state gas used
func GetStateGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, value []byte) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return GetStateGasUsedLt2312(gasUsed, value)
	}

	return GetStateGasUsed2312(gasConfig, gasUsed, value)
}

// GetBatchStateGasUsed returns get batch state gas used
func GetBatchStateGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, payload []byte) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return GetBatchStateGasUsedLt2312(gasUsed, payload)
	}

	return GetBatchStateGasUsed2312(gasConfig, gasUsed, payload)
}

// EmitEventGasUsed returns emit event gas used
func EmitEventGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, contractEvent *common.ContractEvent) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return EmitEventGasUsedLt2312(gasUsed, contractEvent)
	}

	return EmitEventGasUsed2312(gasConfig, gasUsed, contractEvent)
}
