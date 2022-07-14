#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

docker stop chaimaker_vm_test
docker rm chaimaker_vm_test
# docker rmi chainmakerofficial/chainmaker-vm-engine:develop

docker image prune -f

docker ps -a
#docker images

rm -fr ../testdata/org1
rm -rf ../testdata/tmp
rm -fr ../testdata/log
rm -fr ../default.log*
rm -rf ../testdata/tmp
