package utils

import "errors"

var (
	ErrDuplicateTxId   = errors.New("duplicate txId")
	ErrMissingByteCode = errors.New("missing bytecode")
)
