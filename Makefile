VERSION=v2.1.0

build-test:
	cd test/scripts && ./prepare.sh

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

update-gomod:
	cd scripts && ./gomod_scripts.sh

clean:
	cd vm_mgr && rm -rf vendor
	docker image rm chainmakerofficial/chainmaker-vm-docker-go:develop
	docker image prune -f
	cd test/scripts && ./dockerclean.sh

ci:
	make build-test
	golangci-lint run ./...
	go test ./...
	make clean

gomod:
	go get chainmaker.org/chainmaker/localconf/v2@$(VERSION)
	go get chainmaker.org/chainmaker/logger/v2@$(VERSION)
	go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
	go get chainmaker.org/chainmaker/protocol/v2@v2.1.0_alpha_fix
	go get chainmaker.org/chainmaker/utils/v2@$(VERSION)
	go mod tidy
	cat go.mod|grep chainmaker




