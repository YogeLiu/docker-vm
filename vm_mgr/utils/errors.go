package utils

import "errors"

var (
	// cross contract errors

	MissingContractNameError       = errors.New("missing contract name")
	MissingContractVersionError    = errors.New("missing contact version")
	ExceedMaxDepthError            = errors.New("exceed max depth")
	ContractNotDeployedError       = errors.New("contract not deployed")
	CrossContractRuntimePanicError = errors.New("cross contract runtime panic")

	RuntimePanicError   = errors.New("runtime panic")
	TxTimeoutPanicError = errors.New("tx time out")

	RegisterProcessError = errors.New("fail to register process")
)
