#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

VERSION=v2.2.0

docker_image_name=`docker images | grep "chainmakerofficial/chainmaker-vm-docker-go"`

if [ ${docker_image_name} ]; then
  docker image rm chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
  rm -fr ../testdata/org1
  rm -fr ../testdata/log
  rm -fr ../default.log*
fi

cd ../../vm_mgr && go mod vendor
docker build -t chainmakerofficial/chainmaker-vm-docker-go:${VERSION} -f Dockerfile ./
