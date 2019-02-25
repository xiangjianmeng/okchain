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
	"github.com/ok-chain/okchain/util"
)

type STATE_DsIdle struct {
	StateBase
}
type STATE_Wait4POW_SUBMISSION struct {
	StateBase
}
type STATE_DSBLOCK_CONSENSUS_PREP struct {
	StateBase
}
type STATE_DSBLOCK_CONSENSUS struct {
	StateBase
}
type STATE_WAIT4_MICROBLOCK_SUBMISSION struct {
	StateBase
}
type STATE_FINALBLOCK_CONSENSUS_PREP struct {
	StateBase
}
type STATE_FINALBLOCK_CONSENSUS struct {
	StateBase
}
type STATE_VIEWCHANGE_CONSENSUS_PREP struct {
	StateBase
}
type STATE_VIEWCHANGE_CONSENSUS struct {
	StateBase
}

func (state *STATE_DSBLOCK_CONSENSUS) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	return state.processConsensusMsg(r, msg, from)
}

func (state *STATE_FINALBLOCK_CONSENSUS) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	return state.processConsensusMsg(r, msg, from)
}

func (state *STATE_VIEWCHANGE_CONSENSUS) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	if msg.Type == pb.Message_Node_ProcessDSBlock {
		r.ProcessDSBlock(msg, from)
		return nil
	}

	if msg.Type == pb.Message_Node_ProcessFinalBlock {
		r.ProcessFinalBlock(msg, from)
		return nil
	}

	return state.processConsensusMsg(r, msg, from)
}

func (state *STATE_DsIdle) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	_ = msg.Type.String()

	if msg.Type == pb.Message_Node_RevertState {
		role.RevertRole(r.GetPeerServer())
	}
	return nil
}

func (state *STATE_Wait4POW_SUBMISSION) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Debugf("[%s]: enter StateWait4POW_SUBMISSION ProcessMsg", util.GId)
	defer logger.Debugf("[%s]: exit StateWait4POW_SUBMISSION ProcessMsg", util.GId)

	if msg.Type == pb.Message_DS_PoWSubmission {
		r.ProcessPoWSubmission(msg, from)
	}
	return nil
}

func (state *STATE_WAIT4_MICROBLOCK_SUBMISSION) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Debugf("[%s]: enter StateWait4MicroBlock_SUBMISSION ProcessMsg", util.GId)
	defer logger.Debugf("[%s]: exit StateWait4MicroBlock_SUBMISSION ProcessMsg", util.GId)

	if msg.Type == pb.Message_DS_MicroblockSubmission {
		err := r.ProcessMicroBlockSubmission(msg, from)
		if err != nil {
			return err
		}
	}
	return nil
}
