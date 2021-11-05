package test

import (
	"os"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2/mock"

	"chainmaker.org/chainmaker/protocol/v2"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	dockergo "chainmaker.org/chainmaker/vm-docker-go"
)

/*
- create user contract
- invoke user contract
*/

var (
	mockDockerManager   *dockergo.DockerManager
	mockContractId      *commonPb.Contract
	mockTxContext       *mock.MockTxSimContext
	mockRuntimeInstance protocol.RuntimeInstance
)

func TestMain(m *testing.M) {

	//execCommand("./scripts/prepare.sh")

	dockerGo := m.Run()

	//execCommand("./scripts/clean.sh")

	os.Exit(dockerGo)
}
