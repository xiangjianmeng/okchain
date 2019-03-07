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
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/p2p/gossip/common"
	"github.com/ok-chain/okchain/p2p/gossip/filter"
	"github.com/ok-chain/okchain/p2p/gossip/gossip"
	pb "github.com/ok-chain/okchain/protos"
)

var loggerDsLead = logging.MustGetLogger("dsLeadRole")

type RoleDsLead struct {
	*RoleDsBase
	dsConsensusLeader ps.IConsensusLead
}

func newRoleDsLead(peer *ps.PeerServer) ps.IRole {
	dsBase := newRoleDsBase(peer)
	r := &RoleDsLead{RoleDsBase: dsBase}
	r.initBase(r)

	r.name = "DsLead"

	r.dsConsensusLeader = peer.ProduceConsensusLead(peer)
	r.dsConsensusLeader.SetIRole(r)
	return r
}

func (r *RoleDsLead) onDsBlockConsensusCompleted(err error) error {
	// todo: handle err

	// 1. persist ds block and update network topology
	dsblock := r.GetCurrentDSBlock()
	r.peerServer.ConsensusData.PoWSubList.Clear()

	//oldShardingNodes := r.peerServer.ShardingNodes
	err = r.onDsBlockReady(dsblock)
	if err != nil {
		return err
	}

	// 2. go to backup role and next state
	r.peerServer.ChangeRole(ps.PeerRole_DsBackup, ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
	loggerDsLead.Infof("I am next round DS Backup")

	// 3. send ds block to all sharding nodes
	msg := &pb.Message{Type: pb.Message_Node_ProcessDSBlock}
	msg.Timestamp = pb.CreateUtcTimestamp()

	sig, err := r.peerServer.MsgSinger.SignHash(dsblock.Hash().Bytes(), nil)
	if err != nil {
		loggerDsLead.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}
	msg.Signature = sig

	data, err := proto.Marshal(dsblock)
	if err != nil {
		loggerDsLead.Errorf("dsblock marshal failed with error: %s", err.Error())
		return ErrMarshalMessage
	}

	msg.Payload = data
	msg.Peer = r.peerServer.SelfNode

	//loggerDsLead.Debugf("oldShardingNodes Dump")
	//oldShardingNodes.Dump()
	//err = r.peerServer.Multicast(msg, oldShardingNodes)
	//if err != nil {
	//	loggerDsLead.Errorf("send to dsblock to all sharding nodes failed")
	//	return
	//}

	msgData, err := proto.Marshal(msg)
	if err != nil {
		loggerDsLead.Errorf(err.Error())
		return ErrMarshalMessage
	}

	//r.peerServer.Gossip.Gossip(gossip.CreateDataMsg(r.peerServer.DsBlockChain().CurrentBlock().NumberU64()+r.peerServer.TxBlockChain().CurrentBlock().NumberU64(),
	//	msgData, common.ChainID("A")))
	allPeers := filter.SelectAllPeers(r.peerServer.Gossip.Peers())
	r.peerServer.Gossip.SendPri(gossip.CreateDataMsg(r.peerServer.DsBlockChain().CurrentBlock().NumberU64()+r.peerServer.TxBlockChain().CurrentBlock().NumberU64(), msgData, common.ChainID("A")), allPeers...)
	return nil
}

func (r *RoleDsLead) onFinalBlockConsensusCompleted(err error) error {
	//todo: handle err
	err = r.onFinalBlockReady(r.GetCurrentFinalBlock())
	if err != nil {
		return err
	}
	r.peerServer.ConsensusData.MicroBlockList.Clear()

	numOfSharding := len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes) / r.peerServer.GetShardingSize()

	if (numOfSharding != 0 && r.GetCurrentFinalBlock().Header.BlockNumber%uint64(r.peerServer.GetShardingSize()) == 0) ||
		(numOfSharding == 0 && r.GetCurrentFinalBlock().Header.BlockNumber%uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)) == 0) {
		r.ChangeState(ps.STATE_WAIT4_POW_SUBMISSION)
		ctx, cancle := context.WithTimeout(context.Background(), time.Duration(r.peerServer.GetWait4PoWTime())*time.Second)
		loggerDsLead.Debugf("waiting for POW_SUBMISSION")
		go r.Wait4PoWSubmission(ctx, cancle)
	} else {
		loggerDsLead.Debugf("waiting for MICROBLOCK_SUBMISSION")
		r.ChangeState(ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
	}

	msg := &pb.Message{Type: pb.Message_Node_ProcessFinalBlock}
	msg.Timestamp = pb.CreateUtcTimestamp()

	data, err := proto.Marshal(r.GetCurrentFinalBlock())
	if err != nil {
		loggerDsLead.Errorf("final block marshal failed with error: %s", err.Error())
		return ErrMarshalMessage
	}

	msg.Payload = data
	msg.Peer = r.peerServer.SelfNode
	sig, err := r.peerServer.MsgSinger.SignHash(r.GetCurrentFinalBlock().Hash().Bytes(), nil)
	if err != nil {
		loggerDsLead.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}

	msg.Signature = sig

	msgData, err := proto.Marshal(msg)
	if err != nil {
		loggerDsLead.Errorf(err.Error())
		return ErrMarshalMessage
	}

	//r.peerServer.Gossip.Gossip(gossip.CreateDataMsg(r.peerServer.DsBlockChain().CurrentBlock().NumberU64()+r.peerServer.TxBlockChain().CurrentBlock().NumberU64(),
	//	msgData, common.ChainID("A")))
	allPeers := filter.SelectAllPeers(r.peerServer.Gossip.Peers())
	r.peerServer.Gossip.SendPri(gossip.CreateDataMsg(r.peerServer.DsBlockChain().CurrentBlock().NumberU64()+r.peerServer.TxBlockChain().CurrentBlock().NumberU64(),
		msgData, common.ChainID("A")), allPeers...)

	//err = r.peerServer.Multicast(msg, r.peerServer.ShardingNodes)
	//if err != nil {
	//	loggerDsLead.Errorf("send Message_Node_ProcessFinalBlock to all sharding nodes failed")
	//	return
	//}

	if (numOfSharding != 0 && r.GetCurrentFinalBlock().Header.BlockNumber%uint64(r.peerServer.GetShardingSize()) == 0) ||
		(numOfSharding == 0 && r.GetCurrentFinalBlock().Header.BlockNumber%uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)) == 0) {
		r.peerServer.ShardingNodes = []*pb.PeerEndpoint{}
	}

	r.DumpCurrentState()
	return nil
}

