#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

VERSION=v2.3.2
TESTCONTAINERNAME=chaimaker_vm_test

docker_image_name=`docker images | grep "chainmakerofficial/chainmaker-vm-engine"`

if [ "$(docker ps -q -f status=running -f name=${TESTCONTAINERNAME})" ]; then
  echo "stop container"
  docker stop ${TESTCONTAINERNAME}
  sleep 3
fi

if [ "$(docker ps -aq -f status=exited -f name=${TESTCONTAINERNAME})" ]; then
  echo "clean container"
  docker rm ${TESTCONTAINERNAME}
  sleep 2
fi

if [ "${docker_image_name}" ]; then
  docker image rm chainmakerofficial/chainmaker-vm-engine:${VERSION}
  rm -fr ../testdata/org1
  rm -fr ../testdata/log
  rm -fr ../default.log*
fi