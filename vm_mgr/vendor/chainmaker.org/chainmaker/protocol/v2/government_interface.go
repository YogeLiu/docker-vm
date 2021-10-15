/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protocol

import (
	"chainmaker.org/chainmaker/pb-go/v2/config"
	consensuspb "chainmaker.org/chainmaker/pb-go/v2/consensus"
)

type Government interface {
	//used to verify consensus data
	Verify(consensusType consensuspb.ConsensusType, chainConfig *config.ChainConfig) error

	GetGovernanceContract() (*consensuspb.GovernanceContract, error)
}
