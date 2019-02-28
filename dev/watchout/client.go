package main

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/protos"
	"google.golang.org/grpc"
)

const (
	firstAddr = "127.0.0.1:"
	acc1      = "0x9b175d69a0A3236A1e6992d03FA0be0891D8D023" //pubKey if have verify procedure.
	acc2      = "0xD8104E3E6dE811DD0cc07d32cCcE2f4f4B38403a" //always an address.
)

func main() {
	//GetBalance ---
	for i := 0; i <= 15; i++ {
		addr := firstAddr

		addr += fmt.Sprintf("%d", 15000+i)

		func() {
			defer func() (err error) {
				if e := recover(); e != nil {
					debug.PrintStack()
				}
				return nil
			}()

			fmt.Printf("--------------node[%s]------------------\n", addr)
			PrintLatestDsBlock(addr)
			PrintLatestTxBlock(addr)
		}()
	}
}

//GetBalance get the balance of target address
func PrintBalance(nodeAddr, address string) {
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect, %v", err)
	}
	defer conn.Close()
	c := protos.NewBackendClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := c.GetAccount(ctx, &protos.AccountRequest{Address: common.HexToAddress(address).Bytes()})
	if err != nil {
		log.Fatalf("GetAccount failed, %v\n", err)
	}
	fmt.Printf("Addr: %s.. Account: %+v\n", address[:16], r.GetAccount())
}
func PrintLatestDsBlock(nodeAddr string) {
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect, %v", err)
	}
	defer conn.Close()
	c := protos.NewBackendClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := c.GetLatestDsBlock(ctx, &protos.EmptyRequest{})
	if err != nil {
		log.Printf("GetLatestDsBlock failed, %v\n", err)
		return
	}
	fmt.Println("GetLatestDsBlock:", r.NumberU64())
}

func PrintLatestTxBlock(nodeAddr string) {
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect, %v", err)
	}
	defer conn.Close()
	c := protos.NewBackendClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := c.GetLatestTxBlock(ctx, &protos.EmptyRequest{})
	if err != nil {
		log.Printf("GetLatestTxBlock failed, %v\n", err)
		return
	}
	fmt.Println("GetLatestTxBlock:", r.NumberU64())
}

func PrintTransactionsByAccount(nodeAddr, addr string, num uint64) {
	conn, err := grpc.Dial(nodeAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect, %v", err)
	}
	defer conn.Close()
	c := protos.NewBackendClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := c.GetTransactionsByAccount(ctx, &protos.GetTransactionsByAccountRequest{Address: common.FromHex(addr), Number: num})
	if err != nil {
		log.Printf("GetTransactionsByAccount failed, %v\n", err)
	}
	fmt.Println("GetTransactionsByAccount:", r.Transactions)
}
