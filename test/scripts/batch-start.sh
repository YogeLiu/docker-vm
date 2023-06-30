#!/bin/bash
#
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

read -r -p "docker container number: " num

for(( i = 1; i <= $num; i++ ))
do
    LOG_PATH=$(pwd)/org$i/log
    MOUNT_PATH=$(pwd)/org$i/docker-go
    LOG_LEVEL=INFO
    EXPOSE_PORT=$((22351+$i-1))
    RUNTIME_PORT=$((32351+$i-1))
    CONTAINER_NAME=chainmaker-docker-vm-$i
    IMAGE_NAME="chainmakerofficial/chainmaker-vm-engine:v2.3.2"
    
    if  [ ! -d "$MOUNT_PATH" ];then
    mkdir -p "$MOUNT_PATH"
      if [ $? -ne 0 ]; then
        echo "create mount path failed. exit"
        exit 1
      fi
    fi

    if  [ ! -d "$LOG_PATH" ];then
    mkdir -p "$LOG_PATH"
      if [ $? -ne 0 ]; then
        echo "create log path failed. exit"
        exit 1
      fi
    fi

    docker run -itd --net=host \
    -v "$MOUNT_PATH":/mount \
    -v "$LOG_PATH":/log \
    -e CHAIN_RPC_PORT="$EXPOSE_PORT" \
    -e SANDBOX_RPC_PORT="$RUNTIME_PORT" \
    --name "$CONTAINER_NAME" \
    --privileged $IMAGE_NAME
done

docker ps -a