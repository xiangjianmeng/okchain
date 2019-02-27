#!/bin/bash

source okchain.profile

if [ ! -d ${BUILD_BIN} ]; then
    mkdir -p ${BUILD_BIN}
fi

if [ -f ${BUILD_BIN}/$PEER_CLIENT_BINARY ]; then
    rm ${BUILD_BIN}/$PEER_CLIENT_BINARY
fi

#CGO_CFLAGS=" " CGO_LDFLAGS="-lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy" GOBIN=$BUILD_BIN go install $OKCHAIN_PATH/$PEER_CLIENT_BINARY
CGO_CFLAGS=" " CGO_LDFLAGS="-lstdc++ -lm -lz -lbz2 -lsnappy" GOBIN=$BUILD_BIN go install $OKCHAIN_PATH/cmd/$PEER_CLIENT_BINARY


CGO_CFLAGS=" " CGO_LDFLAGS="-lstdc++ -lm -lz -lbz2 -lsnappy" GOBIN=$BUILD_BIN go install ${OKCHAIN_PATH}/cmd/okchaincli

if [ ! -f ${BUILD_BIN}/${PEER_CLIENT_BINARY} ]; then
    echo "Failed to build $PEER_CLIENT_BINARY!"
    exit -1
fi

