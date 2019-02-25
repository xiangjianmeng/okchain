// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"fmt"
	"os"
	"strings"

	"github.com/ok-chain/okchain/rpc"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

var client *rpc.Client

func getClient() *rpc.Client {
	if client != nil {
		return client
	}
	client = NewClient()
	return client
}

func NewClient() *rpc.Client {
	//fmt.Printf("connect to progress ==>jsonrpc port %s\n", viper.GetString("jsonrpcport"))
	url := viper.GetString("url")
	if !strings.Contains(url, "http://") {
		fmt.Println(errors.New("invalid url"))
		os.Exit(1)
	}
	client, err := rpc.DialHTTP(url)
	//var dir string
	//if viper.GetBool("istestnet") {
	//	dir = viper.GetString("peer.topDir") + "/testnet/data/jsonrpc_ipc_endpoint/" + viper.GetString("jsonrpcport") + ".ipc"
	//} else {
	//	dir = viper.GetString("peer.ipcendpointdir") + "/" + viper.GetString("jsonrpcport") + ".ipc"
	//}
	//client, err := rpc.Dial(dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return client
}
