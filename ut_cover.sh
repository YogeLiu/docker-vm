#!/usr/bin/env bash
#
# Copyright (C) BABEC. All rights reserved.
# Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
function ut_cover() {
  go test -v -coverprofile cover.out ./...
  total=$(go tool cover -func=cover.out | tail -1)
  echo ${total}
  rm cover.out
  coverage=$(echo ${total} | grep -P '\d+\.\d+(?=\%)' -o) #如果macOS 不支持grep -P选项，可以通过brew install grep更新grep
  #计算注释覆盖率，需要安装gocloc： go install github.com/hhatto/gocloc/cmd/gocloc@latest
  comment_coverage=$(gocloc --include-lang=Go --output-type=json . | jq '(.total.comment-.total.files*6)/(.total.code+.total.comment)*100')
  echo "注释率：${comment_coverage}%"

  # 如果测试覆盖率低于N，认为ut执行失败
  (( $(awk "BEGIN {print (${coverage} >= $2)}") )) || (echo "$1 单测覆盖率: ${coverage} 低于 $2%"; exit 1)
  (( $(awk "BEGIN {print (${comment_coverage} >= $3)}") )) || (echo "$1 注释覆盖率: ${comment_coverage} 低于 $3%"; exit 1)
  if test -z "$GIT_COMMITTER"; then
      echo "no committer, ignore sql insert"
  else
    echo "insert into ut_result (committer, commit_id, branch, module_name, cover_rate, ci_pass,comment_rate) values ('${GIT_COMMITTER}','${GIT_COMMIT}','${GIT_BRACH_NAME}','${1}','${coverage}',true,${comment_coverage})" | mysql -h192.168.1.121 -P3316 -uroot -p${JENKINS_MYSQL_PWD} -Dchainmaker_test
  fi

}
set -e

ut_cover . 40 15