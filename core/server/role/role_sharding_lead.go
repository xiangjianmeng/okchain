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

package role

import (
	"time"

	"github.com/golang/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/core/txpool"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

type RoleShardingLead struct {
	*RoleShardingBase
	consensusLeader ps.IConsensusLead
	txPool          *txpool.TxPool
}

var loggerShardingLead = logging.MustGetLogger("shardingLeadRole")

func newRoleShardingLead(peer *ps.PeerServer) ps.IRole {
	base := newRoleShardingBase(peer)
	r := &RoleShardingLead{RoleShardingBase: base}
	r.initBase(r)
	r.name = "ShardLead"
	r.txPool = txpool.GetTxPool()
	r.consensusLeader = peer.ProduceConsensusLead(peer)
	r.consensusLeader.SetIRole(r)
	return r
}

func (r *RoleShardingLead) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	err := r.consensusLeader.ProcessConsensusMsg(msg, from, r)
	if err != nil {
		loggerShardingLead.Errorf("handle consensus message failed with error: %s", err.Error())
		return err
	}
	return nil
}

func (r *RoleShardingLead) composeMicroblock() (*pb.MicroBlock, error) {

	mb := &pb.MicroBlock{}
	mb.Header = &pb.TxBlockHeader{}
	mb.Header.Timestamp = pb.CreateUtcTimestamp()
	mb.Header.Version = uint32(1)

	mb.Header.BlockNumber = r.peerServer.TxBlockChain().CurrentBlock().NumberU64() + 1
	if r.currentMicroBlock == nil {
		mb.Header.PreviousBlockHash = []byte("000000000000000000000000000000")
	} else {
		mb.Header.PreviousBlockHash = util.Hash(r.currentMicroBlock).Bytes()
	}

	mb.Header.DSBlockHash = r.peerServer.DsBlockChain().CurrentBlock().Hash().Bytes()

	mb.Header.DSBlockNum = r.peerServer.DsBlockChain().CurrentBlock().NumberU64()
	mb.Header.GasLimit = uint64(0)
	mb.Header.GasUsed = uint64(0)
	mb.ShardID = r.peerServer.ShardingID
	mb.Miner = r.peerServer.SelfNode

	mb.Header.StateRoot = []byte("state delta hash")

	pending := r.peerServer.GetTxPool().PendingTxList()
	//ord, cnt, gasAll := r.peerServer.GetTxPool().PendingNTxList(MICROBLOCK_MAXCOMPOSETX_NUMBER)
	//pending := append(ord, cnt...)
	//mb.Header.GasLimit = gasAll
	//loggerShardingLead.Debugf("pendingNtxlist: ord.len=%d, cnt.len=%d, gasAll=%d", len(ord), len(cnt), gasAll)

	// mb.Transactions = r.filter(pending)
	mb.Transactions = r.peerServer.FilterTxs(pending)
	mb.Header.TxNum = uint64(len(mb.Transactions))
	for _, t := range mb.Transactions {
		mb.Header.GasUsed += t.GasPrice
	}

	mb.Header.TxRoot = pb.ComputeMerkelRoot(mb.Transactions)
	loggerShardingLead.Debugf("txRoot: %+v", mb.Header.TxRoot)

	mb.ShardingLeadCoinBase = r.peerServer.CoinBase

	return mb, nil
}

func (r *RoleShardingLead) onMircoBlockConsensusCompleted(err error) error {
	msg := &pb.Message{Type: pb.Message_DS_MicroblockSubmission}
	msg.Timestamp = pb.CreateUtcTimestamp()
	msg.Peer = r.peerServer.SelfNode
	data, err := proto.Marshal(r.GetCurrentMicroBlock())
	if err != nil {
		loggerShardingLead.Errorf("micro block marshal failed with error: %s", err.Error())
		return ErrMarshalMessage
	}

	msg.Payload = data
	sig, err := r.peerServer.MsgSinger.SignHash(r.GetCurrentMicroBlock().Hash().Bytes(), nil)
	if err != nil {
		loggerShardingLead.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}
	msg.Signature = sig
	loggerShardingLead.Debugf("current micro block is %+v", r.GetCurrentMicroBlock())

	go func() {
		err := r.peerServer.Multicast(msg, r.peerServer.Committee)
		if err != nil {
			loggerShardingLead.Errorf("send to all ds nodes failed")
		}
	}()

	r.ChangeState(ps.STATE_WAITING_FINALBLOCK)
	loggerShardingLead.Debugf("waiting for final block")
	r.DumpCurrentState()
	return nil
}

func (r *RoleShardingLead) onViewChangeConsensusCompleted(err error) error {
	switch r.GetCurrentVCBlock().Header.Stage {
	case "MicroBlockConsensus":
		r.consensusLeader.SetCurrentConsensusStage(pb.ConsensusType_MicroBlockConsensus)
		r.ChangeState(ps.STATE_MICROBLOCK_CONSENSUS)
		r.DumpCurrentState()
		err = r.OnMircoBlockConsensusStarted(r.peerServer.SelfNode)
	default:
		loggerDsLead.Errorf("invalid state %s", r.GetCurrentVCBlock().Header.Stage)
		return ErrHandleInCurrentState
	}

	if err != nil {
		return err
	}
	return nil
}

