VERSION=v2.3.0

BUILD_TIME = $(shell date "+%Y%m%d%H%M%S")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)

build-test:
	cd test/scripts && ./prepare.sh

build-image:
	cd vm_mgr && go mod vendor
	cd vm_mgr && docker build -t chainmaker-vm-engine \
	--build-arg BUILD_TIME=${BUILD_TIME} \
	--build-arg GIT_BRANCH=${GIT_BRANCH} \
	--build-arg GIT_COMMIT=${GIT_COMMIT} \
	-f Dockerfile ./
	docker tag chainmaker-vm-engine chainmakerofficial/chainmaker-vm-engine:${VERSION}
	docker images | grep chainmaker-vm-engine

image-push:
	docker push chainmakerofficial/chainmaker-vm-engine:${VERSION}

update-gomod:
	cd vm_mgr && rm -rf vendor
	cd scripts && ./gomod_update.sh

gen-dockervm-pb:
	cd pb/proto && protoc \
	--gogofaster_out=plugins=grpc:../protogo \
	--gogofaster_opt=paths=source_relative \
	--go-vtproto_out=../protogo --plugin protoc-gen-go-vtproto="$(GOPATH)/bin/protoc-gen-go-vtproto" \
	--go-vtproto_opt=paths=source_relative \
	--go-vtproto_opt=features=marshal+unmarshal+size+pool \
	--go-vtproto_opt=pool=chainmaker.org/chainmaker/vm-engine/pb/protogo.DockerVMMessage \
	dockervm_message.proto

	cd vm_mgr/pb/proto && protoc \
	--gogofaster_out=plugins=grpc:../protogo \
	--gogofaster_opt=paths=source_relative \
	--go-vtproto_out=../protogo --plugin protoc-gen-go-vtproto="$(GOPATH)/bin/protoc-gen-go-vtproto" \
	--go-vtproto_opt=paths=source_relative \
	--go-vtproto_opt=features=marshal+unmarshal+size+pool \
	--go-vtproto_opt=pool=chainmaker.org/chainmaker/vm-engine/vm_mgr/pb/protogo.DockerVMMessage \
	dockervm_message.proto

clean-test:
	cd test/scripts && ./dockerclean.sh

clean:
	cd vm_mgr && rm -rf vendor
	cd test/scripts && ./dockerclean.sh
	docker image rm chainmakerofficial/chainmaker-vm-engine:${VERSION}
	docker image prune -f

ci:
	golangci-lint run ./...
#	make build-test
#	make clean

gomod:
	cd scripts && sh gomod_update.sh

ut:
	./test/scripts/prepare.sh
	make build-image
	# UDS: docker run -itd --rm -v $(shell pwd)/data/org1/docker-go:/mount -v $(shell pwd)/log/org1/dockervm:/log --privileged --name chaimaker_vm_test chainmakerofficial/chainmaker-vm-engine:${VERSION}
	docker run -itd --net=host --privileged --name chaimaker_vm_test chainmakerofficial/chainmaker-vm-engine:${VERSION}
	sh ./ut_cover.sh
	docker stop chaimaker_vm_test

version:
	docker inspect chainmakerofficial/chainmaker-vm-engine:${VERSION} | jq '.[].ContainerConfig.Labels'