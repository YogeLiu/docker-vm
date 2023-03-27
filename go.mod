module chainmaker.org/chainmaker/vm-engine/v2

go 1.16

require (
	chainmaker.org/chainmaker/common/v2 v2.3.2-0.20230322081043-6b57e7eb27cf
	chainmaker.org/chainmaker/localconf/v2 v2.3.1
	chainmaker.org/chainmaker/logger/v2 v2.3.0
	chainmaker.org/chainmaker/pb-go/v2 v2.3.3-0.20230321020437-c64dd5a3c606
	chainmaker.org/chainmaker/protocol/v2 v2.3.3-0.20230320082116-0c7643a35069
	chainmaker.org/chainmaker/utils/v2 v2.3.3-0.20230327083208-e51ca4aeac91
	chainmaker.org/chainmaker/vm-native/v2 v2.3.3-0.20230322035808-f1c6873c5e52 // indirect
	chainmaker.org/chainmaker/vm/v2 v2.3.3-0.20230324024801-bdbc1a66a348
	github.com/docker/distribution v2.7.1+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/google/uuid v1.3.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/orcaman/concurrent-map v1.0.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.1
	go.uber.org/atomic v1.7.0
	google.golang.org/grpc v1.41.0
)

replace github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v1.0.0
