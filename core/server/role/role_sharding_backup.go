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
* * DESCRIPTION
*
 */

package role

import (
	"encoding/hex"
	"reflect"

	"github.com/golang/protobuf/proto"
	logging "github.com/ok-chain/okchain/log"
	ps "github.com/ok-chain/okchain/core/server"
	pb "github.com/ok-chain/okchain/protos"
)

type RoleShardingBackup struct {
	*RoleShardingBase
	consensusBackup ps.IConsensusBackup
}

var loggerShardingBackup = logging.MustGetLogger("shardingBackupRole")

func newRoleShardingBackup(peer *ps.PeerServer) ps.IRole {
	base := newRoleShardingBase(peer)
	r := &RoleShardingBackup{RoleShardingBase: base}
	r.initBase(r)
	r.name = "ShardBackup"
	r.consensusBackup = peer.ProduceConsensusBackup(peer)
	r.consensusBackup.SetIRole(r)
	return r
}

func (r *RoleShardingBackup) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	err := r.consensusBackup.ProcessConsensusMsg(msg, from, r)
	if err != nil {
		loggerShardingBackup.Errorf("handle consensus message failed with error: %s", err.Error())
		return err
	}
	return nil
}

func (r *RoleShardingBackup) onMircoBlockConsensusCompleted(err error) error {
	r.ChangeState(ps.STATE_WAITING_FINALBLOCK)
	loggerShardingBackup.Debugf("change state to STATE_WAITING_FINALBLOCK")
	r.DumpCurrentState()
	return nil
}

func (r *RoleShardingBackup) onViewChangeConsensusCompleted(err error) error {
	switch r.GetCurrentVCBlock().Header.Stage {
	case "MicroBlockConsensus":
		r.consensusBackup.SetCurrentConsensusStage(pb.ConsensusType_MicroBlockConsensus)
		r.ChangeState(ps.STATE_MICROBLOCK_CONSENSUS)
		r.DumpCurrentState()

		go r.consensusBackup.WaitForViewChange()
	default:
		return ErrHandleInCurrentState
	}
	return nil
}

func (r *RoleShardingBackup) OnMircoBlockConsensusStarted(peer *pb.PeerEndpoint) error {
	go r.consensusBackup.WaitForViewChange()

	r.consensusBackup.SetCurrentConsensusStage(pb.ConsensusType_MicroBlockConsensus)
	// update leader before micro block consensus
	r.consensusBackup.UpdateLeader(peer)
	return nil
}

func (r *RoleShardingBackup) OnViewChangeConsensusStarted() error {
	go r.consensusBackup.WaitForViewChange()

	r.consensusBackup.SetCurrentConsensusStage(pb.ConsensusType_ViewChangeConsensus)
	// update leader before viewchange consensus
	r.consensusBackup.UpdateLeader(r.GetNewLeader(r.peerServer.ConsensusData.CurrentStage, r.peerServer.ConsensusData.LastStage))
	vcblock, err := r.composeVCBlock()
	if err != nil {
		return ErrComposeBlock
	}
	r.SetCurrentVCBlock(vcblock)
	return nil
}

