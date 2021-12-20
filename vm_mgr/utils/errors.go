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
	ContractNotDeployedError       = errors.New("contract not deployed")
	CrossContractRuntimePanicError = errors.New("cross contract runtime panic")

	RuntimePanicError   = errors.New("runtime panic")
	TxTimeoutPanicError = errors.New("tx time out")
)
