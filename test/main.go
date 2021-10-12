/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker/common/random/uuid"
	"github.com/gogo/protobuf/proto"
	"google.golang.org/grpc"
)

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

func InitTest() {
	contractPath = "/home/jianan/Documents/workspace/chainmaker-go/test/docker-go/contract_1p2"
	//contractName = "contract_1p2"
	mountSockPath = "/home/jianan/Documents/workspace/chainmaker-go/data/org1/docker-go/mount/sock/cdm.sock"
}

func main() {

	fmt.Println("start test")

	InitTest()

	//createContract := false

	stream, _ := initGRPCConnect(true)
	client := NewCDMClient(stream)

	//// 1) 合约创建
	//if createContract {
	//	testDeployContract(client)
	//	time.Sleep(5 * time.Second)
	//}

	// 2) 批量测试
	min = 100000
	max = 0

	txNum := 20000
	batchNum := 100

	for i := 0; i < batchNum; i++ {
		testPerformance(client, txNum, i)
		time.Sleep(50 * time.Microsecond)
	}

	fmt.Printf("avg time is: [%d] ms\n", totalTime/batchNum)
	fmt.Printf("min time is: [%d] ms\n", min)
	fmt.Printf("max time is: [%d] ms\n", max)

	err := client.stream.CloseSend()
	if err != nil {
		fmt.Println("err in close send")
		return
	}

	fmt.Println("end test")

}

//func testDeployContract(client *CDMClient) {
//
//	txId := GetRandTxId()
//
//	fmt.Printf("\n============ create contract %s [%s] ============\n", contractName, txId)
//
//	cdmMsg := contractCreateMsg(txId)
//
//	startTime := time.Now()
//	err := client.stream.Send(cdmMsg)
//	if err != nil {
//		return
//	}
//	fmt.Println("deploy time is: ", time.Since(startTime))
//
//	recvMsg, _ := client.stream.Recv()
//	var result protogo.TxResponse
//	_ = proto.Unmarshal(recvMsg.Payload, &result)
//	fmt.Printf("\n============ create contract result ============\n [%s]\n", result.String())
//
//}

func testPerformance(client *CDMClient, txNum, batchSeq int) {

	fmt.Printf("\n============ test performance, tx number [%d] for [%d] ============\n", txNum, batchSeq)

	startTime := time.Now()

	waitc := make(chan struct{})
	recvNum := 0

	go func() {
		for {
			_, err := client.stream.Recv()

			if err == io.EOF {
				fmt.Println("recv eof")
				close(waitc)
				return
			}

			if err != nil {
				fmt.Println(err)
				return
			}

			recvNum++
			if recvNum >= txNum {
				currentTime := time.Since(startTime)
				fmt.Printf("[%d] tx running time is: [%s]\n", txNum, currentTime)
				close(waitc)

				if int(currentTime/time.Millisecond) < min {
					min = int(currentTime / time.Millisecond)
				}

				if int(currentTime/time.Millisecond) > max {
					max = int(currentTime / time.Millisecond)
				}

				totalTime += int(currentTime / time.Millisecond)
				return
			}

		}
	}()

	for j := 0; j < txNum/2; j++ {

		reqMsg := contractInvokeMsg("contract_1p2_1")
		err := client.stream.Send(reqMsg)
		if err != nil {
			fmt.Println("fail to send req msg: ", err)
			return
		}

		reqMsg1 := contractInvokeMsg("contract_1p2_2")
		err = client.stream.Send(reqMsg1)
		if err != nil {
			fmt.Println("fail to send req msg: ", err)
			return
		}

	}

	//for j := 0; j < txNum/2; j++ {
	//
	//}

	<-waitc

}

//func contractCreateMsg(txId string) *protogo.CDMMessage {
//
//	// construct cdm message
//	params := make(map[string]string)
//	contractBin, _ := ioutil.ReadFile(contractPath)
//
//	txRequest := &protogo.TxRequest{
//		TxId:            txId,
//		ContractName:    contractName,
//		ContractVersion: "1.0.0",
//		Method:          "init_contract",
//		ByteCode:        contractBin,
//		Parameters:      params,
//	}
//
//	txPayload, _ := proto.Marshal(txRequest)
//
//	cdmMessage := &protogo.CDMMessage{
//		TxId:    txId,
//		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
//		Payload: txPayload,
//	}
//	return cdmMessage
//}

func contractInvokeMsg(contractNName string) *protogo.CDMMessage {

	txId := GetRandTxId()

	// construct cdm message
	params := make(map[string][]byte)
	params["arg1"] = []byte("1")
	params["arg2"] = []byte("2")

	txRequest := &protogo.TxRequest{
		TxId:            txId,
		ContractName:    contractNName,
		ContractVersion: "1.2.1",
		Method:          "invoke_contract",
		Parameters:      params,
		TxContext: &protogo.TxContext{
			CurrentHeight:       0,
			WriteMap:            nil,
			ReadMap:             nil,
			OriginalProcessName: "",
		},
	}

	txPayload, _ := proto.Marshal(txRequest)

	cdmMessage := &protogo.CDMMessage{
		TxId:    txId,
		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
		Payload: txPayload,
	}
	return cdmMessage
}

// NewClientConn create client connection
func NewClientConn(udsOpen bool) (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMessageSize),
			grpc.MaxCallSendMsgSize(maxSendMessageSize),
		),
	}

	if udsOpen {

		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
			unixAddress, _ := net.ResolveUnixAddr("unix", sock)
			conn, err := net.DialUnix("unix", nil, unixAddress)
			return conn, err
		}))

		sockAddress := mountSockPath

		return grpc.DialContext(context.Background(), sockAddress, dialOpts...)

	}

	return grpc.Dial(port, dialOpts...)

}

// GetCDMClientStream get rpc stream
func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}

func initGRPCConnect(udsOpen bool) (protogo.CDMRpc_CDMCommunicateClient, error) {
	conn, err := NewClientConn(udsOpen)
	if err != nil {
		fmt.Println("fail to create connection: ", err)
		return nil, err
	}

	stream, err := GetCDMClientStream(conn)
	if err != nil {
		fmt.Println("fail to get connection stream: ", err)
		return nil, err
	}

	return stream, nil
}

type CDMClient struct {
	txSendCh chan *protogo.CDMMessage // channel receive tx from docker-go instance

	lock sync.Mutex
	// store tx_id to chan, retrieve chan to send tx response back to docker-go instance
	recvChMap map[string]chan *protogo.CDMMessage

	stream protogo.CDMRpc_CDMCommunicateClient

	stop chan bool
}

func NewCDMClient(stream protogo.CDMRpc_CDMCommunicateClient) *CDMClient {

	return &CDMClient{
		txSendCh:  make(chan *protogo.CDMMessage, chanSize),
		recvChMap: make(map[string]chan *protogo.CDMMessage),
		lock:      sync.Mutex{},
		stream:    stream,
		stop:      make(chan bool),
	}
}

func GetRandTxId() string {
	return uuid.GetUUID() + uuid.GetUUID()
}
