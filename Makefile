gen-cdm:
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto

gen-dms:
	cd vm_mgr/pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative dms_message.proto


build-vendor:
	cd vm_mgr && go mod vendor

build-image:
	cd vm_mgr && go mod vendor
	cd vm_mgr && docker build -t chainmakerofficial/chainmaker-docker-go-vm:develop_dockervm -f Dockerfile ./
	docker images | grep chainmaker-docker-go-vm

gomod:
	cd scripts && ./gomod_scripts.sh

clean:
	docker image rm chainmakerofficial/chainmaker-docker-go-vm:develop_dockervm
	docker image prune -f