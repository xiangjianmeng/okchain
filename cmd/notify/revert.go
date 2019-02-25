// Copyright The go-okchain Authors 2019,  All rights reserved.

package notify

import (
	"context"
	"errors"
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

func getRevertCmd() *cobra.Command {
	return revertCmd
}

var revertCmd = &cobra.Command{
	Use:   "revert",
	Short: "revert node state",
	Long:  "revert node state",
	RunE: func(cmd *cobra.Command, args []string) error {
		return sendRevertMsg(cmd, args)
	},
}

func sendRevertMsg(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("please input address & port as parameter")
	}
	message := &pb.Message{Type: pb.Message_Node_RevertState}
	message.Timestamp = pb.CreateUtcTimestamp()

	data, err := proto.Marshal(nil)
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
		fmt.Printf("send revertState msg to %s\n", args[0])
		return nil
	} else {
		return errors.New("response error")
	}
}