func (r *RoleShardingBackup) VerifyBlock(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error {
	var err error
	switch consensusType {
	case pb.ConsensusType_MicroBlockConsensus:
		err = r.verifyMicroBlock(msg, consensusType)
	case pb.ConsensusType_ViewChangeConsensus:
		vcblock, err := r.VerifyVCBlock(msg, from)
		if err != nil {
			return ErrVerifyBlock
		}
		r.SetCurrentVCBlock(vcblock)
	default:
		return ErrHandleInCurrentState
	}

	if err != nil {
		loggerShardingBackup.Debugf("verify block failed with error: %s", err.Error())
		return ErrVerifyBlock
	}
	return nil
}

func (r *RoleShardingBackup) verifyMicroBlock(msg *pb.Message, consensusType pb.ConsensusType) error {
	announce := &pb.ConsensusPayload{}
	err := proto.Unmarshal(msg.Payload, announce)
	if err != nil {
		loggerShardingBackup.Errorf("unmarshal consensus payload failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	microBlock := &pb.MicroBlock{}
	err = proto.Unmarshal(announce.Msg, microBlock)
	if err != nil {
		loggerShardingBackup.Errorf("unmarshal micro block failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	if microBlock.ShardID != r.peerServer.ShardingID {
		loggerShardingBackup.Errorf("shardingID is not as expected")
		return ErrVerifyBlock
	}

	if microBlock.Header.DSBlockNum != r.peerServer.DsBlockChain().CurrentBlock().NumberU64() {
		loggerShardingBackup.Errorf("dsblock number is not equal")
		return ErrVerifyBlock
	}

	if !reflect.DeepEqual(microBlock.Header.DSBlockHash, r.peerServer.DsBlockChain().CurrentBlock().Hash().Bytes()) {
		loggerShardingBackup.Errorf("dsblock hash is not equal")
		return ErrVerifyBlock
	}

	for _, v := range microBlock.Transactions {
		if !r.peerServer.IsTxValidated(v) {
			loggerShardingBackup.Errorf("invalid transaction: %+v", v)
			return ErrVerifyTransaciton
		}
	}

	if !pb.VerifyMerkelRoot(microBlock.Transactions, microBlock.Header.TxRoot) {
		loggerShardingBackup.Errorf("txroot verify failed")
		loggerShardingBackup.Errorf("microblock txroot is %s", hex.EncodeToString(microBlock.Header.TxRoot))
		loggerShardingBackup.Errorf("my txroot is %s", hex.EncodeToString(pb.ComputeMerkelRoot(microBlock.Transactions)))
		return ErrVerifyBlock
	}

	r.currentMicroBlock = microBlock
	r.SetCurrentMicroBlock(microBlock)

	return nil
}

func (r *RoleShardingBackup) produceFinalResponse(block proto.Message, envelope *pb.Message, payload *pb.ConsensusPayload, consensusType pb.ConsensusType) (*pb.Message, error) {
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

func (r *RoleShardingBackup) preConsensusProcessMicroBlock(block proto.Message, response *pb.Message, payload *pb.ConsensusPayload) error {
	sign := r.peerServer.PrivateKey.BlSSign(r.currentMicroBlock.Hash().Bytes())
	r.currentMicroBlock.Header.Signature = sign.Serialize()

	sig, err := r.peerServer.MsgSinger.SignHash(r.currentMicroBlock.Hash().Bytes(), nil)
	if err != nil {
		loggerShardingBackup.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}
	response.Signature = sig

	data, err := r.produceConsensusPayload(r.GetCurrentMicroBlock(), pb.ConsensusType_MicroBlockConsensus)
	if err != nil {
		return err
	}
	response.Payload = data
	return nil
}

func (r *RoleShardingBackup) preConsensusProcessVCBlock(block proto.Message, response *pb.Message, payload *pb.ConsensusPayload) error {
	sign := r.peerServer.PrivateKey.BlSSign(r.GetCurrentVCBlock().Hash().Bytes())
	r.GetCurrentVCBlock().Header.Signature = sign.Serialize()

	sig, err := r.peerServer.MsgSinger.SignHash(r.GetCurrentVCBlock().Hash().Bytes(), nil)
	if err != nil {
		loggerShardingBackup.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}
	response.Signature = sig

	data, err := r.produceConsensusPayload(r.GetCurrentVCBlock(), pb.ConsensusType_ViewChangeConsensus)
	if err != nil {
		return err
	}
	response.Payload = data
	return nil
}

func (r *RoleShardingBackup) ComposeFinalResponse(consensusType pb.ConsensusType) (*pb.Message, error) {
	response, err := r.produceConsensusMessage(consensusType, pb.Message_Consensus_FinalResponse)
	if err != nil {
		loggerShardingBackup.Errorf("compose micro block consensus failed with error: %s", err.Error())
		return response, err
	}

	return response, nil
}

func (r *RoleShardingBackup) getConsensusData(consensusType pb.ConsensusType) (proto.Message, error) {
	switch consensusType {
	case pb.ConsensusType_MicroBlockConsensus:
		return r.GetCurrentMicroBlock(), nil
	case pb.ConsensusType_ViewChangeConsensus:
		return r.GetCurrentVCBlock(), nil
	default:
		return nil, ErrHandleInCurrentState
	}
}
