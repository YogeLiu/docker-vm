/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto"
)

const (
	// 密码算法默认值，若是此项，将采用配置文件中配置的密码算法
	CRYPTO_ALGO_HASH_DEFAULT = ""
	CRYPTO_ALGO_SYM_DEFAULT  = ""
	CRYPTO_ALGO_ASYM_DEFAULT = ""

	// 哈希算法
	CRYPTO_ALGO_SHA256   = "SHA256"
	CRYPTO_ALGO_SHA3_256 = "SHA3_256"
	CRYPTO_ALGO_SM3      = "SM3"

	// 对称加密
	CRYPTO_ALGO_AES    = "AES"
	CRYPTO_ALGO_AES128 = "AES128"
	CRYPTO_ALGO_AES192 = "AES192"
	CRYPTO_ALGO_AES256 = "AES256"
	CRYPTO_ALGO_SM4    = "SM4"

	// 非对称秘钥
	CRYPTO_ALGO_RSA512        = "RSA512"
	CRYPTO_ALGO_RSA1024       = "RSA1024"
	CRYPTO_ALGO_RSA2048       = "RSA2048"
	CRYPTO_ALGO_RSA3072       = "RSA3072"
	CRYPTO_ALGO_SM2           = "SM2"
	CRYPTO_ALGO_ECC_P256      = "ECC_P256"
	CRYPTO_ALGO_ECC_P384      = "ECC_P384"
	CRYPTO_ALGO_ECC_P521      = "ECC_P521"
	CRYPTO_ALGO_ECC_Ed25519   = "ECC_Ed25519"
	CRYPTO_ALGO_ECC_Secp256k1 = "ECC_Secp256k1"
)

type HashType uint

const (
	HASH_TYPE_SM3      HashType = 20
	HASH_TYPE_SHA256   HashType = HashType(crypto.SHA256)
	HASH_TYPE_SHA3_256 HashType = HashType(crypto.SHA3_256)
)

const (
	SM3 = crypto.Hash(HASH_TYPE_SM3)
)

// constant UID for SM2-SM3
const CRYPTO_DEFAULT_UID = "1234567812345678"

// 秘钥类型
type KeyType int

const (
	// 对称秘钥
	AES KeyType = iota
	SM4
	// 非对称秘钥
	RSA512
	RSA1024
	RSA2048
	RSA3072
	SM2
	ECC_Secp256k1
	ECC_NISTP256
	ECC_NISTP384
	ECC_NISTP521
	ECC_Ed25519
)

var KeyType2NameMap = map[KeyType]string{
	AES:           CRYPTO_ALGO_AES,
	SM4:           CRYPTO_ALGO_SM4,
	RSA512:        CRYPTO_ALGO_RSA512,
	RSA1024:       CRYPTO_ALGO_RSA1024,
	RSA2048:       CRYPTO_ALGO_RSA2048,
	RSA3072:       CRYPTO_ALGO_RSA3072,
	SM2:           CRYPTO_ALGO_SM2,
	ECC_Secp256k1: CRYPTO_ALGO_ECC_Secp256k1,
	ECC_NISTP256:  "ECC_NISTP256",
	ECC_NISTP384:  "ECC_NISTP384",
	ECC_NISTP521:  "ECC_NISTP521",
	ECC_Ed25519:   CRYPTO_ALGO_ECC_Ed25519,
}
var Name2KeyTypeMap = map[string]KeyType{
	CRYPTO_ALGO_AES:           AES,
	CRYPTO_ALGO_SM4:           SM4,
	CRYPTO_ALGO_RSA512:        RSA512,
	CRYPTO_ALGO_RSA1024:       RSA1024,
	CRYPTO_ALGO_RSA2048:       RSA2048,
	CRYPTO_ALGO_RSA3072:       RSA3072,
	CRYPTO_ALGO_SM2:           SM2,
	CRYPTO_ALGO_ECC_Secp256k1: ECC_Secp256k1,
	"ECC_NISTP256":            ECC_NISTP256,
	"ECC_NISTP384":            ECC_NISTP384,
	"ECC_NISTP521":            ECC_NISTP521,
	CRYPTO_ALGO_ECC_Ed25519:   ECC_Ed25519,
}

