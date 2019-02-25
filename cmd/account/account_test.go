// Copyright The go-okchain Authors 2018,  All rights reserved.

package account

import (
	"testing"

	"fmt"

	"github.com/golang/protobuf/proto"
)

func BenchmarkSendRawTransaction(b *testing.B) {
	//from_str := "0xbcf72fc47d4aeec195dba5e0e26cea2b75416ec4"
	//from := common.BytesToAddress(common.FromHex(from_str))
	//to_str := "0x3d651eb9ae5626260e2468d589aee62b86953046"
	//to := common.BytesToAddress(common.FromHex(to_str))
	//var gas uint64 = 1000000
	//var gasPrice uint64 = 1
	//var value uint64 = 1
	//var nonce uint64 = 103
	//passwd := "123"
	//args := server.SendTxArgs{
	//	From:     from,
	//	To:       &to,
	//	Gas:      &gas,
	//	GasPrice: &gasPrice,
	//	Value:    &value,
	//	Nonce:    &nonce,
	//}
	////nodeaddr := "http://192.168.168.68:16014"
	//viper.Set("account.keystoreDir", "../keystore")
	////hash, err := sendRawTransaction(args, passwd, nodeaddr)
	////if err != nil {
	////	panic(err)
	////}
	////fmt.Println(hash.String())
	proto.String("")
	fmt.Println(proto.Uint64(0))
}
