# vm-docker-go 单独部署

## 1. 新增配置

新增配置

docker_vm_host:  合约管理服务
docker_vm_port:  合约管理服务端口号

删除了chainmaker配置里用于配置合约管理container的参数

```yml
vm:
  enable_dockervm: true
  dockervm_mount_path: ../data/org1/docker-go     
  docker_vm_host: 10.197.78.11
  docker_vm_port: 22356
  max_send_msg_size: 20
  max_recv_msg_size: 20
```

## 2. 启动方式

docker 参数，如果不设置会采用默认参数：

```
"ENV_ENABLE_UDS=false",
"ENV_USER_NUM=100",
"ENV_TX_TIME_LIMIT=2",
"ENV_LOG_LEVEL=DEBUG",
"ENV_LOG_IN_CONSOLE=false",
"ENV_MAX_CONCURRENCY=50",
"ENV_Docker_VM_Port=22359",
```

启动命令

```shell
docker run -it -e ENV_LOG_LEVEL=DEBUG -e ENV_LOG_IN_CONSOLE=true -p22359:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.0_alpha_qc
```



启动四个容器的脚本，分别监听22351 - 22354:


```shell
docker run -d -e ENV_ENABLE_UDS=false -e ENV_USER_NUM=100 -e ENV_TX_TIME_LIMIT=2 -e ENV_LOG_LEVEL=DEBUG -e ENV_LOG_IN_CONSOLE=false -e ENV_MAX_CONCURRENCY=50 -p22351:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.0_alpha_qc

docker run -d -e ENV_ENABLE_UDS=false -e ENV_USER_NUM=100 -e ENV_TX_TIME_LIMIT=2 -e ENV_LOG_LEVEL=DEBUG -e ENV_LOG_IN_CONSOLE=false -e ENV_MAX_CONCURRENCY=50 -p22352:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.0_alpha_qc

docker run -d -e ENV_ENABLE_UDS=false -e ENV_USER_NUM=100 -e ENV_TX_TIME_LIMIT=2 -e ENV_LOG_LEVEL=DEBUG -e ENV_LOG_IN_CONSOLE=false -e ENV_MAX_CONCURRENCY=50 -p22353:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.0_alpha_qc

docker run -d -e ENV_ENABLE_UDS=false -e ENV_USER_NUM=100 -e ENV_TX_TIME_LIMIT=2 -e ENV_LOG_LEVEL=DEBUG -e ENV_LOG_IN_CONSOLE=false -e ENV_MAX_CONCURRENCY=50 -p22354:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.0_alpha_qc

```

