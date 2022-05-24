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
	ContractNotDeployedError       = errors.New("contract not deployed")
	CrossContractRuntimePanicError = errors.New("cross contract runtime panic")

	RuntimePanicError   = errors.New("runtime panic")
	TxTimeoutPanicError = errors.New("tx timeout")

	RegisterProcessError = errors.New("failed to register process")
)
