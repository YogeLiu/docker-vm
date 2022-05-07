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
	CalledSameContractError        = errors.New("called same contract")
	ExceedMaxDepthError            = errors.New("exceed max depth")
	ContractFileError              = errors.New("bad contract file")
	ContractExecError              = errors.New("bad contract exec file")
	ContractNotDeployedError       = errors.New("contract not deployed")
	CrossContractRuntimePanicError = errors.New("cross contract runtime panic")

	RuntimePanicError        = errors.New("runtime panic")
	TxTimeoutPanicError      = errors.New("tx time out")
	CrossTxTimeoutPanicError = errors.New("cross tx time out")

	RegisterProcessError = errors.New("fail to register process")
)
