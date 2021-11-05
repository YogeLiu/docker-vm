#!/bin/bash

docker_image_name=(`docker images | grep "chainmakerofficial/chainmaker-vm-docker-go"`)

if [ ${docker_image_name} ]; then
  echo "remove previous one"
  docker image rm chainmakerofficial/chainmaker-vm-docker-go:develop
fi

echo "create new one"
echo ${PWD}
cd ../../vm_mgr && go mod vendor
docker build -t chainmakerofficial/chainmaker-vm-docker-go:develop -f Dockerfile ./
