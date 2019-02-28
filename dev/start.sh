#!/bin/bash

export http_proxy=""
export https_proxy=""
export ftp_proxy=""
export no_proxy=""
export HTTP_PROXY=""
export HTTPS_PROXY=""
export FTP_PROXY=""
export NO_PROXY=""

export OKCHAIN_PATH=github.com/ok-chain/okchain
export OKCHAIN_TOP=${GOPATH}/src/github.com/ok-chain/okchain
export BUILD_BIN=${OKCHAIN_TOP}/build/bin
export PEER_CLIENT_BINARY=okchaind
export OKCHAIN_DEV_SCRIPT_TOP=$OKCHAIN_TOP/dev
export OKCHAIN_BASE_PORT=15000


while getopts "s:n:lk" opt; do
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
    k)
      echo "KILL = 1"
      let KILL=1
      ;;
    \?)
      echo "Invalid option: -$OPTARG"
      ;;
  esac
done

function killbyname {
    NAME=$1
    MYNAME="start.sh"

    if [ $# -eq 0 ]; then
        echo "$MYNAME <process name>"
        exit
    fi

    ps -ef|grep "$NAME"|grep -v grep |grep -v $MYNAME |awk '{print "kill -9 "$2"  "$8}'
    ps -ef|grep "$NAME"|grep -v grep |grep -v $MYNAME |awk '{print "kill -9 "$2}' | sh
    echo "All <$NAME*> killed!"
}

function startnode {
    PEER_ID=$1
    PEER_ROLE=$2

    if [ ! -z "${PEER_ROLE}" ];then
        export OKCHAIN_PEER_MODE=lookup
    else
        export OKCHAIN_PEER_MODE=""
        export OKCHAIN_PEER_LOOKUPNODEURL=${ENV_LOCAL_HOST_IP}":"${OKCHAIN_BASE_PORT}
    fi
    
    let PEER_DEBUG_LISTEN_PORT=${OKCHAIN_BASE_PORT}+${PEER_ID}+10000
    let JSONRPC_LISTEN_PORT=${OKCHAIN_BASE_PORT}+${PEER_ID}+1000
    let GOSSIP_LISTEN_PORT=${OKCHAIN_BASE_PORT}+${PEER_ID}
    # listen port
    export OKCHAIN_PEER_LISTENADDRESS="0.0.0.0":${GOSSIP_LISTEN_PORT}
    export OKCHAIN_PEER_DEBUG_LISTENADDRESS="0.0.0.0":${PEER_DEBUG_LISTEN_PORT}
    export OKCHAIN_PEER_JSONRPCADDRESS="0.0.0.0":${JSONRPC_LISTEN_PORT}
    export OKCHAIN_PEER_GOSSIP_LISTENPORT=${GOSSIP_LISTEN_PORT}
    export OKCHAIN_PEER_GRPC_LISTENPORT=${GOSSIP_LISTEN_PORT}

    FULL_NODE_ID=okchain_node_${PEER_ID}
    export OKCHAIN_PEER_DATADIR=${OKCHAIN_TOP}/dev/data/${FULL_NODE_ID}
    export OKCHAIN_LEDGER_TXBLOCKCHAIN_GENESISCONF=${OKCHAIN_TOP}/dev/genesis.json
    export OKCHAIN_LEDGER_BASEDIR=${OKCHAIN_PEER_DATADIR}/db/
    export OKCHAIN_PEER_ROLEID=${FULL_NODE_ID}
    export OKCHAIN_PEER_LOGPATH=${OKCHAIN_PEER_DATADIR}/${FULL_NODE_ID}.log
    # rm -rf $OKCHAIN_PEER_LOGPATH
    export OKCHAIN_ACCOUNT_KEYSTOREDIR=${OKCHAIN_PEER_DATADIR}/keystore
    export OKCHAIN_PEER_IPCENDPOINT=${OKCHAIN_TOP}/dev/data/jsonrpc_ipc_endpoint/${JSONRPC_LISTEN_PORT}.ipc
    if [ ! -d ${OKCHAIN_LEDGER_BASEDIR} ]; then
        mkdir -p ${OKCHAIN_LEDGER_BASEDIR}
    fi

    if [ -d ${OKCHAIN_TOP}/dev/keystore ]; then
        cp -rf ${OKCHAIN_TOP}/dev/keystore ${OKCHAIN_PEER_DATADIR}
    fi

    export STDOUT_LOG_FILE=${OKCHAIN_PEER_DATADIR}/stdout.log
    export OKCHAIN_LOGGING_NODE=debug:ledger=debug:role=debug:state=debug:consensus=debug:gossip=info:peer=debug:txpool=debug:VM=debug:CORE=debug

    if [ ! -f ${BUILD_BIN}/${FULL_NODE_ID} ]; then
        cd ${BUILD_BIN}
        ln -snf ${PEER_CLIENT_BINARY} ${FULL_NODE_ID}
        cd ../..
    fi

    echo ${FULL_NODE_ID}
    echo '' > ${STDOUT_LOG_FILE}
    GODEBUG=gctrace=1 nohup ${BUILD_BIN}/${FULL_NODE_ID} node start >> ${STDOUT_LOG_FILE} 2>>${STDOUT_LOG_FILE} &
}

function main {
   if [ ! -z ${KILL} ]; then
        killbyname okchain_node_
        exit
   fi

   if [ ! -z ${BOOTNODE} ]; then
        killbyname okchain_node_

        echo "removing data"
        rm -rf data/
        echo "data remove done"

        echo "start bootnode ==========="
        startnode 0 1
        sleep 5
        echo -e "start bootnode done ======\n"
    fi

    echo "start node ==========="
    if [ -z ${NEWNODE_START} ]; then
        let NEWNODE_START=1
        let NODE_NUM=10
    fi

    let node_n=${NEWNODE_START}+${NODE_NUM}
    for ((index=${NEWNODE_START}; index<${node_n}; index++)) do
        startnode ${index}
    done
    echo "start node done ======"
}

main
