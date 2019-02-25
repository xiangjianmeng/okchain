// Copyright The go-okchain Authors 2018,  All rights reserved.

package state

import (
	ps "github.com/ok-chain/okchain/core/server"
	pb "github.com/ok-chain/okchain/protos"
)

type STATE_Idle struct {
	StateBase
}

func (state *STATE_Idle) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	switch msg.Type {
	case pb.Message_Node_ProcessFinalBlock:
		return r.ProcessFinalBlock(msg, from)
	case pb.Message_Node_ProcessDSBlock:
		return r.ProcessDSBlock(msg, from)
	case pb.Message_DS_SetPrimary:
		return r.ProcessSetPrimary(msg, from)
	case pb.Message_Node_ProcessStartPoW:
		return r.ProcessStartPoW(msg, from)
	case pb.Message_Node_NewAdd:
		r.ProcessNewAdd()
		return nil
	}

	return nil
}
