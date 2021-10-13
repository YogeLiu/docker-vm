package pkcs11

import (
	"crypto/rand"
	"fmt"

	"github.com/pkg/errors"

	"chainmaker.org/chainmaker/common/v2/crypto/sym/modes"

	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"github.com/miekg/pkcs11"
)

var defaultAESOpts = &bccrypto.EncOpts{
	EncodingType: modes.PADDING_NONE,
	BlockMode:    modes.BLOCK_MODE_CBC,
	EnableMAC:    true,
	Hash:         0,
	Label:        nil,
	EnableASN1:   true,
}

var _ bccrypto.SymmetricKey = (*aesKey)(nil)

type aesKey struct {
	p11Ctx    *P11Handle
	keyId     []byte
	keyType   P11KeyType
	keyObject pkcs11.ObjectHandle
	keySize   int
	blockSize int
}

func NewAESKey(ctx *P11Handle, keyId []byte) (bccrypto.SymmetricKey, error) {
	obj, err := ctx.findSecretKey(keyId)
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to find aes key [%s]", err)
	}

	sk := aesKey{p11Ctx: ctx,
		keyId:     keyId,
		keyObject: *obj,
		keyType:   AES,
		blockSize: 16,
	}

	sk.keySize, err = ctx.getSecretKeySize(*obj)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to get aes keySize")
	}
	return &sk, nil
}

func (s *aesKey) Bytes() ([]byte, error) {
	return s.keyId, nil
}

func (s *aesKey) Type() bccrypto.KeyType {
	return bccrypto.AES
}

func (s *aesKey) String() (string, error) {
	return string(s.keyId), nil
}

func (s *aesKey) Encrypt(plain []byte) ([]byte, error) {
	return s.EncryptWithOpts(plain, defaultSM4Opts)
}

func (s *aesKey) EncryptWithOpts(plain []byte, opts *bccrypto.EncOpts) ([]byte, error) {
	iv := make([]byte, s.blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}
	ciphertext, err := s.p11Ctx.Encrypt(s.keyObject, pkcs11.NewMechanism(pkcs11.CKM_AES_CBC, iv), plain)
	if err != nil {
		return nil, err
	}
	return append(iv, ciphertext...), nil
}

func (s *aesKey) Decrypt(ciphertext []byte) ([]byte, error) {
	return s.DecryptWithOpts(ciphertext, defaultAESOpts)
}

func (s *aesKey) DecryptWithOpts(ciphertext []byte, opts *bccrypto.EncOpts) ([]byte, error) {
	if len(ciphertext) < s.blockSize {
		return nil, errors.New("invalid ciphertext length")
	}

	iv := ciphertext[:s.blockSize]

	out, err := s.p11Ctx.Decrypt(s.keyObject, pkcs11.NewMechanism(pkcs11.CKM_AES_CBC, iv), ciphertext[s.blockSize:])
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to encrypt [%s]", err)
	}

	return out, nil
}
