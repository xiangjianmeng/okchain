// Copyright The go-okchain Authors 2018,  All rights reserved.

package notify

import (
	"context"
	"errors"
	"strings"
)

import (
	"github.com/spf13/cobra"
	"golang.org/x/proto"
)

import (
	"strconv"

	"fmt"

	pb "github.com/ok-chain/okchain/protos"
	"google.golang.org/grpc"
)

func setPrimaryCmd() *cobra.Command {
	return nodeSetPrimaryCmd
}

var nodeSetPrimaryCmd = &cobra.Command{
	Use:   "setprimary",
	Short: "Set the ds primary node",
	Long:  "Set the primary node of the ds committee",
	RunE: func(cmd *cobra.Command, args []string) error {
		return SetPrimary(cmd, args)
	},
}

func SetPrimary(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return errors.New("please input address & port as parameter")
	}
	addrPort := strings.Split(args[0], ":")
	if len(addrPort) != 2 {
		return errors.New("please input first parameter format as '127.0.0.1:8000'")
	}

	primary := strings.Split(args[1], ":")
	if len(primary) != 2 {
		return errors.New("please input second parameter format as '127.0.0.1:8000'")
	}

	message := &pb.Message{Type: pb.Message_DS_SetPrimary}
	message.Timestamp = pb.CreateUtcTimestamp()
	message.Signature = []byte("xxxxx")
	peer := &pb.PeerEndpoint{}
	peer.Pubkey = []byte("this is a pubkey")
	peer.Address = primary[0]
	port, _ := strconv.Atoi(args[1])
	peer.Port = uint32(port)
	id := &pb.PeerID{}
	id.Name = args[1]
	peer.Id = id
	data, err := proto.Marshal(peer)
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

	//fmt.Printf("setprimary resp: %v\n", r)

	if r.Code == 1 {
		fmt.Printf("send setprimary msg to %s\n", args[0])
		return nil
	} else {
		return errors.New("response error")
	}
}
