#
# Copyright (C) BABEC. All rights reserved.
# SPDX-License-Identifier: Apache-2.0
#
set -x
BRANCH=v2.1.0
ALPHA=v2.1.0_alpha_fix

cd ../
go get chainmaker.org/chainmaker/localconf/v2@${BRANCH}
go get chainmaker.org/chainmaker/logger/v2@${BRANCH}
go get chainmaker.org/chainmaker/pb-go/v2@${BRANCH}
go get chainmaker.org/chainmaker/protocol/v2@${ALPHA}
go get chainmaker.org/chainmaker/utils/v2@${BRANCH}
go mod tidy


cd ./vm_mgr
go get chainmaker.org/chainmaker/protocol/v2@${ALPHA}
go mod tidy
