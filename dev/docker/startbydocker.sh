#!/bin/bash

OKCHAIN_TOP=${GOPATH}/src/github.com/ok-chain/okchain
DOCKER_DATA_PATH=${OKCHAIN_TOP}/dev/docker/data
BOOTNODE_IP=172.10.83.100

while getopts "s:n:l" opt; do
  case $opt in
    l)
      echo "BOOTNODE = 1"
      BOOTNODE=1
      ;;
    s)
      echo "START = $OPTARG"
      let NEWNODE_START=$OPTARG
      ;;
    n)
      echo "NUM = $OPTARG"
      let NODE_NUM=$OPTARG
      ;;
    \?)
      echo "Invalid option: -$OPTARG"
      ;;
  esac
done

function startnode {
    idx=${1}
    isbootnode=${2}
    if [ ! -z "${isbootnode}" ];then
        hostname=bootnode.com
        str="--ip ${BOOTNODE_IP} -e OKCHAIN_PEER_MODE=lookup "
    else
        hostname=peer${idx}.com
        str="-e OKCHAIN_PEER_LOOKUPNODEURL=${BOOTNODE_IP}:15000 "
    fi

    let gossipport=15000+${idx}
    let jsonrpcport=16000+${idx}
    let debugport=25000+${idx}
    shell="docker run -d --name ${hostname} -v ${DOCKER_DATA_PATH}/${hostname}:/opt/data \
           -p ${gossipport}:15000 -p ${jsonrpcport}:16000 -p ${debugport}:25000 \
           --net okchain \
           -e OKCHAIN_PEER_GRPC_LISTENPORT=15000 \
           -e OKCHAIN_PEER_GOSSIP_LISTENPORT=15000 \
           -e OKCHAIN_PEER_ALLNODESYNC=false \
           -e OKCHAIN_PEER_DATADIR=/opt/data/ \
           -e OKCHAIN_PEER_IPCENDPOINT=/opt/data/16000.ipc \
           -e OKCHAIN_LEDGER_BASEDIR=/opt/data/db/ \
           -e OKCHAIN_PEER_LOGPATH=/opt/data/peer.log \
           -e OKCHAIN_ACCOUNT_KEYSTOREDIR=/opt/data/keystore \
           -e OKCHAIN_LEDGER_TXBLOCKCHAIN_GENESISCONF=/opt/gopath/src/github.com/ok-chain/okchain/dev/genesis.json \
           -e OKCHAIN_LOGGING_NODE=debug:ledger=debug:role=debug:state=debug:consensus=debug:gossip=info:peer=debug:txpool=debug:VM=debug:CORE=debug \
           -e OKCHAIN_PEER_LISTENADDRESS=0.0.0.0:15000 \
           -e OKCHAIN_PEER_DEBUG_LISTENADDRESS=0.0.0.0:25000 \
           -e OKCHAIN_PEER_JSONRPCADDRESS=0.0.0.0:16000 \
           -e OKCHAIN_PEER_ROLEID=${hostname} "
     endstr="okchain/testnet-okchaind okchaind node start "
     shell=${shell}${str}${endstr}
     #echo $shell
     $shell

}


function main {
    if [ ! -z ${BOOTNODE} ]; then
        echo "removing docker_all_node"
        docker rm -f $(docker ps -aq)
        echo -e "remove docker_all_node done\n"

        echo "removing network"
        docker network rm okchain
        echo -e "remove network done\n"

        echo "creating network"
        docker network create --driver bridge --subnet=172.10.83.0/24 okchain
        echo -e "create network done\n"

        echo "removing data_dir"
        rm -rf ${DOCKER_DATA_PATH}
        echo -e "remove data_dir done\n"

        echo "start bootnode"
        startnode 0 1
        echo -e "start bootnode done\n"
    fi

    echo "start node"
    if [ -z ${NEWNODE_START} ]; then
        let NEWNODE_START=1
        let NODE_NUM=10
    fi
    let node_n=${NEWNODE_START}+${NODE_NUM}
    for ((index=${NEWNODE_START}; index<${node_n}; index++)) do
        startnode ${index}
    done
    echo "start node done"
}

main