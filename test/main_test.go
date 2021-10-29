package test

import (
	"os"
	"sync"
	"testing"

	"chainmaker.org/chainmaker/vm-docker-go/pb/protogo"
)

/*
- create user contract
- invoke user contract
*/

const (
	maxRecvMessageSize = 100 * 1024 * 1024 // 100 MiB
	maxSendMessageSize = 100 * 1024 * 1024 // 100 MiB
	port               = ":12355"
	chanSize           = 1000
	//stateChanSize      = 1000
)

var (
	contractPath  = ""
	contractName  = ""
	mountSockPath = ""
	min           int
	max           int
	totalTime     int
)

type CDMClient struct {
	txSendCh chan *protogo.CDMMessage // channel receive tx from docker-go instance

	lock sync.Mutex
	// store tx_id to chan, retrieve chan to send tx response back to docker-go instance
	recvChMap map[string]chan *protogo.CDMMessage

	stream protogo.CDMRpc_CDMCommunicateClient

	stop chan bool
}

func TestMain(m *testing.M) {
	//os.cmd(go mod vendor)
	dockerGo := m.Run()
	os.Exit(dockerGo)
}