func (r *RoleDsLead) onViewChangeConsensusCompleted(err error) error {
	switch r.GetCurrentVCBlock().Header.Stage {
	case "DsBlockConsensus":
		r.dsConsensusLeader.SetCurrentConsensusStage(pb.ConsensusType_DsBlockConsensus)
		r.ChangeState(ps.STATE_DSBLOCK_CONSENSUS)
		r.DumpCurrentState()
		r.onDsBlockConsensusStart()
	case "FinalBlockConsensus":
		r.dsConsensusLeader.SetCurrentConsensusStage(pb.ConsensusType_FinalBlockConsensus)
		r.ChangeState(ps.STATE_FINALBLOCK_CONSENSUS)
		r.DumpCurrentState()
		r.onFinalBlockConsensusStart()
	default:
		loggerDsLead.Errorf("invalid state %s", r.GetCurrentVCBlock().Header.Stage)
		return ErrHandleInCurrentState
	}
	return nil
}

func (r *RoleDsLead) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	err := r.dsConsensusLeader.ProcessConsensusMsg(msg, from, r)
	if err != nil {
		loggerDsLead.Errorf("handle consensus message failed with error: %s", err.Error())
		return err
	}
	return nil
}

func (r *RoleDsLead) onWait4PoWSubmissionDone() error {
	// todo: ensure all backups go to state STATE_DSBLOCK_CONSENSUS before InitiateConsensus
	// http://gitlab.okcoin-inc.com/okchain/go-okchain/issues/24#3-consensus_start_waittime
	if r.peerServer.ConsensusData.PoWSubList.Len() == 0 {
		ctx, cancle := context.WithTimeout(context.Background(), time.Duration(r.peerServer.GetWait4PoWTime())*time.Second)
		go r.Wait4PoWSubmission(ctx, cancle)
		return nil
	}

	r.ChangeState(ps.STATE_DSBLOCK_CONSENSUS)
	loggerDsLead.Debugf("wait for POW_SUBMISSION finished, begin to start dsblock consensus")

	r.onDsBlockConsensusStart()
	return nil
}

