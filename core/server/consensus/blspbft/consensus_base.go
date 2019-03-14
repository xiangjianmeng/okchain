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

package blspbft

import (
	"time"

	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

var logger = logging.MustGetLogger("consensusBase")

func init() {
	ps.ConsensusBackupFactory = newConsensusBackup
	ps.ConsensusLeadFactory = newConsensusLead
}

type ConsensusBase struct {
	role       ps.IRole
	peerServer *ps.PeerServer
	leader     *pb.PeerEndpoint
	state      ConsensusState_Type
}

func newConsensusBase(peer *ps.PeerServer) *ConsensusBase {
	c := &ConsensusBase{}
	c.peerServer = peer
	c.leader = &pb.PeerEndpoint{}
	return c
}

func (c *ConsensusBase) SetIRole(irole ps.IRole) {
	c.role = irole
}

func (c *ConsensusBase) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (c *ConsensusBase) GetCurrentLeader() *pb.PeerEndpoint {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (c *ConsensusBase) GetCurrentConsensusStage() pb.ConsensusType {
	return consensusHandler.currentType
}

func (c *ConsensusBase) SetCurrentConsensusStage(consensusType pb.ConsensusType) {
	consensusHandler.currentType = consensusType
}

func (c *ConsensusBase) WaitForViewChange() {
	ticker := time.NewTicker(time.Duration(c.peerServer.GetViewchangeTimeOut()) * time.Second)
	defer ticker.Stop()
	select {
	// waiting for channel signal when consensus finished
	case <-c.peerServer.VcChan:
		logger.Debugf("consensus finished before viewchange timeout")
		return
	// waiting for viewchange timeout
	case t := <-ticker.C:
		logger.Debugf("viewchange timeout when consensus handling, %+v, timeout set is %ds",
			t.String(), c.peerServer.GetViewchangeTimeOut())
		consensusHandler.StartViewChange(c.role)
		return
	}
}
