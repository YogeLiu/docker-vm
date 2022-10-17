/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package utils

import "errors"

var (
	// cross contract errors
	MissingContractNameError       = errors.New("missing contract name")
	MissingContractVersionError    = errors.New("missing contact version")
	ExceedMaxDepthError            = errors.New("exceed max depth")
	ContractFileError              = errors.New("bad contract file")
	ContractExecError              = errors.New("bad contract exec file")
	ContractNotExistError          = errors.New("contract exec file not exist")
	ContractNotDeployedError       = errors.New("contract not deployed")
	CrossContractRuntimePanicError = errors.New("cross contract runtime panic")

	RuntimePanicError   = errors.New("runtime panic")
	TxTimeoutPanicError = errors.New("tx time out")

	RegisterProcessError = errors.New("fail to register process")

	SandboxExitDefaultError = errors.New("exit with no additional error message")
)
