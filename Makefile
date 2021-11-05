vm-docker-go:
	make build-image

gen-cdm:
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto
	cd vm_mgr/pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto

gen-dms:
	cd vm_mgr/pb_sdk/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative dms_message.proto


build-vendor:
	cd vm_mgr && go mod vendor

build-image:
	cd vm_mgr && go mod vendor
	cd vm_mgr && docker build -t chainmakerofficial/chainmaker-vm-docker-go:develop -f Dockerfile ./
	docker images | grep chainmaker-vm-docker-go

image-push:
	docker push chainmakerofficial/chainmaker-vm-docker-go:develop

gomod:
	cd scripts && ./gomod_scripts.sh

clean:
	cd vm_mgr && rm -rf vendor
	docker image rm chainmakerofficial/chainmaker-vm-docker-go:develop
	docker image prune -f
