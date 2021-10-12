package test

import (
	"testing"
)

// NewClientConn create client connection
// func NewClientConn(udsOpen bool) (*grpc.ClientConn, error) {

// 	dialOpts := []grpc.DialOption{
// 		grpc.WithInsecure(),
// 		grpc.FailOnNonTempDialError(true),
// 		grpc.WithDefaultCallOptions(
// 			grpc.MaxCallRecvMsgSize(maxRecvMessageSize),
// 			grpc.MaxCallSendMsgSize(maxSendMessageSize),
// 		),
// 	}

// 	if udsOpen {

// 		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
// 			unixAddress, _ := net.ResolveUnixAddr("unix", sock)
// 			conn, err := net.DialUnix("unix", nil, unixAddress)
// 			return conn, err
// 		}))

// 		sockAddress := mountSockPath

// 		return grpc.DialContext(context.Background(), sockAddress, dialOpts...)

// 	}

// 	return grpc.Dial(port, dialOpts...)

// }

// func GetCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
// 	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
// }

// func NewCDMClient(stream protogo.CDMRpc_CDMCommunicateClient) *CDMClient {

// 	return &CDMClient{
// 		txSendCh:  make(chan *protogo.CDMMessage, chanSize),
// 		recvChMap: make(map[string]chan *protogo.CDMMessage),
// 		lock:      sync.Mutex{},
// 		stream:    stream,
// 		stop:      make(chan bool),
// 	}
// }

// func GenerateUUID() string {
// 	return uuid.GetUUID() + uuid.GetUUID()
// }

// func mockContractCreateMsg(txId string) (*protogo.CDMMessage, error) {

// 	contractBin, contractFileErr := ioutil.ReadFile("/root/go/src/chainmaker-go/module/vm/docker-go/test/testdata/DockerGoCal00.7z")
// 	if contractFileErr != nil {
// 		fmt.Printf("load contract file failed: %v\n", contractFileErr)
// 		return nil, contractFileErr
// 	}
// 	txRequest := &protogo.TxRequest{
// 		TxId:            txId,
// 		ContractName:    "DockerGoCal00",
// 		ContractVersion: "1.0.0",
// 		Method:          "init_contract",
// 		ByteCode:        contractBin,
// 	}

// 	txPayload, txPayloadErr := proto.Marshal(txRequest)
// 	if txPayloadErr != nil {
// 		fmt.Printf("Generate txPayload error: %v\n", txPayloadErr)
// 		return nil, txPayloadErr
// 	}

// 	cdmMessage := &protogo.CDMMessage{
// 		TxId:    txId,
// 		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
// 		Payload: txPayload,
// 	}

// 	return cdmMessage, nil

// }

func TestCreateDockerGo(t *testing.T) {
	//mock create user contract payload
	// txId := GenerateUUID()

	// cdmMsg, cdmMsgErr := mockContractCreateMsg(txId)
	// if cdmMsgErr != nil {
	// 	fmt.Printf("mock cdmMsg failed: %v\n", cdmMsgErr)
	// }
	// fmt.Printf("###here is mock message %+v\n", cdmMsg)

	// //start container

	// //sock:=

	// //initialize grpc connection
	// conn, connErr := NewClientConn(true)
	// if connErr != nil {
	// 	fmt.Printf("create new client faile %v\n", connErr)
	// }
	// stream, streamErr := GetCDMClientStream(conn)
	// if streamErr != nil {
	// 	fmt.Printf("create stream faile %v\n", streamErr)
	// }
	// client := NewCDMClient(stream)

	// //grpc message send & receive
	// clientErr := client.stream.Send(cdmMsg)
	// if clientErr != nil {
	// 	fmt.Printf("send message failed %v\n", clientErr)
	// }

	// recMsg, recMsgErr := client.stream.Recv()
	// if recMsgErr != nil {
	// 	fmt.Printf("receive message failed %v\n", recMsgErr)
	// }

	// result := &protogo.TxResponse{}
	// _ = proto.Unmarshal(recMsg.Payload, result)

	// fmt.Printf("here is result %v", result)

}
