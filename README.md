# vm-docker-go 单独部署

## 1. 配置说明

新增配置

docker_vm_host:  合约管理服务
docker_vm_port:  合约管理服务端口号

删除了chainmaker配置里用于配置合约管理container的参数

```yml
vm:
  enable_dockervm: true
  uds_open: false
#  dockervm_mount_path 存放合约文件和socket文件。如果开启本地socket，需要将该路径中的以chainid命名的文件夹 mount 到容器里的 /mount 目录下
  dockervm_mount_path: ../data/org1/docker-go  
  docker_vm_host: 10.197.78.11
  docker_vm_port: 22359
  max_send_msg_size: 20
  max_recv_msg_size: 20
```

## 2. 部署启动流程


### 2.1. 启动合约服务容器

1. 打包合约服务的镜像
```shell
make build-images 
```

2. 参数说明：
容器中的参数，如果不设置会采用默认参数，默认如下

```
# 是否开启unix domain socket 通信
ENV_ENABLE_UDS=false
# 最大用户数，同样约束了最大进程数量
ENV_USER_NUM=100
# 交易过期时间，单位（s）
ENV_TX_TIME_LIMIT=2
# 日志等级
ENV_LOG_LEVEL=INFO
# 日志是否打印到标准输出
ENV_LOG_IN_CONSOLE=false
# 每个合约最大启用的进程数量
ENV_MAX_CONCURRENCY=50
# 监听的端口。如果启用unix domain socket，则监听 /mount/sock/cdm.sock 路径。
ENV_VM_SERVICE_PORT=22359
# 是否开启 pprof
ENV_ENABLE_PPROF=
# 指定 pprof 端口
ENV_PPROF_PORT=
ENV_MAX_SEND_MSG_SIZE= 
ENV_MAX_RECV_MSG_SIZE=
```

3. 启动命令

容器的运行需要privileged的权限，启动命令添加 --privileged 参数

3.1 以tcp方式启动：
参数中需要再添加对外暴露的端口映射

```shell
docker run -it -p22359:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1
```

例如 启动四个容器的脚本，分别监听22351 - 22354，并打印容器输出到标准输出:

```shell
docker run -d -e ENV_LOG_IN_CONSOLE=true -p22351:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

docker run -d -e ENV_LOG_IN_CONSOLE=true -p22352:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

docker run -d -e ENV_LOG_IN_CONSOLE=true -p22353:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

docker run -d -e ENV_LOG_IN_CONSOLE=true -p22354:22359 --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

```
3.1 以uds方式启动：
参数中需要再添加：
1. 启动uds的环境变量 -e ENV_ENABLE_UDS=true
2. 通过 -v 指定本地合约文件和socket文件的映射

```shell
docker run -it -e ENV_ENABLE_UDS=true -v /root/chainmaker.org/chainmaker-go/build/release/chainmaker-v2.2.1-wx-org.chainmaker.org/data/wx-org.chainmaker.org/docker-go/chain1:/mount --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1
```


例如 启动四个容器的脚本:
```shell
docker run -it -e ENV_ENABLE_UDS=true -v /root/chainmaker.org/chainmaker-go/build/release/chainmaker-v2.2.1-wx-org1.chainmaker.org/data/wx-org1.chainmaker.org/docker-go/chain1:/mount --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

docker run -it -e ENV_ENABLE_UDS=true -v /root/chainmaker.org/chainmaker-go/build/release/chainmaker-v2.2.1-wx-org2.chainmaker.org/data/wx-org2.chainmaker.org/docker-go/chain1:/mount --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

docker run -it -e ENV_ENABLE_UDS=true -v /root/chainmaker.org/chainmaker-go/build/release/chainmaker-v2.2.1-wx-org3.chainmaker.org/data/wx-org3.chainmaker.org/docker-go/chain1:/mount --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

docker run -it -e ENV_ENABLE_UDS=true -v /root/chainmaker.org/chainmaker-go/build/release/chainmaker-v2.2.1-wx-org4.chainmaker.org/data/wx-org4.chainmaker.org/docker-go/chain1:/mount --privileged chainmakerofficial/chainmaker-vm-docker-go:v2.2.1

```

### 2.2. 配置启动 chainmaker

#### 2.2.1 tcp方式
保持 uds_open 为false，并配置docker_vm_host和docker_vm_port的值
```
uds_open: false
docker_vm_host: 10.197.78.11
docker_vm_port: 22359
```
#### 2.2.2 uds方式

配置中 如果开启 uds
请保证 dockervm_mount_path 的路径与 容器中mount进容器中的路径一致。

```
uds_open: true
dockervm_mount_path: ../data/org1/docker-go  
```

正常启动chainmaker: ./chainmaker strat -c config 