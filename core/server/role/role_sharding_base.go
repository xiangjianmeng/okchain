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

package role

import (
	"bytes"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/crypto/multibls"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
)

type RoleShardingBase struct {
	*RoleBase
	currentMicroBlock *pb.MicroBlock
}

var loggerShardingBase = logging.MustGetLogger("shardingBase")

func newRoleShardingBase(peer *ps.PeerServer) *RoleShardingBase {
	base := &RoleBase{peerServer: peer}
	r := &RoleShardingBase{RoleBase: base}

	return r
}

func (r *RoleShardingBase) OnConsensusCompleted(err error, boolMapSign2 *pb.BoolMapSignature) error {
	stateType := reflect.TypeOf(r.state).String()
	loggerShardingBase.Debugf("Invoked: state<%s>", stateType)

	r.peerServer.VcChan <- "finished"
	// todo: use state machine to handle OnConsensusCompleted in each state
	switch stateType {
	case "*state.STATE_MICROBLOCK_CONSENSUS":
		loggerShardingBase.Debugf("micro block consensus completed")
		err = r.imp.onMircoBlockConsensusCompleted(err)
	case "*state.STATE_VIEWCHANGE_CONSENSUS":
		loggerShardingBase.Debugf("viewchange consensus completed")
		err = r.imp.onViewChangeConsensusCompleted(err)
	default:
		return ErrHandleInCurrentState
	}

	r.peerServer.DumpRoleAndState()
	if err != nil {
		return err
	}
	return nil
}

func (r *RoleShardingBase) composeVCBlock() (*pb.VCBlock, error) {
	vcblock := &pb.VCBlock{}
	vcblock.Header = &pb.VCBlockHeader{}
	vcblock.Header.Timestamp = pb.CreateUtcTimestamp()
	vcblock.Header.ViewChangeDSNo = r.peerServer.DsBlockChain().CurrentBlock().NumberU64()
	vcblock.Header.ViewChangeTXNo = r.peerServer.TxBlockChain().CurrentBlock().NumberU64()
	vcblock.Header.Stage = pb.ConsensusType_name[int32(r.peerServer.ConsensusData.LastStage)]
	vcblock.Header.NewLeader = r.GetNewLeader(r.peerServer.ConsensusData.CurrentStage, r.peerServer.ConsensusData.LastStage)
	r.SetCurrentVCBlock(vcblock)
	return vcblock, nil
}