type BitsSize int

const (
	BITS_SIZE_128  BitsSize = 128
	BITS_SIZE_192  BitsSize = 192
	BITS_SIZE_256  BitsSize = 256
	BITS_SIZE_512  BitsSize = 512
	BITS_SIZE_1024 BitsSize = 1024
	BITS_SIZE_2048 BitsSize = 2048
	BITS_SIZE_3072 BitsSize = 3072
)

var HashAlgoMap = map[string]HashType{
	CRYPTO_ALGO_SHA256:   HASH_TYPE_SHA256,
	CRYPTO_ALGO_SHA3_256: HASH_TYPE_SHA3_256,
	CRYPTO_ALGO_SM3:      HASH_TYPE_SM3,
}

var SymAlgoMap = map[string]KeyType{
	// 对称秘钥
	CRYPTO_ALGO_AES:    AES,
	CRYPTO_ALGO_AES128: AES,
	CRYPTO_ALGO_AES192: AES,
	CRYPTO_ALGO_AES256: AES,
	CRYPTO_ALGO_SM4:    SM4,
}

var AsymAlgoMap = map[string]KeyType{
	// 非对称秘钥
	CRYPTO_ALGO_RSA512:        RSA512,
	CRYPTO_ALGO_RSA1024:       RSA1024,
	CRYPTO_ALGO_RSA2048:       RSA2048,
	CRYPTO_ALGO_RSA3072:       RSA3072,
	CRYPTO_ALGO_SM2:           SM2,
	CRYPTO_ALGO_ECC_P256:      ECC_NISTP256,
	CRYPTO_ALGO_ECC_P384:      ECC_NISTP384,
	CRYPTO_ALGO_ECC_P521:      ECC_NISTP521,
	CRYPTO_ALGO_ECC_Ed25519:   ECC_Ed25519,
	CRYPTO_ALGO_ECC_Secp256k1: ECC_Secp256k1,
}

// Signing options
type SignOpts struct {
	Hash         HashType
	UID          string
	EncodingType string
}

// Encryption options
type EncOpts struct {
	EncodingType string
	BlockMode    string
	EnableMAC    bool
	Hash         HashType
	Label        []byte
	EnableASN1   bool
}

// === 秘钥接口 ===
type Key interface {
	// 获取秘钥字节数组
	Bytes() ([]byte, error)

	// 获取秘钥类型
	Type() KeyType

	// 获取编码后秘钥(PEM格式)
	String() (string, error)
}

// === 对称秘钥加解密接口 ===
type SymmetricKey interface {
	Key

	// 加密接口
	Encrypt(plain []byte) ([]byte, error)
	EncryptWithOpts(plain []byte, opts *EncOpts) ([]byte, error)

	// 解密接口
	Decrypt(ciphertext []byte) ([]byte, error)
	DecryptWithOpts(ciphertext []byte, opts *EncOpts) ([]byte, error)
}

// === 非对称秘钥签名+验签接口 ===
// 私钥签名接口
type PrivateKey interface {
	Key

	// 私钥签名
	Sign(data []byte) ([]byte, error)

	SignWithOpts(data []byte, opts *SignOpts) ([]byte, error)

	// 返回公钥
	PublicKey() PublicKey

	// 转换为crypto包中的 PrivateKey 接口类
	ToStandardKey() crypto.PrivateKey
}

// 公钥验签接口
type PublicKey interface {
	Key

	// 公钥验签
	Verify(data []byte, sig []byte) (bool, error)

	VerifyWithOpts(data []byte, sig []byte, opts *SignOpts) (bool, error)

	// 转换为crypto包中的 PublicKey 接口类
	ToStandardKey() crypto.PublicKey
}

// Encryption interface

type DecryptKey interface {
	Key

	Decrypt(ciphertext []byte) ([]byte, error)

	DecryptWithOpts(ciphertext []byte, opts *EncOpts) ([]byte, error)

	EncryptKey() EncryptKey
}

type EncryptKey interface {
	Key

	Encrypt(data []byte) ([]byte, error)

	EncryptWithOpts(data []byte, opts *EncOpts) ([]byte, error)
}

type Encryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}
