package test

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2/mock"

	"chainmaker.org/chainmaker/protocol/v2"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	docker_go "chainmaker.org/chainmaker/vm-docker-go"
)

/*
- create user contract
- invoke user contract
*/

var (
	mockDockerManager   *docker_go.DockerManager
	mockContractId      *commonPb.Contract
	mockTxContext       *mock.MockTxSimContext
	mockRuntimeInstance protocol.RuntimeInstance
)

func TestMain(m *testing.M) {

	//command := "echo hello"
	cmd := exec.Command("/bin/bash", "-c", "echo hello")
	_, _ = cmd.Output()

	fmt.Println("image init successful")
	fmt.Println(os.Getwd())

	dockerGo := m.Run()

	//cmd = exec.Command("./test/testdata/clean.sh")
	//_ = cmd.Run()

	fmt.Println("end")

	os.Exit(dockerGo)
}
