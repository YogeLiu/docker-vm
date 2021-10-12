module chainmaker.org/chainmaker/vm-docker-go/dockercontainer

go 1.15

require (
	chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk v0.0.0-00010101000000-000000000000
	chainmaker.org/chainmaker/protocol/v2 v2.0.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.0
	go.uber.org/zap v1.18.1
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	google.golang.org/grpc v1.39.0
	gopkg.in/natefinch/lumberjack.v2 v2.0.0
)

replace chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk => ./pb_sdk
