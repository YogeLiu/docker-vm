VERSION=v2.2.2_qc


DATETIME=$(shell date "+%Y%m%d%H%M%S")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)
CURRENT_PATH = $(shell pwd)
VERSION_HOME=${CURRENT_PATH}/vm_mgr/config

GOLDFLAGS += -X "${VERSION_HOME}.CurrentVersion=${VERSION}"
GOLDFLAGS += -X "${VERSION_HOME}.BuildDateTime=${DATETIME}"
GOLDFLAGS += -X "${VERSION_HOME}.GitBranch=${GIT_BRANCH}"
GOLDFLAGS += -X "${VERSION_HOME}.GitCommit=${GIT_COMMIT}"

test1:
	echo ${VERSION_HOME}

build-test:
	cd test/scripts && ./prepare.sh

build-image:
	cd vm_mgr && go mod vendor
	cd vm_mgr && docker build -t chainmakerofficial/chainmaker-vm-docker-go:${VERSION} -f Dockerfile ./
	docker images | grep chainmaker-vm-docker-go
	docker image prune -f

image-push:
	docker push chainmakerofficial/chainmaker-vm-docker-go:${VERSION}

update-gomod:
	cd vm_mgr && rm -rf vendor
	cd scripts && ./gomod_update.sh

gen-cdm:
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto
	cd vm_mgr/pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative cdm_message.proto

gen-dms:
	cd vm_mgr/pb_sdk/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative dms_message.proto

clean-test:
	cd test/scripts && ./dockerclean.sh

clean:
	cd vm_mgr && rm -rf vendor
	cd test/scripts && ./dockerclean.sh
	docker image rm chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
	docker image prune -f

ci:
	make build-test
	golangci-lint run ./...
	make clean

gomod:
	go get chainmaker.org/chainmaker/common/v2@develop
	go get chainmaker.org/chainmaker/localconf/v2@$(VERSION)
	go get chainmaker.org/chainmaker/logger/v2@develop
	go get chainmaker.org/chainmaker/pb-go/v2@$(VERSION)
	go get chainmaker.org/chainmaker/protocol/v2@develop
	go get chainmaker.org/chainmaker/utils/v2@develop
	go mod tidy

ut:
	./test/scripts/prepare.sh
	make build-image
	docker run -itd --rm -p22359:22359 -e ENV_LOG_IN_CONSOLE=true --privileged --name chaimaker_vm_test chainmakerofficial/chainmaker-vm-docker-go:v2.2.2_qc
	./ut_cover.sh
	docker stop chaimaker_vm_test

solo:
	./scripts/solo.sh

solo-stop:
	docker stop chainmaker_vm_solo
