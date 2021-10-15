#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
set -x
BRANCH=develop

go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go get chainmaker.org/chainmaker/protocol/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/common/v2@${BRANCH}
go mod tidy