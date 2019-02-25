package config

import (
	"fmt"
	"net"
	"testing"
)

func TestLocalAddr(t *testing.T) {

	addrs, err := net.InterfaceAddrs()

	if err != nil {
		t.Fatalf("verify error: %s", err)
	} else {

		for _, address := range addrs {

			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {

				fmt.Printf("%s\n", ipnet.IP.String())
				t.Logf(ipnet.IP.String())

			}
		}
	}

}
