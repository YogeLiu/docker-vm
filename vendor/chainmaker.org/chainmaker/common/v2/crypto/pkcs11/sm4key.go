package pkcs11

import (
	"crypto/rand"
	"fmt"

	bccrypto "chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/sym/modes"
	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

var defaultSM4Opts = &bccrypto.EncOpts{
	EncodingType: modes.PADDING_NONE,
	BlockMode:    modes.BLOCK_MODE_GCM,
	EnableMAC:    true,
	Hash:         0,
	Label:        nil,
	EnableASN1:   false,
}

var _ bccrypto.SymmetricKey = (*sm4Key)(nil)

type sm4Key struct {
	p11Ctx    *P11Handle
	keyId     []byte
	keyType   P11KeyType
	keyObject pkcs11.ObjectHandle
	blockSize int
}

func NewSM4Key(ctx *P11Handle, keyId []byte) (bccrypto.SymmetricKey, error) {
	obj, err := ctx.findSecretKey(keyId)
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to find sm4 key [%s]", err)
	}

	return &sm4Key{
		p11Ctx:    ctx,
		keyId:     keyId,
		keyObject: *obj,
		keyType:   SM4,
		blockSize: 16,
	}, nil
}

func (s *sm4Key) Bytes() ([]byte, error) {
	return s.keyId, nil
}

func (s *sm4Key) Type() bccrypto.KeyType {
	return bccrypto.SM4
}

func (s *sm4Key) String() (string, error) {
	return string(s.keyId), nil
}

func (s *sm4Key) Encrypt(plain []byte) ([]byte, error) {
	return s.EncryptWithOpts(plain, defaultSM4Opts)
}

func (s *sm4Key) EncryptWithOpts(plain []byte, opts *bccrypto.EncOpts) ([]byte, error) {
	iv := make([]byte, s.blockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	ciphertex, err := s.p11Ctx.Encrypt(s.keyObject, pkcs11.NewMechanism(CKM_SM4_CBC, iv), plain)
	if err != nil {
		return nil, err
	}
	return append(iv, ciphertex...), nil
}

func (s *sm4Key) Decrypt(ciphertext []byte) ([]byte, error) {
	return s.DecryptWithOpts(ciphertext, defaultSM4Opts)
}

func (s *sm4Key) DecryptWithOpts(ciphertext []byte, opts *bccrypto.EncOpts) ([]byte, error) {
	if len(ciphertext) < s.blockSize {
		return nil, errors.New("invalid ciphertext length")
	}

	iv := ciphertext[:s.blockSize]
	out, err := s.p11Ctx.Decrypt(s.keyObject, pkcs11.NewMechanism(CKM_SM4_CBC, iv), ciphertext[s.blockSize:])
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to encrypt [%s]", err)
	}

	return out, nil
}
