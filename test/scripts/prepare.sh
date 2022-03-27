#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

VERSION=v2.2.1
TESTCONTAINERNAME=chaimaker_vm_test

docker_image_name=(`docker images | grep "chainmakerofficial/chainmaker-vm-docker-go"`)

if [ ! "$(docker ps -q -f name=${TESTCONTAINERNAME})" ]; then
  if [ "$(docker ps -aq -f status=running -f name=${TESTCONTAINERNAME})" ]; then
    # stop container
    docker stop ${TESTCONTAINERNAME}
    sleep 2
  fi
  if [ "$(docker ps -aq -f status=exited -f name=${TESTCONTAINERNAME})" ]; then
    # clean container
    docker rm ${TESTCONTAINERNAME}
    sleep 2
  fi
fi

if [ ${docker_image_name} ]; then
  docker image rm chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
  rm -fr ../testdata/org1
  rm -fr ../testdata/log
  rm -fr ../default.log*
fi

