package utils

import "errors"

var (
	ErrDuplicateTxId    = errors.New("duplicate txId")
	ErrMissingByteCode  = errors.New("missing bytecode")
	ErrClientReachLimit = errors.New("clients reach limit")
)
