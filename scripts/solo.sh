VERSION=v2.2.2_qc

CURRENT_PATH=$(pwd)
TEST_PATH=${CURRENT_PATH}/test/testdata
MOUNT_PATH=${TEST_PATH}/org1/data/node1/docker-go/chain1
LOG_PATH=${TEST_PATH}/org1/log/node1/docker-go/chain1

docker run -td --rm \
  -p22359:22359 \
	-e ENV_USER_NUM=100 \
	-e ENV_TX_TIME_LIMIT=8 \
	-e ENV_MAX_CONCURRENCY=10 \
	-e ENV_LOG_LEVEL=INFO \
	-e ENV_LOG_IN_CONSOLE=false \
	-v ${MOUNT_PATH}:/mount \
	-v ${LOG_PATH}:/log \
	--privileged \
	--name chainmaker_vm_solo \
	chainmakerofficial/chainmaker-vm-docker-go:${VERSION}