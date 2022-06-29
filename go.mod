module chainmaker.org/chainmaker/vm-docker-go/v2

go 1.16

require (
	chainmaker.org/chainmaker/chainconf/v2 v2.2.3-0.20220607072052-653ff1f72ed5 // indirect
	chainmaker.org/chainmaker/common/v2 v2.2.2-0.20220628025818-290c39d5f1c8
	chainmaker.org/chainmaker/localconf/v2 v2.2.2-0.20220607115425-03689a750027
	chainmaker.org/chainmaker/pb-go/v2 v2.2.2-0.20220629075646-a48b109a5aad
	chainmaker.org/chainmaker/protocol/v2 v2.2.3-0.20220628032755-cebcbb60be67
	chainmaker.org/chainmaker/utils/v2 v2.2.3-0.20220615103110-6849a8ada870
	chainmaker.org/chainmaker/vm/v2 v2.2.3-0.20220629083133-95b5793b6908
	github.com/docker/distribution v2.7.1+incompatible
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	go.uber.org/atomic v1.7.0
	google.golang.org/grpc v1.41.0
)

replace github.com/linvon/cuckoo-filter => chainmaker.org/third_party/cuckoo-filter v0.0.0-20220601084543-8591df469f8f
