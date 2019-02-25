// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package util

import (
	"fmt"
	"net"

	"github.com/spf13/viper"
)

func GetLocalIpAddr() string {
	addrs, err := net.InterfaceAddrs()

	ipaddr := ""
	if err != nil {
		fmt.Println(err)
		ipaddr = "127.0.0.1"
	} else {
		for _, address := range addrs {
			// 检查ip地址判断是否回环地址
			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					ipaddr = ipnet.IP.String()
				}
			}
		}
	}

	expectedListenAddr := viper.GetString("peer.localip")
	if len(expectedListenAddr) > 0 {
		ipaddr = expectedListenAddr
	}
	return ipaddr
}
