/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hash

import (
	"crypto/sha256"
	"fmt"
	"hash"

	"chainmaker.org/chainmaker/common/v2/crypto"
	"github.com/tjfoc/gmsm/sm3"
	"golang.org/x/crypto/sha3"
)

type Hash struct {
	hashType crypto.HashType
}

func (h *Hash) Get(data []byte) ([]byte, error) {
	var hf func() hash.Hash

	switch h.hashType {
	case crypto.HASH_TYPE_SM3:
		hf = sm3.New
	case crypto.HASH_TYPE_SHA256:
		hf = sha256.New
	case crypto.HASH_TYPE_SHA3_256:
		hf = sha3.New256
	default:
		return nil, fmt.Errorf("unknown hash algorithm")
	}

	f := hf()

	f.Write(data)
	return f.Sum(nil), nil
}

func Get(hashType crypto.HashType, data []byte) ([]byte, error) {
	h := Hash{
		hashType: hashType,
	}
	return h.Get(data)
}

func GetByStrType(hashType string, data []byte) ([]byte, error) {
	h := Hash{
		hashType: crypto.HashAlgoMap[hashType],
	}
	return h.Get(data)
}

func GetHashAlgorithm(hashType crypto.HashType) (hash.Hash, error) {
	switch hashType {
	case crypto.HASH_TYPE_SM3:
		return sm3.New(), nil
	case crypto.HASH_TYPE_SHA256:
		return sha256.New(), nil
	case crypto.HASH_TYPE_SHA3_256:
		return sha3.New256(), nil
	default:
		return nil, fmt.Errorf("unknown hash algorithm")
	}
}
