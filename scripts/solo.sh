VERSION=develop

CURRENT_PATH=$(pwd)
TEST_PATH=${CURRENT_PATH}/test/testdata
LOG_PATH=${TEST_PATH}/org1/log/node1/dockervm/chain1

docker run -td --rm \
  -p22359:22359 \
	-e ENV_USER_NUM=100 \
	-e ENV_TX_TIME_LIMIT=2 \
	-e ENV_MAX_CONCURRENCY=10 \
	-e ENV_LOG_LEVEL=DEBUG \
	-e ENV_LOG_IN_CONSOLE=true \
	-v ${LOG_PATH}:/log \
	--privileged \
	--name chainmaker_vm_solo \
	chainmakerofficial/chainmaker-vm-docker-go:${VERSION}
