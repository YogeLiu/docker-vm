/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pkcs11

import (
	"fmt"

	"github.com/miekg/pkcs11"
	"github.com/pkg/errors"
)

type P11Handle struct {
	ctx              *pkcs11.Ctx
	sessions         chan pkcs11.SessionHandle
	slot             uint
	sessionCacheSize int
	hash             string
}

func New(lib string, label string, password string, sessionCacheSize int, hash string) (*P11Handle, error) {
	if lib == "" {
		return nil, errors.New("PKCS11 error: empty library path")
	}

	if password == "" {
		return nil, errors.New("PKCS11 error: no pin provided")
	}

	if sessionCacheSize <= 0 {
		sessionCacheSize = 10
	}

	ctx := pkcs11.New(lib)
	if ctx == nil {
		return nil, fmt.Errorf("PKCS11 error: fail to initialize [%s]", lib)
	}

	err := ctx.Initialize()
	if err != nil {
		return nil, err
	}

	slots, err := ctx.GetSlotList(true)
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to get slot list [%v]", err)
	}

	found := false
	var slot uint
	slot, found = findSlot(ctx, slots, label)
	if !found {
		return nil, fmt.Errorf("PKCS11 error: fail to find token with label [%s]", label)
	}

	var session pkcs11.SessionHandle
	for i := 0; i < 10; i++ {
		session, err = ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
		if err != nil {
			continue
		}
		break
	}
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to open session [%v]", err)
	}

	err = ctx.Login(session, pkcs11.CKU_USER, password)
	if err != nil {
		return nil, fmt.Errorf("PKCS11 error: fail to login session [%v]", err)
	}

	sessions := make(chan pkcs11.SessionHandle, sessionCacheSize)
	p11Handle := &P11Handle{
		ctx:              ctx,
		sessions:         sessions,
		slot:             slot,
		sessionCacheSize: sessionCacheSize,
		hash:             hash,
	}
	p11Handle.returnSession(session)

	return p11Handle, nil
}

func (p11 *P11Handle) getSession() (pkcs11.SessionHandle, error) {
	var session pkcs11.SessionHandle
	select {
	case session = <-p11.sessions:
		return session, nil
	default:
		var err error
		for i := 0; i < 10; i++ {
			session, err = p11.ctx.OpenSession(p11.slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
			if err == nil {
				return session, nil
			}
		}
		return 0, fmt.Errorf("PKCS11 error: fail to open session [%v]", err)
	}
}

func (p11 *P11Handle) returnSession(session pkcs11.SessionHandle) {
	select {
	case p11.sessions <- session:
		return
	default:
		_ = p11.ctx.CloseSession(session)
		return
	}
}

func findSlot(ctx *pkcs11.Ctx, slots []uint, label string) (uint, bool) {
	var slot uint
	var found bool
	for _, s := range slots {
		info, err := ctx.GetTokenInfo(s)
		if err != nil {
			continue
		}
		if info.Label == label {
			found = true
			slot = s
			break
		}
	}
	return slot, found
}

func listSlot(ctx *pkcs11.Ctx) (map[string]string, error) {
	slots, err := ctx.GetSlotList(true)
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)
	for i, s := range slots {
		info, err := ctx.GetTokenInfo(s)
		if err != nil {
			return nil, err
		}
		res[fmt.Sprintf("%d", i)] = info.Label
	}
	return res, nil
}
