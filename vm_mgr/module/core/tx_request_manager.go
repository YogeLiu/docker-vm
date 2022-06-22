/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"fmt"
	"strings"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

// SysCallElapsedTime syscall include read date from chainmaker node and cross call contract
type SysCallElapsedTime struct {
	OpType               protogo.CDMType
	StartTime            int64
	EndTime              int64
	TotalTime            int64
	StorageTimeInSysCall int64
}

func NewSysCallElapsedTime(opType protogo.CDMType, startTime int64, totalTime int64, storageTime int64) *SysCallElapsedTime {
	return &SysCallElapsedTime{
		OpType:               opType,
		StartTime:            startTime,
		TotalTime:            totalTime,
		StorageTimeInSysCall: storageTime,
	}
}

func (s *SysCallElapsedTime) ToString() string {
	if s == nil {
		return ""
	}
	return fmt.Sprintf("%s start: %v, spend: %dμs, r/w store: %dμs; ",
		s.OpType.String(), time.Unix(s.StartTime/1e9, s.StartTime%1e9), s.TotalTime/1000, s.StorageTimeInSysCall/1000,
	)
}

func (e *TxElapsedTime) PrintCallList() string {
	if e.SysCallList == nil {
		return "no syscalls"
	}
	var sb strings.Builder
	for _, sysCallTime := range e.SysCallList {
		sb.WriteString(sysCallTime.ToString())
	}
	sb.WriteString("cross calls: ")
	for _, sysCallTime := range e.CrossCallList {
		sb.WriteString(sysCallTime.ToString())
	}
	return sb.String()
}

type TxElapsedTime struct {
	ChainId               string
	TxId                  string
	StartTime             int64
	EndTime               int64
	TotalTime             int64
	SysCallCnt            int32
	SysCallTime           int64
	StorageTimeInSysCall  int64
	ContingentSysCallCnt  int32
	ContingentSysCallTime int64
	CrossCallCnt          int32
	CrossCallTime         int64
	SysCallList           []*SysCallElapsedTime
	CrossCallList         []*SysCallElapsedTime
}

func NewTxElapsedTime(txId string, startTime int64) *TxElapsedTime {
	return &TxElapsedTime{
		TxId:      txId,
		StartTime: startTime,
	}
}

func (e *TxElapsedTime) ToString() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf("%s spend time: %dμs, syscall: %dμs(%d), r/w store: %dμs, possible syscall: %dμs(%d)"+
		"cross contract: %dμs(%d)",
		e.TxId, e.TotalTime/1000, e.SysCallTime/1000, e.SysCallCnt, e.StorageTimeInSysCall/1000,
		e.ContingentSysCallTime/1000, e.ContingentSysCallCnt, e.CrossCallTime/1000, e.CrossCallCnt,
	)
}

// AddSysCallElapsedTime todo add lock (maybe do not need)
func (e *TxElapsedTime) AddSysCallElapsedTime(sysCallElapsedTime *SysCallElapsedTime) {
	if sysCallElapsedTime == nil {
		return
	}

	switch sysCallElapsedTime.OpType {
	case protogo.CDMType_CDM_TYPE_GET_BYTECODE, protogo.CDMType_CDM_TYPE_GET_CONTRACT_NAME:
		e.ContingentSysCallCnt += 1
		e.ContingentSysCallTime += sysCallElapsedTime.TotalTime
		e.StorageTimeInSysCall += sysCallElapsedTime.StorageTimeInSysCall
	case protogo.CDMType_CDM_TYPE_GET_STATE, protogo.CDMType_CDM_TYPE_GET_BATCH_STATE,
		protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR, protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR,
		protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER, protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER,
		protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS:
		e.SysCallCnt += 1
		e.SysCallTime += sysCallElapsedTime.TotalTime
		e.StorageTimeInSysCall += sysCallElapsedTime.StorageTimeInSysCall
	default:
		return
	}

	e.SysCallList = append(e.SysCallList, sysCallElapsedTime)
	return
}

// AddCallContractElapsedTime todo add lock (maybe do not need)
func (e *TxElapsedTime) AddCallContractElapsedTime(crossCallElapsedTime *SysCallElapsedTime) {
	if crossCallElapsedTime == nil {
		return
	}
	e.CrossCallCnt += 1
	e.CrossCallTime += crossCallElapsedTime.TotalTime

	e.SysCallList = append(e.CrossCallList, crossCallElapsedTime)
	return
}

func (e *TxElapsedTime) Add(t *TxElapsedTime) {
	if t == nil {
		return
	}

	t.TotalTime += t.TotalTime

	e.SysCallCnt += t.SysCallCnt
	e.SysCallTime += t.SysCallTime
	e.StorageTimeInSysCall += t.StorageTimeInSysCall

	e.ContingentSysCallCnt += t.ContingentSysCallCnt
	e.ContingentSysCallTime += t.ContingentSysCallTime

	e.CrossCallCnt += t.CrossCallCnt
	e.CrossCallTime += t.CrossCallTime
}
