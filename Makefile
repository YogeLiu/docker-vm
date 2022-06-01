VERSION=develop
VERSION230=v2.3.0_qc

BUILD_TIME = $(shell date "+%Y%m%d%H%M%S")
GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT = $(shell git log --pretty=format:'%h' -n 1)

build-test:
	cd test/scripts && ./prepare.sh

build-image:
	cd vm_mgr && go mod vendor
	cd vm_mgr && docker build -t chainmakerofficial/chainmaker-vm-docker-go:${VERSION} \
	--build-arg BUILD_TIME=${BUILD_TIME} \
	--build-arg GIT_BRANCH=${GIT_BRANCH} \
	--build-arg GIT_COMMIT=${GIT_COMMIT} \
	-f Dockerfile ./
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

ut:
	./test/scripts/prepare.sh
	make build-image
	docker run -itd --rm -p22359:22359 -e ENV_USER_NUM=100 -e ENV_MAX_CONCURRENCY=20 --privileged --name chaimaker_vm_test chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
	./ut_cover.sh
	docker stop chaimaker_vm_test

gomod:
	cd scripts && sh gomod_update.sh

version:
	docker inspect chainmakerofficial/chainmaker-vm-docker-go:${VERSION} | jq '.[].ContainerConfig.Labels'

solo:
	./scripts/solo.sh

solo-stop:
	docker stop chainmaker_vm_solo

