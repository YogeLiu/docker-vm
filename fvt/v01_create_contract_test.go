package test

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontroller"
	"chainmaker.org/chainmaker-go/localconf"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

/*
1 get chainmaker configuration setting from mock file
2 generate a new docker manager
3 start a docker container

*/

//getMockedCMConfig get chainmaker configuration from mocked data
func getMockedCMConfig() (*localconf.CMConfig, error) {
	// 0. load env
	cmd := &cobra.Command{}
	cmViper := viper.New()
	err := cmViper.BindPFlags(cmd.PersistentFlags())
	if err != nil {
		return nil, err
	}
	// 1. load the path of the config files
	mockedConfigration := "./testdata/chainmaker.yml"
	ymlFile := mockedConfigration
	if !filepath.IsAbs(ymlFile) {
		ymlFile, _ = filepath.Abs(ymlFile)
		mockedConfigration = ymlFile
	}

	// 2. load the config file
	cmViper.SetConfigFile(ymlFile)
	if err := cmViper.ReadInConfig(); err != nil {
		return nil, err
	}

	logConfigFile := cmViper.GetString("log.config_file")
	if logConfigFile != "" {
		cmViper.SetConfigFile(logConfigFile)
		if err := cmViper.MergeInConfig(); err != nil {
			return nil, err
		}
	}
	flagSets := make([]*pflag.FlagSet, 0)
	for _, command := range cmd.Commands() {
		flagSets = append(flagSets, command.PersistentFlags())
		err := cmViper.BindPFlags(command.PersistentFlags())
		if err != nil {
			return nil, err
		}
	}

	// 3. create new CMConfig instance
	cmConfig := &localconf.CMConfig{}
	if err := cmViper.Unmarshal(cmConfig); err != nil {
		return nil, err
	}
	return cmConfig, nil
}

//newClientConn mock a client connection
func newClientConn(cmConfig *localconf.CMConfig) (*grpc.ClientConn, error) {

	dialOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.FailOnNonTempDialError(true),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxRecvMessageSize),
			grpc.MaxCallSendMsgSize(maxSendMessageSize),
		),
	}

	if cmConfig.DockerConfig.DockerRpcConfig.UdsOpen {

		dialOpts = append(dialOpts, grpc.WithContextDialer(func(ctx context.Context, sock string) (net.Conn, error) {
			unixAddress, _ := net.ResolveUnixAddr("unix", sock)
			conn, err := net.DialUnix("unix", nil, unixAddress)
			return conn, err
		}))

		sockAddress := cmConfig.DockerConfig.MountPath + "/sock/cdm.sock"

		return grpc.DialContext(context.Background(), sockAddress, dialOpts...)

	}

	return grpc.Dial(port, dialOpts...)

}

//getCDMClientStream get rpc stream
func getCDMClientStream(conn *grpc.ClientConn) (protogo.CDMRpc_CDMCommunicateClient, error) {
	return protogo.NewCDMRpcClient(conn).CDMCommunicate(context.Background())
}

// //newCDMClient create a cdmClient
// func newCDMClient(stream protogo.CDMRpc_CDMCommunicateClient) *CDMClient {

// 	return &CDMClient{
// 		txSendCh:  make(chan *protogo.CDMMessage, chanSize),
// 		recvChMap: make(map[string]chan *protogo.CDMMessage),
// 		lock:      sync.Mutex{},
// 		stream:    stream,
// 		stop:      make(chan bool),
// 	}
// }

// //generateUUID generate uuid as txID
// func generateUUID() string {
// 	return uuid.GetUUID() + uuid.GetUUID()
// }

// //mockContractCreateMsg mock to create user contract message
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
// 		ByteCode:        nil,
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

func TestCreateDockerGoContract(t *testing.T) {

	//step1: get chainmaker configuration setting from mocked data
	cmConfig, err := getMockedCMConfig()
	if err != nil {
		t.Errorf("get the mocked chainmaker configuration failed %v\n", err)
	}

	// fmt.Printf("here is cmConfig %+v\n", cmConfig.DockerConfig.DockerRpcConfig.UdsOpen)
	//step2: generate a docker manager instance
	chainId := "chain1"
	dockerMangerInstance := dockercontroller.NewDockerManager(chainId, cmConfig)

	//step3: start docker manger instance
	dockerContainErr := dockerMangerInstance.StartContainer()
	if dockerContainErr != nil {
		t.Errorf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	time.Sleep(2 * time.Second)

	// fmt.Printf("here is cmConfig %+v\n", cmConfig)

	// //initialize grpc connection
	// grpcClientConn, connErr := newClientConn(cmConfig)
	// if connErr != nil {
	// 	t.Errorf("initialized grpc connection failed %v", connErr)
	// }

	// //get rpc stream
	// stream, streamErr := getCDMClientStream(grpcClientConn)
	// if streamErr != nil {
	// 	t.Errorf("get rpc stream failed %v\n", streamErr)
	// }

	// //create grpc client
	// client := newCDMClient(stream)

	// //mock a create user contract payload
	// txId := generateUUID()
	// cdmMsg, cdmMsgErr := mockContractCreateMsg(txId)
	// if cdmMsgErr != nil {
	// 	fmt.Printf("mock cdmMsg failed: %v\n", cdmMsgErr)
	// }

	// //send create user contract by grpc connection
	// clientErr := client.stream.Send(cdmMsg)
	// if clientErr != nil {
	// 	t.Errorf("send message by grpc to create user contract failed %v", clientErr)
	// }
	// time.Sleep(2 * time.Second)
	// recvMsg, _ := client.stream.Recv()
	// result := protogo.TxResponse{}
	// _ = proto.Unmarshal(recvMsg.Payload, &result)
	// fmt.Printf("here is result %v\n", result.String())

}
