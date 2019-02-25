// Copyright The go-okchain Authors 2018,  All rights reserved.

/*
Copyright 2018 The Okchain Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
* DESCRIPTION

 */

package state

import (
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/core/server/role"
	pb "github.com/ok-chain/okchain/protos"
)

//////////////////////////////////////////////////////////////////////////////////////////////////////
// sharding state
//////////////////////////////////////////////////////////////////////////////////////////////////////
type STATE_ShardingIdle struct {
	StateBase
}
type STATE_POW_SUBMISSION struct {
	StateBase
}
type STATE_WAITING_DSBLOCK struct {
	StateBase
}
type STATE_MICROBLOCK_CONSENSUS_PREP struct {
	StateBase
}
type STATE_MICROBLOCK_CONSENSUS struct {
	StateBase
}
type STATE_WAITING_FINALBLOCK struct {
	StateBase
}
type STATE_FALLBACK_CONSENSUS_PREP struct {
	StateBase
}
type STATE_FALLBACK_CONSENSUS struct {
	StateBase
}
type STATE_WAITING_FALLBACKBLOCK struct {
	StateBase
}

func (state *STATE_WAITING_DSBLOCK) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Debugf("StateWAITING_DSBLOCK ProcessMsg")
	if msg.Type == pb.Message_Node_ProcessDSBlock {
		r.ProcessDSBlock(msg, from)
	}
	return nil
}

func (state *STATE_WAITING_FINALBLOCK) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Debugf("StateWAITING_FINALBLOCK ProcessMsg")
	if msg.Type == pb.Message_Node_ProcessFinalBlock {
		r.ProcessFinalBlock(msg, from)
	}
	return nil
}

func (state *STATE_MICROBLOCK_CONSENSUS) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	if msg.Type == pb.Message_Node_ProcessFinalBlock {
		r.ProcessFinalBlock(msg, from)
		return nil
	}

	if msg.Type == pb.Message_Node_ProcessDSBlock {
		r.ProcessDSBlock(msg, from)
		return nil
	}

	return state.processConsensusMsg(r, msg, from)
}

func (state *STATE_ShardingIdle) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Debugf("StateShardingIdle ProcessMsg: %s", msg.Type)

	if msg.Type == pb.Message_Node_RevertState {
		role.RevertRole(r.GetPeerServer())
	}
	return nil
}
