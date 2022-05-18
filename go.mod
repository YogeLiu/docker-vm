module chainmaker.org/chainmaker/vm-docker-go/v2

go 1.15

require (
	chainmaker.org/chainmaker/common/v2 v2.2.1-0.20220419120639-3f11d401538b
	chainmaker.org/chainmaker/localconf/v2 v2.2.0
	chainmaker.org/chainmaker/logger/v2 v2.2.0
	chainmaker.org/chainmaker/pb-go/v2 v2.2.1-0.20220330115503-be7240795241
	chainmaker.org/chainmaker/protocol/v2 v2.2.2-0.20220507040216-42659b58ef08
	chainmaker.org/chainmaker/utils/v2 v2.2.1
	chainmaker.org/chainmaker/vm-native/v2 v2.2.1 // indirect
	chainmaker.org/chainmaker/vm/v2 v2.2.1-0.20220511123249-b79689d7e465
	github.com/Microsoft/go-winio v0.5.1 // indirect
	github.com/containerd/containerd v1.5.9 // indirect
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.7+incompatible
	github.com/docker/go-connections v0.4.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/mitchellh/mapstructure v1.4.2
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.7.0
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	golang.org/x/time v0.0.0-20210608053304-ed9ce3a009e4 // indirect
	google.golang.org/grpc v1.41.0
)

replace (
	chainmaker.org/chainmaker/logger/v2 v2.2.0 => ../logger
)
