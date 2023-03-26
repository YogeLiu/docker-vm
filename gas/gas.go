package gas

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	gasutils "chainmaker.org/chainmaker/utils/v2/gas"
)

const (
	blockVersion2312 = uint32(2030102)
)

// PutStateGasUsed returns put state gas used
func PutStateGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, contractName, key, field string, value []byte) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return PutStateGasUsedLt2312(gasUsed, contractName, key, field, value)
	}

	return 0, nil
}

// GetStateGasUsed returns put state gas used
func GetStateGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, value []byte) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return GetStateGasUsedLt2312(gasUsed, value)
	}

	return 0, nil
}

// GetBatchStateGasUsed returns get batch state gas used
func GetBatchStateGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, payload []byte) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return GetBatchStateGasUsedLt2312(gasUsed, payload)
	}

	return 0, nil
}

// EmitEventGasUsed returns emit event gas used
func EmitEventGasUsed(
	blockVersion uint32, gasConfig *gasutils.GasConfig,
	gasUsed uint64, contractEvent *common.ContractEvent) (uint64, error) {

	if blockVersion < blockVersion2312 {
		return EmitEventGasUsedLt2312(gasUsed, contractEvent)
	}

	return 0, nil
}
