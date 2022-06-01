module chainmaker.org/chainmaker/vm-docker-go/v2

go 1.15

require (
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220601090655-ddcadbaf280f
	chainmaker.org/chainmaker/localconf/v2 v2.2.2-0.20220601092821-2162b2905ed5
	chainmaker.org/chainmaker/logger/v2 v2.2.2-0.20220601091955-6c66ad476f3a
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220601073343-3015c97c2728
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220601091317-c1b2cd0fb763
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220601092510-ec93c1095f0f
	github.com/docker/distribution v2.7.1+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.7.0
	google.golang.org/grpc v1.41.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0 // with test error google.golang.org/grpc/naming: module google.golang.org/grpc@latest found (v1.47.0), but does not contain package google.golang.org/grpc/naming
