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

# ENV_LOCAL_HOST_IP 指定本地 LAN IP, 也可在 ~/.bashrc 设置
export ENV_LOCAL_HOST_IP=`ifconfig  | grep 192.168 | awk '{print $2}' | cut -d: -f2`


# 多网卡主机上需要用这个环境变量显示指定该主机使用哪个IP和其他peer通信
# 单网卡机器不需指定

export OKCHAIN_PEER_LOCALIP=${ENV_LOCAL_HOST_IP}
