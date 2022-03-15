VERSION=develop

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

gen-dockervm:
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
	make build-test
	golangci-lint run ./...
	go test -v ./...
	make clean

gomod:
	cd scripts && sh gomod_update.sh