# docker部署OKChain网络

## docker-compose

1. 执行`docker-compose up -d`
2. 稍等几秒，可执行`docker ps -a`查看到节点容器
3. 如需加入新节点，修改`newnode-compose.yaml`中的ID、port等信息，执行`docker-compose -f newnode-compose.yaml up -d`


## start.sh 自定义节点数量

1. 执行`./start.sh -l -s 10 -n 20`，其中，

    `-l`表示是否启动Bootnode节点，如果是第一次启动网络则必须使用该选项，如果是在已有网络中加入新节点，则不能使用该选项

    `-s`表示非Bootnode节点的起始编号，用于节点容器名和对外映射端口，容器名为`peer{s}.com`依次递增，端口为`15000+{s}`依次递增；Bootnode节点名固定为`Bootnode.com`，端口为`15000`

    `-n`表示非Bootnode节点数量