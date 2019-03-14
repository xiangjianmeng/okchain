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

package state

import (
	"reflect"

	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

var logger = logging.MustGetLogger("state")

func init() {
	ps.StateContainerFactory = newStateContainer
}

type StateBase struct {
	imp ps.IState
}

func (state *StateBase) GetName() string {
	return reflect.TypeOf(state.imp).String()
}

func (state *StateBase) processConsensusMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	if msg.Type == pb.Message_Consensus_Announce ||
		msg.Type == pb.Message_Consensus_Commit ||
		msg.Type == pb.Message_Consensus_Challenge ||
		msg.Type == pb.Message_Consensus_Response ||
		msg.Type == pb.Message_Consensus_CollectiveSig ||
		msg.Type == pb.Message_Consensus_FinalCommit ||
		msg.Type == pb.Message_Consensus_FinalChallenge ||
		msg.Type == pb.Message_Consensus_FinalResponse ||
		msg.Type == pb.Message_Consensus_FinalCollectiveSig ||
		msg.Type == pb.Message_Consensus_BroadCastBlock {

		logger.Debugf("[%s]: ProcessMsg: %s, CurRole<%s>, CurState<%s>", util.GId,
			msg.Type.String(), reflect.TypeOf(r), reflect.TypeOf(state.imp))
		return r.ProcessConsensusMsg(msg, from)
	}

	if msg.Type == pb.Message_Node_ProcessDSBlock ||
		msg.Type == pb.Message_Node_ProcessFinalBlock {
		logger.Debugf("[%s]: ProcessMsg: %s, CurRole<%s>, CurState<%s>", util.GId,
			msg.Type.String(), reflect.TypeOf(r), reflect.TypeOf(state.imp))
		return r.ProcessMsg(msg, from)
	}
	return nil
}

func (state *StateBase) ProcessMsg(r ps.IRole, msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Errorf("CurRole<%s>, CurState<%s>",
		reflect.TypeOf(r), reflect.TypeOf(state.imp))

	util.OkChainPanic("Invalid invoke")
	return nil
}

func (state *StateBase) AwaitDone() {
	util.OkChainPanic("Invalid invoke")
}
