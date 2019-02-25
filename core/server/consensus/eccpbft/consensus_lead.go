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
* BFT 共识 lead 节点实现
 */

package eccpbft

import (
	"math"
	"sync"
	"time"

	logging "github.com/ok-chain/okchain/log"
	ps "github.com/ok-chain/okchain/core/server"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

var loggerLead = logging.MustGetLogger("consensusLead")

type ConsensusLead struct {
	*ConsensusBase
	counter              int
	toleranceSize        int
	mux                  sync.Mutex
	consensusBackupNodes pb.PeerEndpointList
}

func newConsensusLead(peer *ps.PeerServer) ps.IConsensusLead {
	base := newConsensusBase(peer)
	c := &ConsensusLead{ConsensusBase: base}
	c.counter = 0
	return c
}

func (cl *ConsensusLead) GetToleranceSize() int {
	return cl.toleranceSize
}

// ProcessConsensusMsg process consensus message
func (cl *ConsensusLead) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint, r ps.IRole) error {
	cl.role = r

	switch msg.Type {
	case pb.Message_Consensus_Commit:
		cl.processMessageCommit(msg, from)
	case pb.Message_Consensus_Response:
		cl.processMessageResponse(msg, from)
	case pb.Message_Consensus_FinalCommit:
		cl.processMessageFinalCommit(msg, from)
	case pb.Message_Consensus_FinalResponse:
		cl.processMessageFinalResponse(msg, from)
	default:
		util.OkChainPanic("Invalid message")
	}
	return nil
}

// InitiateConsensus
func (cl *ConsensusLead) InitiateConsensus(msg *pb.Message, to []*pb.PeerEndpoint, r ps.IRole) error {
	time.Sleep(time.Duration(CONSENSUS_START_WAITTIME) * time.Second)

	cl.state = ANNOUNCE_DONE

	cl.consensusBackupNodes = to
	cl.toleranceSize = int(math.Floor(float64(cl.consensusBackupNodes.Length())*ToleranceFraction)) + 1
	loggerLead.Infof("toleranceSize: %d", cl.toleranceSize)

	cl.sendMsg2Backups(msg)
	return nil
}

func (cl *ConsensusLead) sendMsg2Backups(msg *pb.Message) error {
	err := cl.peerServer.Multicast(msg, cl.consensusBackupNodes)
	if err != nil {
		loggerLead.Errorf("send message to all DS node failed")
	}

	return nil
}

func (cl *ConsensusLead) send2Backups(msgType pb.Message_Type) error {
	res := &pb.Message{}
	res.Type = msgType
	res.Peer = cl.peerServer.SelfNode
	err := cl.peerServer.Multicast(res, cl.consensusBackupNodes)
	if err != nil {
		loggerLead.Errorf("send message to all DS node failed")
	}

	return nil
}

func (cl *ConsensusLead) processMessageCommit(msg *pb.Message, from *pb.PeerEndpoint) error {
	cl.verify(ANNOUNCE_DONE, CHALLENGE_DONE, pb.Message_Consensus_Challenge)
	return nil
}

func (cl *ConsensusLead) verify(curState, netState ConsensusState_Type, msgType pb.Message_Type) bool {
	send2Backups := false

	loggerLead.Infof("%s: verify<%s>", util.GId, msgType.String())
	defer loggerLead.Infof("%s: verify<%s> done", util.GId, msgType.String())
	//time.Sleep(time.Second * 2)

	{
		loggerLead.Infof("%s: update counter<%s>", util.GId, msgType.String())
		defer loggerLead.Infof("%s: update counter<%s> done", util.GId, msgType.String())
		cl.mux.Lock()
		defer cl.mux.Unlock()
		if cl.state == curState {
			cl.counter++
			if cl.counter == cl.toleranceSize {
				cl.counter = 0
				cl.state = netState
				send2Backups = true
			}
		}
	}

	if send2Backups {
		cl.send2Backups(msgType)
	}

	return send2Backups
}

func (cl *ConsensusLead) processMessageResponse(msg *pb.Message, from *pb.PeerEndpoint) error {
	cl.verify(CHALLENGE_DONE, COLLECTIVESIG_DONE, pb.Message_Consensus_CollectiveSig)
	return nil
}

func (cl *ConsensusLead) processMessageFinalCommit(msg *pb.Message, from *pb.PeerEndpoint) error {
	cl.verify(COLLECTIVESIG_DONE, FINALCHALLENGE_DONE, pb.Message_Consensus_FinalChallenge)

	return nil
}

func (cl *ConsensusLead) processMessageFinalResponse(msg *pb.Message, from *pb.PeerEndpoint) error {
	res := cl.verify(FINALCHALLENGE_DONE, FINALCOLLECTIVESIG_DONE, pb.Message_Consensus_FinalCollectiveSig)

	if res {
		cl.counter = 0
		// notify sharding lead or ds lead
		err := cl.role.OnConsensusCompleted(nil)
		if err != nil {
			return err
		}
	}
	return nil
}
