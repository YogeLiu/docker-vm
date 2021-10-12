module chainmaker.org/chainmaker-go/docker-go

go 1.15

require (
	chainmaker.org/chainmaker-go/docker-go/dockercontainer v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/localconf v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker-go/logger v0.0.0
	chainmaker.org/chainmaker-go/utils v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker/common v0.0.0-20210722032200-380ced605d25
	chainmaker.org/chainmaker/pb-go/v2 v2.0.0
	chainmaker.org/chainmaker/protocol/v2 v2.0.0
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/Microsoft/hcsshim v0.8.17 // indirect
	github.com/containerd/containerd v1.5.2 // indirect
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/moby/sys/mount v0.2.0 // indirect
	github.com/moby/term v0.0.0-20201216013528-df9cb8a40635 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4 // indirect
	golang.org/x/tools v0.1.5 // indirect
	google.golang.org/grpc v1.39.0
)

replace (
	chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk => ./dockercontainer/pb_sdk

	chainmaker.org/chainmaker-go/docker-go/dockercontainer => ./dockercontainer
	chainmaker.org/chainmaker-go/localconf => ../../conf/localconf
	chainmaker.org/chainmaker-go/logger => ../../logger
	chainmaker.org/chainmaker-go/utils => ../../utils

)
