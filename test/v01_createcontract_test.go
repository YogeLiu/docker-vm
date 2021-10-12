package test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"
	"time"

	dockergo "chainmaker.org/chainmaker-go/docker-go"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontroller"
	"chainmaker.org/chainmaker-go/docker-go/dockercontroller/module"
	"chainmaker.org/chainmaker-go/localconf"
	"chainmaker.org/chainmaker-go/logger"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

/*
1 get chainmaker configuration setting from mock file
2 generate a new docker manager
3 start a docker container
4 mock TxSimContext(interaction with chain)
5 mock docker-go RuntimeInstace

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

func TestCreateDockerGoContract(t *testing.T) {

	//step1: get chainmaker configuration setting from mocked data
	cmConfig, err := getMockedCMConfig()
	if err != nil {
		t.Errorf("get the mocked chainmaker configuration failed %v\n", err)
	}

	//step2: generate a docker manager instance
	chainId := "chain1"
	dockerMangerInstance := dockercontroller.NewDockerManager(chainId, cmConfig)

	//step3: start docker manger instance
	dockerContainErr := dockerMangerInstance.StartContainer()
	if dockerContainErr != nil {
		t.Errorf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	time.Sleep(2 * time.Second)
	//step4: mock contractId and txContext
	fmt.Printf("======step4 mock contractId and txContext=======\n")
	contractId, txContext := InitContextTest(commonPb.RuntimeType_DOCKER_GO)
	fmt.Printf("the contractId %v\n", contractId.RuntimeType)
	//step5: generate new CDM Clinet for testing
	fmt.Printf("======step5 generate new cdm client=======\n")
	CDMCMockClient := module.NewCDMClient(chainId, cmConfig)
	_ = CDMCMockClient.StartClient()

	runtimeMockInstance := &dockergo.RuntimeInstance{
		ChainId: chainId,
		Client:  CDMCMockClient,
		Log:     logger.GetLogger(logger.MODULE_VM),
	}

	//step6: creare a user contract
	method := "init_contract"
	fmt.Printf("======step6 create contract=======\n")
	contractBin, contractFileErr := ioutil.ReadFile("/root/go/src/chainmaker-go/module/vm/docker-go/test/testdata/DockerGoCal00.7z")
	if contractFileErr != nil {
		t.Errorf("get byte code failed %v", contractFileErr)
	}
	parameters := make(map[string][]byte)
	parameters["__sender_pk__"] = []byte("c8c088cd9e333950f47cd5f5e3d6ebdadf522459553da90c138ca8ce16549480")
	parameters["__creator_org_id__"] = []byte("wx-org1.chainmaker.org")
	parameters["__creator_role__"] = []byte("CLIENT")
	parameters["__block_height__"] = []byte("1")
	parameters["__sender_org_id__"] = []byte("wx-org1.chainmaker.org")
	parameters["__creator_pk__"] = []byte("c8c088cd9e333950f47cd5f5e3d6ebdadf522459553da90c138ca8ce16549480")
	parameters["__tx_id__"] = []byte("a4f108f00005492bb608b7237b23fac0adfe521fb08b4f86aefb774843a4fc1e")
	parameters["__sender_role__"] = []byte("CLIENT")
	gasUsed := uint64(1232)

	result := runtimeMockInstance.Invoke(contractId, method, contractBin, parameters, txContext, gasUsed)

	fmt.Printf("here is ##### %v\n", result)

}