func (r *RoleShardingBase) ProcessDSBlock(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
	// verify message signature.
	var err error
	peerServer := r.peerServer
	err = peerServer.MessageVerify(pbMsg)
	if err != nil {
		idleLogger.Errorf("message signature verify failed with error: %s", err.Error())
		return ErrVerifyMessage
	}
	// verify signature in round 2.
	dsBlockSign2 := &pb.DSBlockWithSig2{}
	err = proto.Unmarshal(pbMsg.Payload, dsBlockSign2)
	if err != nil {
		idleLogger.Errorf("unmarshal dsBlockSign2 message failed with error: %s", err.Error())
		return err
	}
	err = peerServer.VerifyDSBlockMultiSign2(dsBlockSign2)
	if err != nil {
		idleLogger.Errorf("verify dsBlockSign2 failed with error: %s", err.Error())
		return nil
	}
	// verify and insert dsblock.
	dsblock := dsBlockSign2.Block

	err = r.onDsBlockReady(dsblock)
	if err != nil {
		loggerShardingBase.Errorf("handle dsblock error: %s", err.Error())
		return err
	}

	if bytes.Equal(dsblock.Header.WinnerPubKey, peerServer.GetPubkey()) {
		loggerShardingBase.Infof("I am new DS committee leader!")
		r.peerServer.ChangeRole(ps.PeerRole_DsLead, ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
	} else {
		r.initShardingNodeInfo(dsblock)
		index := r.peerServer.ShardingNodes.Index(peerServer.SelfNode)

		targetRole := ps.PeerRole_ShardingBackup
		targetState := ps.STATE_MICROBLOCK_CONSENSUS
		if index == 0 {
			loggerShardingBase.Infof("I am this round ShardingLead")
			targetRole = ps.PeerRole_ShardingLead
			newRole := r.peerServer.ChangeRole(targetRole, targetState)
			err = newRole.OnMircoBlockConsensusStarted(r.peerServer.ShardingNodes[0])
		} else if index == -1 {
			loggerShardingBase.Infof("I am this round Idle")
			targetRole = ps.PeerRole_Idle
			targetState = ps.STATE_IDLE
			r.peerServer.ChangeRole(targetRole, targetState)
		} else {
			loggerShardingBase.Infof("I am this round ShardingBackup")
			newRole := r.peerServer.ChangeRole(targetRole, targetState)
			err = newRole.OnMircoBlockConsensusStarted(r.peerServer.ShardingNodes[0])
		}
	}

	if err != nil {
		return err
	}
	r.peerServer.DumpRoleAndState()

	return nil
}

func (r *RoleShardingBase) ProcessFinalBlock(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
	// verify message signature.
	var err error
	peerServer := r.peerServer
	err = peerServer.MessageVerify(pbMsg)
	if err != nil {
		idleLogger.Errorf("message signature verify failed with error: %s", err.Error())
		return ErrVerifyMessage
	}
	// verify signature in round 2.
	txBlockSign2 := &pb.TxBlockWithSig2{}
	err = proto.Unmarshal(pbMsg.Payload, txBlockSign2)
	if err != nil {
		idleLogger.Errorf("unmarshal dsBlockSign2 message failed with error: %s", err.Error())
		return err
	}
	err = peerServer.VerifyFinalBlockMultiSign2(txBlockSign2)
	if err != nil {
		idleLogger.Errorf("verify dsBlockSign2 failed with error: %s", err.Error())
		return nil
	}
	// verify and insert dsblock.
	txblock := txBlockSign2.Block

	r.onFinalBlockReady(txblock)

	numOfSharding := len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes) / r.peerServer.GetShardingSize()

	if (numOfSharding != 0 && txblock.Header.BlockNumber%uint64(r.peerServer.GetShardingSize()) == 0) ||
		(numOfSharding == 0 && txblock.Header.BlockNumber%uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)) == 0) {
		// start pow
		loggerShardingBase.Debugf("start pow, waiting for find a results")

		// only for local testing, sleep 1s and start mine
		time.Sleep(1 * time.Second)

		// call miner function
		pk := multibls.PubKey{}
		pk.Deserialize(r.peerServer.SelfNode.Pubkey)
		miningResult, err := r.mining(pk.GetHexString(), txblock.Header.BlockNumber)

		if err != nil {
			return ErrMine
		}

		powSubmission := r.composePoWSubmission(miningResult, txblock.Header.BlockNumber, r.peerServer.SelfNode.Pubkey)

		loggerShardingBase.Debugf("powsubmission is %+v", powSubmission)

		data, err := proto.Marshal(powSubmission)
		if err != nil {
			loggerShardingBase.Errorf("PoWSubmission marshal error %s", err.Error())
			return ErrMarshalMessage
		}
		res := &pb.Message{}
		res.Type = pb.Message_DS_PoWSubmission
		res.Timestamp = pb.CreateUtcTimestamp()
		res.Payload = data
		res.Peer = r.peerServer.SelfNode
		sig, err := r.peerServer.MsgSinger.SignHash(powSubmission.Hash().Bytes(), nil)
		if err != nil {
			loggerShardingBase.Errorf("bls message sign failed with error: %s", err.Error())
			return ErrSignMessage
		}
		res.Signature = sig

		err = r.peerServer.Multicast(res, r.peerServer.Committee)
		if err != nil {
			loggerShardingBase.Errorf("send message to all DS nodes failed")
			return ErrMultiCastMessage
		}

		r.ChangeState(ps.STATE_WAITING_DSBLOCK)
		loggerShardingBase.Debugf("waiting dsblock...")
	} else {

		index := r.peerServer.ShardingNodes.Index(r.peerServer.SelfNode)

		numOfSharding := len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes) / r.peerServer.GetShardingSize()

		targetRole := ps.PeerRole_ShardingBackup
		if (numOfSharding != 0 && uint64(index) == txblock.Header.BlockNumber%uint64(r.peerServer.GetShardingSize())) ||
			(numOfSharding == 0 && uint64(index) == txblock.Header.BlockNumber%uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes))) {
			targetRole = ps.PeerRole_ShardingLead
			loggerShardingBase.Infof("I am this round ShardingLead")
		} else {
			loggerShardingBase.Infof("I am this round ShardingBackup")
		}

		round := 0
		if numOfSharding != 0 {
			round = int(txblock.Header.BlockNumber % uint64(r.peerServer.GetShardingSize()))
		} else {
			round = int(txblock.Header.BlockNumber % uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)))
		}

		newRole := r.peerServer.ChangeRole(targetRole, ps.STATE_MICROBLOCK_CONSENSUS)
		err = newRole.OnMircoBlockConsensusStarted(r.peerServer.ShardingNodes[round])
	}

	if err != nil {
		return err
	}
	return nil
}
