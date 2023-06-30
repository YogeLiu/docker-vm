#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#

VERSION=v2.3.2

docker_image_name=(`docker images | grep "chainmakerofficial/chainmaker-vm-engine"`)

if [ ${docker_image_name} ]; then
  docker image rm chainmakerofficial/chainmaker-vm-engine:${VERSION}
fi

rm -rf default.log*