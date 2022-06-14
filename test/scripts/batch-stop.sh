#!/bin/bash
#
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
 
read -r -p "docker container number: " num

for(( i = 1; i <= $num; i++ ))
do
    CONTAINER_NAME=chainmaker-docker-vm-$i
    rm -rf ./org$i
    docker stop "$CONTAINER_NAME"
done

docker ps -a