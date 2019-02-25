// Copyright The go-okchain Authors 2018,  All rights reserved.

package notify

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"strconv"
	"strings"
)

import (
	"github.com/spf13/cobra"
	"golang.org/x/proto"
)

import (
	"fmt"

	pb "github.com/ok-chain/okchain/protos"
	"google.golang.org/grpc"
)

func startPoWCmd() *cobra.Command {
	return nodeStartPoWCmd
}

var nodeStartPoWCmd = &cobra.Command{
	Use:   "startpow",
	Short: "Start pow",
	Long:  "Start pow calculate for ds leader election",
	RunE: func(cmd *cobra.Command, args []string) error {
		return StartPoW(cmd, args)
	},
}

func StartPoW(cmd *cobra.Command, args []string) error {

	if len(args) != 1 {
		return errors.New("please input address & port as parameter")
	}
	addrPort := strings.Split(args[0], ":")
	if len(addrPort) != 2 {
		return errors.New("please input parameter format as '127.0.0.1:8000'")
	}

	message := &pb.Message{Type: pb.Message_Node_ProcessStartPoW}
	message.Timestamp = pb.CreateUtcTimestamp()
	startPow := &pb.StartPoW{}
	startPow.BlockNum = 0
	startPow.Difficulty = 100000
	rand1 := big.NewInt(rand.Int63())
	rand2 := big.NewInt(rand.Int63())
	startPow.Rand1 = rand1.String()
	startPow.Rand2 = rand2.String()
	peer := &pb.PeerEndpoint{}

	peer.Address = addrPort[0]
	peer.Pubkey = []byte("this is a pubkey")
	port, err := strconv.Atoi(addrPort[1])
	if err != nil {
		return err
	}
	peer.Port = uint32(port)
	id := &pb.PeerID{Name: args[0]}
	peer.Id = id
	startPow.Peer = peer

	data, err := proto.Marshal(startPow)
	if err != nil {
		return err
	}
	message.Payload = data

	conn, err := grpc.Dial(args[0], grpc.WithInsecure())

	if err != nil {
		return errors.New("failed to connect server")
	}
	defer conn.Close()
	c := pb.NewBackendClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r, err := c.Notify(ctx, message)

	if err != nil {
		fmt.Printf("%s", err)
		return errors.New("Failed to notify peer")

	}

	//fmt.Printf("startpow resp: %v\n", r)

	if r.Code == 1 {
		fmt.Printf("send startpow msg to %s\n", args[0])
		return nil
	} else {
		return errors.New("response error")
	}
}
