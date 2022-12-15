/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"os"
	"testing"

	"chainmaker.org/chainmaker/protocol/v3/mock"

	"chainmaker.org/chainmaker/protocol/v3"

	commonPb "chainmaker.org/chainmaker/pb-go/v3/common"
)

/*
- create user contract
- invoke user contract
*/

var (
	mockDockerManager   protocol.VmInstancesManager
	mockContractId      *commonPb.Contract
	mockTxContext       *mock.MockTxSimContext
	mockRuntimeInstance protocol.RuntimeInstance
)

func TestMain(m *testing.M) {

	dockerGo := m.Run()

	os.Exit(dockerGo)
}
