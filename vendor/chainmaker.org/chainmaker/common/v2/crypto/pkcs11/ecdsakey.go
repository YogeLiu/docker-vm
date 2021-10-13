/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"crypto"
	"crypto/ecdsa"
	"encoding/asn1"
	"io"

	bcsm2 "chainmaker.org/chainmaker/common/v2/crypto/asym/sm2"
	"github.com/tjfoc/gmsm/sm2"

	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	bcecdsa "chainmaker.org/chainmaker/common/v2/crypto/asym/ecdsa"
	"chainmaker.org/chainmaker/common/v2/crypto/hash"
	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

type ecdsaPrivateKey struct {
	priv *p11EcdsaPrivateKey
}

func (e ecdsaPrivateKey) Public() crypto.PublicKey {
	return e.priv.pubKey.ToStandardKey()
}

func (e ecdsaPrivateKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	return e.priv.Sign(digest)
}

// p11EcdsaPrivateKey represents pkcs11 ecdsa/sm2 private key
type p11EcdsaPrivateKey struct {
	p11Ctx    *P11Handle
	pubKey    bccrypto.PublicKey
	keyId     []byte
	keyType   P11KeyType
	keyObject pkcs11.ObjectHandle

	signer crypto.Signer
}

func NewP11ECDSAPrivateKey(p11 *P11Handle, keyId []byte, keyType P11KeyType) (bccrypto.PrivateKey, error) {
	if p11 == nil || keyId == nil {
		return nil, errors.New("Invalid parameter, p11 or keyId is nil")
	}

	obj, err := p11.findPrivateKey(keyId)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to find private key, keyId = %s", string(keyId))
	}

	pubKey, err := p11.ExportECDSAPublicKey(keyId, keyType)
	if err != nil {
		return nil, errors.WithMessagef(err, "New ecdsa PrivateKey failed")
	}

	var bcPubKey bccrypto.PublicKey
	switch keyType {
	case SM2:
		bcPubKey = &bcsm2.PublicKey{K: pubKey.(*sm2.PublicKey)}
	case ECDSA:
		bcPubKey = &bcecdsa.PublicKey{K: pubKey.(*ecdsa.PublicKey)}
	default:
		return nil, errors.New("unknown key type")
	}

	p11PrivateKey := &p11EcdsaPrivateKey{
		p11Ctx:    p11,
		pubKey:    bcPubKey,
		keyId:     keyId,
		keyType:   keyType,
		keyObject: *obj,
	}

	p11PrivateKey.signer = &ecdsaPrivateKey{p11PrivateKey}

	return p11PrivateKey, nil
}

func (sk *p11EcdsaPrivateKey) Type() bccrypto.KeyType {
	return sk.PublicKey().Type()
}

func (sk *p11EcdsaPrivateKey) Bytes() ([]byte, error) {
	return sk.keyId, nil
}

func (sk *p11EcdsaPrivateKey) String() (string, error) {
	return string(sk.keyId), nil
}

func (sk *p11EcdsaPrivateKey) PublicKey() bccrypto.PublicKey {
	return sk.pubKey
}

func (sk *p11EcdsaPrivateKey) Sign(data []byte) ([]byte, error) {
	var mech uint
	switch sk.Type() {
	case bccrypto.SM2:
		// test needed to verify correctness
		mech = CKM_SM3_SM2_APPID1_DER
	case bccrypto.ECC_Secp256k1, bccrypto.ECC_NISTP256, bccrypto.ECC_NISTP384, bccrypto.ECC_NISTP521:
		mech = pkcs11.CKM_ECDSA
	}
	sig, err := sk.p11Ctx.Sign(sk.keyObject, pkcs11.NewMechanism(mech, nil), data)
	if err != nil {
		return nil, err
	}

	var rsSig ecdsaSignature
	_, err = asn1.Unmarshal(sig, &rsSig)
	if err != nil {
		return nil, errors.WithMessage(err, "pkcs11 ecdsa sign failed")
	}

	return asn1.Marshal(bcecdsa.Sig{R: rsSig.R, S: rsSig.S})
}

func (sk *p11EcdsaPrivateKey) SignWithOpts(msg []byte, opts *bccrypto.SignOpts) ([]byte, error) {
	if opts == nil {
		return sk.Sign(msg)
	}
	if opts.Hash == bccrypto.HASH_TYPE_SM3 && sk.Type() == bccrypto.SM2 {
		//pkSM2, ok := sk.PublicKey().ToStandardKey().(*sm2.PublicKey)
		//if !ok {
		//	return nil, fmt.Errorf("SM2 private key does not match the type it claims")
		//}
		//uid := opts.UID
		//if len(uid) == 0 {
		//	uid = bccrypto.CRYPTO_DEFAULT_UID
		//}
		//
		//za, err := sm2.ZA(pkSM2, []byte(uid))
		//if err != nil {
		//	return nil, fmt.Errorf("PKCS11 error: fail to create SM3 digest for msg [%v]", err)
		//}
		//e := sm3.New()
		//e.Write(za)
		//e.Write(msg)
		//dgst := e.Sum(nil)[:32]

		return sk.Sign(msg)
	}
	dgst, err := hash.Get(opts.Hash, msg)
	if err != nil {
		return nil, err
	}
	return sk.Sign(dgst)
}

func (sk *p11EcdsaPrivateKey) ToStandardKey() crypto.PrivateKey {
	return sk.signer
}
