VERSION=refactor

build-test:
	cd test/scripts && ./prepare.sh

build-image:
	cd vm_mgr && go mod vendor
	cd vm_mgr && docker build -t chainmakerofficial/chainmaker-vm-docker-go:${VERSION} -f Dockerfile ./
	docker images | grep chainmaker-vm-docker-go

image-push:
	docker push chainmakerofficial/chainmaker-vm-docker-go:${VERSION}

update-gomod:
	cd vm_mgr && rm -rf vendor
	cd scripts && ./gomod_update.sh

gen-dockervm-pb:
	cd pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative dockervm_message.proto
	cd vm_mgr/pb/proto && protoc -I=. --gogofaster_out=plugins=grpc:../protogo --gogofaster_opt=paths=source_relative dockervm_message.proto

clean-test:
	cd test/scripts && ./dockerclean.sh

clean:
	cd vm_mgr && rm -rf vendor
	cd test/scripts && ./dockerclean.sh
	docker image rm chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
	docker image prune -f

ci:
	golangci-lint run ./...
#	make build-test
#	make clean

ut:
	./test/scripts/prepare.sh
	make build-image
	# UDS: docker run -itd --rm -v $(shell pwd)/data/org1/docker-go:/mount -v $(shell pwd)/log/org1/dockervm:/log --privileged --name chaimaker_vm_test chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
	docker run -itd --net=host --privileged --name chaimaker_vm_test chainmakerofficial/chainmaker-vm-docker-go:refactor
	sh ./ut_cover.sh
	docker stop chaimaker_vm_test

gomod:
	cd scripts && sh gomod_update.sh