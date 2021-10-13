/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hash

import (
	"crypto/sha256"
	"math"

	"chainmaker.org/chainmaker/common/v2/crypto"
)

// nolint: deadcode,unused
var h = sha256.New()

func GetMerkleRoot(hashType string, hashes [][]byte) ([]byte, error) {
	if len(hashes) == 0 {
		return nil, nil
	}

	merkleTree, err := BuildMerkleTree(hashType, hashes)
	if err != nil {
		return nil, err
	}
	return merkleTree[len(merkleTree)-1], nil
}

// take leaf node hash array and build merkle tree
func BuildMerkleTree(hashType string, hashes [][]byte) ([][]byte, error) {
	var hasher = Hash{
		hashType: crypto.HashAlgoMap[hashType],
	}

	var err error
	if len(hashes) == 0 {
		return nil, nil
	}

	// use array to store merkle tree entries
	nextPowOfTwo := getNextPowerOfTwo(len(hashes))
	arraySize := nextPowOfTwo*2 - 1
	merkelTree := make([][]byte, arraySize)

	// 1. copy hashes first
	copy(merkelTree[:len(hashes)], hashes[:])

	// 2. compute merkle step by step
	offset := nextPowOfTwo
	for i := 0; i < arraySize-1; i += 2 {
		switch {
		case merkelTree[i] == nil:
			// parent node is nil if left is nil
			merkelTree[offset] = nil
		case merkelTree[i+1] == nil:
			// hash(left, left) if right is nil
			merkelTree[offset], err = hashMerkleBranches(hasher, merkelTree[i], merkelTree[i])
			if err != nil {
				return nil, err
			}
		default:
			// default hash(left||right)
			merkelTree[offset], err = hashMerkleBranches(hasher, merkelTree[i], merkelTree[i+1])
			if err != nil {
				return nil, err
			}
		}
		offset++
	}

	return merkelTree, nil
}

func GetMerklePath(hashType string, hash []byte, merkleTree [][]byte,
	paths *[][]byte, withRoot bool) (brother []byte, parent []byte) {
	brother, parent = getPath(hashType, hash, merkleTree, withRoot)
	if brother != nil {
		*paths = append(*paths, brother)
		GetMerklePath(hashType, parent, merkleTree, paths, withRoot)
	}
	return brother, parent
}

func getPath(hashType string, hash []byte, merkleTree [][]byte, withRoot bool) (brother []byte, parent []byte) {
	var hasher = Hash{
		hashType: crypto.HashAlgoMap[hashType],
	}

	for i, bytes := range merkleTree {
		if isEqualStr(bytes, hash) {
			if isEvenNum(i) {
				return getEvenHashMerkleBranches(i, hasher, merkleTree, withRoot)
			}
			parent, _ = hashMerkleBranches(hasher, merkleTree[i-1], hash)
			return merkleTree[i-1], parent
		}
	}
	return nil, nil
}

func getEvenHashMerkleBranches(i int, hasher Hash, merkleTree [][]byte, withRoot bool) (brother []byte, parent []byte) {
	if i+1 < len(merkleTree) {
		parent, _ = hashMerkleBranches(hasher, merkleTree[i], merkleTree[i+1])
		return merkleTree[i+1], parent
	} else if withRoot { //root
		parent, _ = hashMerkleBranches(hasher, merkleTree[i], merkleTree[i])
		return merkleTree[i], parent
	}
	return brother, parent
}

func isEvenNum(num int) bool {
	return num&'1' == 0
}

func isEqualStr(dst, src []byte) bool {
	return string(dst) == string(src)
}

func getNextPowerOfTwo(n int) int {
	if n&(n-1) == 0 {
		return n
	}

	exponent := uint(math.Log2(float64(n))) + 1
	return 1 << exponent
}

func hashMerkleBranches(hasher Hash, left []byte, right []byte) ([]byte, error) {
	data := make([]byte, len(left)+len(right))
	copy(data[:len(left)], left)
	copy(data[len(left):], right)
	return hasher.Get(data)
}

func getNextPowerOfTen(n int) (int, int) {
	//if n&(n-1) == 0 {
	//	return n, 0
	//}
	if n == 1 {
		return 1, 0
	}

	exponent := int(math.Log10(float64(n-1))) + 1
	rootsSize := 0
	for i := 0; i < exponent; i++ {
		rootsSize += int(math.Pow10(i))
	}
	return int(math.Pow10(exponent)), rootsSize
}
