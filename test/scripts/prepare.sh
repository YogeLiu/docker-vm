#!/bin/bash

VERSION=v2.1.0

docker_image_name=(`docker images | grep "chainmakerofficial/chainmaker-vm-docker-go"`)

if [ ${docker_image_name} ]; then
  docker image rm chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
fi

cd ../../vm_mgr && go mod vendor
docker build -t chainmakerofficial/chainmaker-vm-docker-go:${VERSION} -f Dockerfile ./
