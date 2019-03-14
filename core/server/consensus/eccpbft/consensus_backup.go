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
* BFT 共识backup 节点实现
 */

package eccpbft

import (
	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

var loggerBackup = logging.MustGetLogger("consensusBackup")

func init() {
}

type ConsensusBackup struct {
	*ConsensusBase
}

func newConsensusBackup(peer *ps.PeerServer) ps.IConsensusBackup {
	base := newConsensusBase(peer)
	c := &ConsensusBackup{ConsensusBase: base}
	return c
}

func (c *ConsensusBackup) UpdateLeader(leader *pb.PeerEndpoint) {
	c.leader = leader
	logger.Infof("ConsensusBase:UpdateLeader: %s", c.leader)
}

func (cl *ConsensusBackup) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint, r ps.IRole) error {
	cl.role = r
	switch msg.Type {
	case pb.Message_Consensus_Announce:
		cl.processMessageAnnounce(msg, from)
	case pb.Message_Consensus_Challenge:
		cl.processMessageChallenge(msg, from)
	case pb.Message_Consensus_CollectiveSig:
		cl.processMessageCollectiveSig(msg, from)
	case pb.Message_Consensus_FinalChallenge:
		cl.processMessageFinalChallenge(msg, from)
	case pb.Message_Consensus_FinalCollectiveSig:
		cl.processMessageFinalCollectiveSig(msg, from)
	default:
		util.OkChainPanic("Invalid message")
	}
	return nil
}

func (cb *ConsensusBackup) send2Lead(msgType pb.Message_Type) error {
	res := &pb.Message{}
	res.Type = msgType
	res.Peer = cb.peerServer.SelfNode
	loggerBackup.Infof("send message to: %s", cb.leader)

	go func() {
		err := cb.peerServer.Multicast(res, pb.PeerEndpointList{cb.leader})
		if err != nil {
			loggerBackup.Errorf("send message to all DS node failed")
		}
	}()
	return nil
}

func (cb *ConsensusBackup) processMessageAnnounce(msg *pb.Message, from *pb.PeerEndpoint) error {
	cb.send2Lead(pb.Message_Consensus_Commit)
	return nil
}

func (cb *ConsensusBackup) processMessageChallenge(msg *pb.Message, from *pb.PeerEndpoint) error {
	cb.send2Lead(pb.Message_Consensus_Response)
	return nil
}

func (cb *ConsensusBackup) processMessageCollectiveSig(msg *pb.Message, from *pb.PeerEndpoint) error {
	cb.send2Lead(pb.Message_Consensus_FinalCommit)
	return nil
}

func (cb *ConsensusBackup) processMessageFinalChallenge(msg *pb.Message, from *pb.PeerEndpoint) error {
	cb.send2Lead(pb.Message_Consensus_FinalResponse)
	return nil
}

func (cb *ConsensusBackup) processMessageFinalCollectiveSig(msg *pb.Message, from *pb.PeerEndpoint) error {
	// notify sharding backup or ds backup
	err := cb.role.OnConsensusCompleted(nil)
	if err != nil {
		return err
	}
	return nil
}
