/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/common/v2/msgbus"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/consensus/chainedbft"
	"chainmaker.org/chainmaker/pb-go/v2/txpool"
)

const DefaultBlockVersion = uint32(20) // default version of chain
// Block committer, put block and read write set into ledger(DB).
type BlockCommitter interface {
	// Put block into ledger(DB) after block verify. Invoke by consensus or sync module.
	AddBlock(blk *common.Block) error
}

// Block proposer, generate new block when node is consensus proposer.
type BlockProposer interface {
	// Start proposer.
	Start() error
	// Stop proposer
	Stop() error
	// Receive propose signal from txpool module.
	OnReceiveTxPoolSignal(proposeSignal *txpool.TxPoolSignal)
	// Receive signal indicates if node is proposer from consensus module.
	OnReceiveProposeStatusChange(proposeStatus bool)
	// Receive signal from chained bft consensus(Hotstuff) and propose new block.
	OnReceiveChainedBFTProposal(proposal *chainedbft.BuildProposal)
}

// Block verifier, verify if a block is valid
type BlockVerifier interface {
	// Verify if a block is valid
	VerifyBlock(block *common.Block, mode VerifyMode) error
}

//go:generate stringer -type=VerifyMode
type VerifyMode int

const (
	CONSENSUS_VERIFY VerifyMode = iota
	SYNC_VERIFY
)

type CoreEngine interface {
	Start()
	Stop()
	GetBlockCommitter() BlockCommitter
	GetBlockVerifier() BlockVerifier
	msgbus.Subscriber
	//HotStuffHelper
	GetHotStuffHelper() HotStuffHelper
}