func (r *RoleShardingLead) getConsensusData(consensusType pb.ConsensusType) (proto.Message, error) {
	switch consensusType {
	case pb.ConsensusType_MicroBlockConsensus:
		mblock, err := r.composeMicroblock()
		if err != nil {
			loggerShardingLead.Errorf("compose micro block error: %s", err.Error())
			return nil, ErrComposeBlock
		}
		loggerShardingLead.Debugf("compose micro block is %+v", mblock)
		r.SetCurrentMicroBlock(mblock)
		return r.GetCurrentMicroBlock(), nil
	case pb.ConsensusType_ViewChangeConsensus:
		vcblock, err := r.composeVCBlock()
		if err != nil {
			loggerDsLead.Errorf("compose vcblock error: %s", err.Error())
			return nil, ErrComposeBlock
		}
		r.SetCurrentVCBlock(vcblock)
		return r.GetCurrentVCBlock(), nil
	default:
		return nil, ErrHandleInCurrentState
	}
}

func (r *RoleShardingLead) OnMircoBlockConsensusStarted(peer *pb.PeerEndpoint) error {
	go r.consensusLeader.WaitForViewChange()

	loggerShardingLead.Debugf("wait %ds to start consensus", CONSENSUS_START_WAITTIME)
	time.Sleep(time.Duration(CONSENSUS_START_WAITTIME) * time.Second) //OnMircoBlockConsensusStarted
	r.consensusLeader.SetCurrentConsensusStage(pb.ConsensusType_MicroBlockConsensus)

	announce, err := r.produceConsensusMessage(pb.ConsensusType_MicroBlockConsensus, pb.Message_Consensus_Announce)
	if err != nil {
		loggerShardingLead.Errorf("compose micro block consensus announce message failed with error: %s", err.Error())
		return err
	}

	loggerShardingLead.Debugf("announce message is %+v", announce)
	loggerShardingLead.Debugf("announce send to peers are %+v", r.peerServer.ShardingNodes)
	r.consensusLeader.InitiateConsensus(announce, r.peerServer.ShardingNodes, r.imp)
	return nil
}

func (r *RoleShardingLead) OnViewChangeConsensusStarted() error {
	go r.consensusLeader.WaitForViewChange()

	loggerShardingLead.Debugf("wait %ds to start consensus", CONSENSUS_START_WAITTIME)
	time.Sleep(time.Duration(CONSENSUS_START_WAITTIME) * time.Second) //OnViewChangeConsensusStarted
	r.consensusLeader.SetCurrentConsensusStage(pb.ConsensusType_ViewChangeConsensus)

	announce, err := r.produceConsensusMessage(pb.ConsensusType_ViewChangeConsensus, pb.Message_Consensus_Announce)
	if err != nil {
		loggerShardingLead.Errorf("compose viewchange consensus announce message failed with error: %s", err.Error())
		return err
	}

	loggerShardingLead.Debugf("announce message is %+v", announce)
	r.consensusLeader.InitiateConsensus(announce, r.peerServer.ShardingNodes, r.imp)
	return nil
}

func (r *RoleShardingLead) produceAnnounce(block proto.Message, envelope *pb.Message, payload *pb.ConsensusPayload, consensusType pb.ConsensusType) (*pb.Message, error) {
	var err error
	switch consensusType {
	case pb.ConsensusType_MicroBlockConsensus:
		err = r.preConsensusProcessMicroBlock(block, envelope, payload)
	case pb.ConsensusType_ViewChangeConsensus:
		err = r.preConsensusProcessVCBlock(block, envelope, payload)
	default:
		return nil, ErrHandleInCurrentState
	}
	if err != nil {
		return nil, err
	}

	return envelope, nil
}

func (r *RoleShardingLead) preConsensusProcessMicroBlock(block proto.Message, announce *pb.Message, _ *pb.ConsensusPayload) error {
	var microBlock *pb.MicroBlock
	var ok bool
	if microBlock, ok = block.(*pb.MicroBlock); ok {
		sig, err := r.peerServer.MsgSinger.SignHash(microBlock.Hash().Bytes(), nil)
		if err != nil {
			loggerShardingLead.Errorf("bls message sign failed with error: %s", err.Error())
			return ErrSignMessage
		}
		announce.Signature = sig
	}

	data, err := r.produceConsensusPayload(microBlock, pb.ConsensusType_MicroBlockConsensus)
	if err != nil {
		return err
	}
	announce.Payload = data
	return nil
}

func (r *RoleShardingLead) preConsensusProcessVCBlock(block proto.Message, announce *pb.Message, _ *pb.ConsensusPayload) error {
	var vcblock *pb.VCBlock
	var ok bool
	if vcblock, ok = block.(*pb.VCBlock); ok {
		sig, err := r.peerServer.MsgSinger.SignHash(vcblock.Hash().Bytes(), nil)
		if err != nil {
			loggerShardingLead.Errorf("bls message sign failed with error: %s", err.Error())
			return ErrSignMessage
		}
		announce.Signature = sig
	}

	data, err := r.produceConsensusPayload(vcblock, pb.ConsensusType_ViewChangeConsensus)
	if err != nil {
		return err
	}
	announce.Payload = data
	return nil
}
