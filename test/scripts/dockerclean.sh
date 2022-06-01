#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

docker stop chain1-chainmaker-vm-docker-go-container
docker rm chain1-chainmaker-vm-docker-go-container
# docker rmi chainmakerofficial/chainmaker-vm-docker-go:develop

docker image prune -f

docker ps -a
#docker images

rm -fr ../testdata/org1
rm -rf ../testdata/tmp
rm -fr ../testdata/log
rm -fr ../default.log*
rm -rf ../testdata/tmp