func (r *RoleDsLead) onWait4MicroBlockSubmissionDone() error {
	r.ChangeState(ps.STATE_FINALBLOCK_CONSENSUS)
	loggerDsLead.Debugf("wait for MICROBLOCK_SUBMISSION finished, begin to start final block consensus")
	r.onFinalBlockConsensusStart()
	return nil
}

func (r *RoleDsLead) onFinalBlockConsensusStart() {
	go r.dsConsensusLeader.WaitForViewChange()

	loggerShardingLead.Debugf("wait %ds to start consensus", CONSENSUS_START_WAITTIME)
	time.Sleep(time.Duration(CONSENSUS_START_WAITTIME) * time.Second) //onFinalBlockConsensusStart

	announce, err := r.produceConsensusMessage(pb.ConsensusType_FinalBlockConsensus, pb.Message_Consensus_Announce)
	if err != nil {
		loggerDsLead.Errorf("compose final block consensus announce message failed with error: %s", err.Error())
		return
	}

	loggerDsLead.Debugf("announce message is %+v", announce)
	r.dsConsensusLeader.SetCurrentConsensusStage(pb.ConsensusType_FinalBlockConsensus)
	r.dsConsensusLeader.InitiateConsensus(announce, r.peerServer.Committee, r.imp)
}

func (r *RoleDsLead) OnViewChangeConsensusStarted() error {
	go r.dsConsensusLeader.WaitForViewChange()

	loggerShardingLead.Debugf("wait %ds to start consensus", CONSENSUS_START_WAITTIME)
	time.Sleep(time.Duration(CONSENSUS_START_WAITTIME) * time.Second) //OnViewChangeConsensusStarted

	r.peerServer.Committee = append(r.peerServer.Committee[1:len(r.peerServer.Committee)], r.peerServer.Committee[0])
	loggerDsLead.Debugf("current committee is %+v", r.peerServer.Committee)
	r.dsConsensusLeader.SetCurrentConsensusStage(pb.ConsensusType_ViewChangeConsensus)

	announce, err := r.produceConsensusMessage(pb.ConsensusType_ViewChangeConsensus, pb.Message_Consensus_Announce)
	if err != nil {
		loggerDsLead.Errorf("compose viewchange consensus announce message failed with error: %s", err.Error())
		return err
	}

	loggerDsLead.Debugf("announce message is %+v", announce)
	r.dsConsensusLeader.InitiateConsensus(announce, r.peerServer.Committee, r.imp)
	return nil
}

func (r *RoleDsLead) getConsensusData(consensusType pb.ConsensusType) (proto.Message, error) {
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		dSBlock, err := r.composeDSBlock()
		if err != nil {
			loggerDsLead.Errorf("compose dsblock error: %s", err.Error())
			return nil, ErrComposeBlock
		}
		loggerDsLead.Debugf("compose dsblock is %+v", dSBlock)
		r.SetCurrentDSBlock(dSBlock)
		return r.GetCurrentDSBlock(), nil
	case pb.ConsensusType_FinalBlockConsensus:
		finalBlock, err := r.composeFinalBlock()
		if err != nil {
			loggerDsLead.Errorf("compose final block error: %s", err.Error())
			return nil, ErrComposeBlock
		}
		loggerDsLead.Debugf("compose final block is %+v", finalBlock)
		r.SetCurrentFinalBlock(finalBlock)
		return r.GetCurrentFinalBlock(), nil
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

func (r *RoleDsLead) onDsBlockConsensusStart() {
	go r.dsConsensusLeader.WaitForViewChange()

	loggerShardingLead.Debugf("wait %ds to start consensus", CONSENSUS_START_WAITTIME)
	time.Sleep(time.Duration(CONSENSUS_START_WAITTIME) * time.Second) //onDsBlockConsensusStart

	announce, err := r.produceConsensusMessage(pb.ConsensusType_DsBlockConsensus, pb.Message_Consensus_Announce)
	if err != nil {
		loggerDsLead.Errorf("compose dsblock consensus announce message failed with error: %s", err.Error())
		return
	}

	loggerDsLead.Debugf("announce message is %+v", announce)
	r.dsConsensusLeader.SetCurrentConsensusStage(pb.ConsensusType_DsBlockConsensus)
	r.dsConsensusLeader.InitiateConsensus(announce, r.peerServer.Committee, r.imp)
}

func (r *RoleDsLead) preConsensusProcessDsBlock(block proto.Message, announce *pb.Message, payload *pb.ConsensusPayload) error {
	var dsblock *pb.DSBlock
	var ok bool
	if dsblock, ok = block.(*pb.DSBlock); ok {
		sig, err := r.peerServer.MsgSinger.SignHash(dsblock.Hash().Bytes(), nil)
		if err != nil {
			loggerDsLead.Errorf("bls message sign failed with error: %s", err.Error())
			return ErrSignMessage
		}
		announce.Signature = sig
	}

	data, err := r.produceConsensusPayload(dsblock, pb.ConsensusType_DsBlockConsensus)
	if err != nil {
		return err
	}
	announce.Payload = data
	return nil
}

func (r *RoleDsLead) preConsensusProcessFinalBlock(block proto.Message, announce *pb.Message, payload *pb.ConsensusPayload) error {
	var txblock *pb.TxBlock
	var ok bool
	if txblock, ok = block.(*pb.TxBlock); ok {
		sig, err := r.peerServer.MsgSinger.SignHash(txblock.Hash().Bytes(), nil)

		if err != nil {
			loggerDsLead.Errorf("bls message sign failed with error: %s", err.Error())
			return ErrSignMessage
		}
		announce.Signature = sig
	}

	data, err := r.produceConsensusPayload(txblock, pb.ConsensusType_DsBlockConsensus)
	if err != nil {
		return err
	}
	announce.Payload = data
	return nil
}

func (r *RoleDsLead) preConsensusProcessVCBlock(block proto.Message, announce *pb.Message, payload *pb.ConsensusPayload) error {
	var vcblock *pb.VCBlock
	var ok bool
	if vcblock, ok = block.(*pb.VCBlock); ok {
		loggerDsLead.Debugf("vcblock is %+v", vcblock)
		sig, err := r.peerServer.MsgSinger.SignHash(vcblock.Hash().Bytes(), nil)
		if err != nil {
			loggerDsLead.Errorf("bls message sign failed with error: %s", err.Error())
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

func (r *RoleDsLead) produceAnnounce(block proto.Message, envelope *pb.Message, payload *pb.ConsensusPayload, consensusType pb.ConsensusType) (*pb.Message, error) {
	var err error
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		err = r.preConsensusProcessDsBlock(block, envelope, payload)
	case pb.ConsensusType_FinalBlockConsensus:
		err = r.preConsensusProcessFinalBlock(block, envelope, payload)
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
