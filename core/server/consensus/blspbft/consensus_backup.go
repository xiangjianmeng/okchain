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

package blspbft

import (
	"math"
	"reflect"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/core/blockchain"
	"github.com/ok-chain/okchain/crypto/multibls"
	logging "github.com/ok-chain/okchain/log"
	ps "github.com/ok-chain/okchain/core/server"
	pb "github.com/ok-chain/okchain/protos"
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

func (cb *ConsensusBackup) SetIRole(irole ps.IRole) {
	cb.role = irole
}

func (cb *ConsensusBackup) UpdateLeader(leader *pb.PeerEndpoint) {
	cb.leader = leader
	loggerBackup.Debugf("UpdateLeader: %s", cb.leader)
}

func (cb *ConsensusBackup) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint, r ps.IRole) error {

	if !reflect.DeepEqual(cb.GetCurrentLeader().Pubkey, from.Pubkey) {
		loggerBackup.Errorf("incorrect message source, expect %+v, actual %+v", cb.GetCurrentLeader().Id.Name, from.Id.Name)
		return ErrIncorrectMessageSource
	}
	cb.role = r
	switch msg.Type {
	case pb.Message_Consensus_Announce:
		cb.processMessageAnnounce(msg, from)
	case pb.Message_Consensus_FinalCollectiveSig:
		cb.processMessageFinalCollectiveSig(msg, from)
	default:
		return ErrInvalidMessage
	}
	return nil
}

func (cb *ConsensusBackup) send2Lead(res *pb.Message) error {
	loggerBackup.Debugf("send message to: %s", cb.leader)
	err := cb.peerServer.Multicast(res, pb.PeerEndpointList{cb.leader})
	if err != nil {
		loggerBackup.Errorf("send message to all nodes failed")
		return ErrMultiCastMessage
	}
	return nil
}

func (cb *ConsensusBackup) processMessageAnnounce(msg *pb.Message, from *pb.PeerEndpoint) error {
	// 1. verify block
	err := consensusHandler.VerifyMessage(msg, from, cb.role)
	if err != nil {
		return ErrVerifyBlock
	} else {
		loggerBackup.Debugf("verify announce message passed")
	}

	// 2. compose final response message
	response, err := consensusHandler.ComposedFinalResponse(cb.role)
	if err != nil {
		return ErrComposeMessage
	}

	loggerBackup.Debugf("final response is %+v", response)

	// 3. send result to leader
	cb.send2Lead(response)
	return nil
}

func (cb *ConsensusBackup) processMessageFinalCollectiveSig(msg *pb.Message, from *pb.PeerEndpoint) error {
	// verify final collectivesig message
	err := cb.verifyBlock(msg, from, consensusHandler.currentType)
	if err != nil {
		loggerBackup.Errorf("verify final collectivesig message failed, error: %s", err.Error())
		return ErrVerifyBlock
	} else {
		loggerBackup.Debugf("verify final collectivesig message pass")
	}

	// notify sharding backup or ds backup
	err = cb.role.OnConsensusCompleted(nil)
	if err != nil {
		return err
	}
	return nil
}

func (cb *ConsensusBackup) GetCurrentLeader() *pb.PeerEndpoint {
	return cb.leader
}

func (cb *ConsensusBackup) verifyBlock(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error {
	var err error
	var ret bool
	var multisig []byte
	var boolMap []bool
	var multiPub []byte
	var committeeSort pb.PeerEndpointList

	finalPubkey := &multibls.PubKey{}
	finalSign := &multibls.Sig{}

	consensusPayload := &pb.ConsensusPayload{}
	err = proto.Unmarshal(msg.Payload, consensusPayload)
	if err != nil {
		return ErrUnmarshalMessage
	}

	counter := 0
	expected := 0
	// sort committee
	if consensusType == pb.ConsensusType_FinalBlockConsensus || consensusType == pb.ConsensusType_DsBlockConsensus ||
		(consensusType == pb.ConsensusType_ViewChangeConsensus && cb.role.GetCurrentVCBlock().Header.Stage != "MicroBlockConsensus") {
		for i := 0; i < cb.peerServer.Committee.Len(); i++ {
			committeeSort = append(committeeSort, cb.peerServer.Committee[i])
		}
		sort.Sort(pb.ByPubkey{PeerEndpointList: committeeSort})
		expected = int(math.Floor(float64(len(committeeSort))*ToleranceFraction + 1))
	} else {
		if cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum == 1 {
			expected = int(math.Floor(float64(len(cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes))*ToleranceFraction)) + 1
		} else {
			expected = int(math.Floor(float64(len(cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)/
				int(cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum))*ToleranceFraction)) + 1
		}
	}

	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		dsblock := &pb.DSBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, dsblock)
		if err != nil {
			loggerBackup.Errorf("dsblock unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		loggerBackup.Debugf("dsblock is %+v", dsblock)
		cb.role.GetCurrentDSBlock().Header.Signature = nil

		multisig = dsblock.Header.Signature
		multiPub = dsblock.Header.MultiPubKey
		boolMap = dsblock.Header.BoolMap

		dsblock.Header.Signature = nil
		dsblock.Header.MultiPubKey = nil
		dsblock.Header.BoolMap = nil
		if !reflect.DeepEqual(cb.role.GetCurrentDSBlock(), dsblock) {
			loggerBackup.Errorf("dsblock not match")
			loggerBackup.Errorf("my dsblock is %+v", cb.role.GetCurrentDSBlock())
			loggerBackup.Errorf("msg dsblock is %+v", dsblock)
			return ErrVerifyBlock
		}

		// verify message is signed by corrected owner
		err = cb.peerServer.MsgVerify(msg, cb.role.GetCurrentDSBlock().Hash().Bytes())
		if err != nil {
			loggerBackup.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}

		// verify multi-pubkey according to boolmap
		err = pb.BlsMultiPubkeyVerify(boolMap, committeeSort, multiPub)
		if err != nil {
			loggerBackup.Errorf("bls multi pubkey verify failed with error: %s", err.Error())
			return ErrBLSPubKeyVerify
		}

		dsblock.Header.Signature = multisig
		dsblock.Header.MultiPubKey = multiPub
		dsblock.Header.BoolMap = boolMap
		finalPubkey.Deserialize(dsblock.Header.MultiPubKey)
		finalSign.Deserialize(dsblock.Header.Signature)
		// verify multi sign
		ret = finalSign.BLSVerify(finalPubkey, dsblock.MHash().Bytes())
		if !ret {
			loggerBackup.Errorf("bls multi signature verify failed")
			return ErrVerifyMessage
		} else {
			cb.role.SetCurrentDSBlock(dsblock)
		}
	case pb.ConsensusType_FinalBlockConsensus:
		txblock := &pb.TxBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, txblock)
		if err != nil {
			loggerBackup.Errorf("txblock unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		loggerBackup.Debugf("txblock is %+v", txblock)
		cb.role.GetCurrentFinalBlock().Header.Signature = nil

		multisig = txblock.Header.Signature
		multiPub = txblock.Header.MultiPubKey
		boolMap = txblock.Header.BoolMap
		coinbases := txblock.Header.DSCoinBase

		err := txblock.CheckCoinBase2(committeeSort, cb.peerServer.DsBlockChain().(*blockchain.DSBlockChain).GetPubkey2CoinbaseCache())
		if err != nil {
			return ErrCheckCoinBase
		}

		txblock.Header.Signature = nil
		txblock.Header.MultiPubKey = nil
		txblock.Header.BoolMap = nil
		cb.role.GetCurrentFinalBlock().Header.DSCoinBase = coinbases

		//txblock.Header.DSCoinBase = cb.role.GetCurrentFinalBlock().Header.DSCoinBase
		stateRoot, gasUsed, err := cb.peerServer.CalStateRoot(cb.role.GetCurrentFinalBlock())
		if err != nil {
			loggerBackup.Errorf("cal stateRoot failed, error: %s", err.Error())
			return ErrCalStateRoot
		}

		cb.role.GetCurrentFinalBlock().Header.StateRoot = stateRoot
		cb.role.GetCurrentFinalBlock().Header.GasUsed = gasUsed

		if !reflect.DeepEqual(cb.role.GetCurrentFinalBlock(), txblock) {
			loggerBackup.Errorf("final block not match")
			loggerBackup.Errorf("my txblock is %+v", cb.role.GetCurrentFinalBlock())
			loggerBackup.Errorf("msg txblock is %+v", txblock)
			return ErrVerifyBlock
		}

		cb.role.GetCurrentFinalBlock().Header.DSCoinBase = coinbases
		err = cb.peerServer.MsgVerify(msg, cb.role.GetCurrentFinalBlock().Hash().Bytes())
		if err != nil {
			loggerBackup.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}

		err = pb.BlsMultiPubkeyVerify(boolMap, committeeSort, multiPub)
		if err != nil {
			loggerBackup.Errorf("bls multi pubkey verify failed with error: %s", err.Error())
			return ErrBLSPubKeyVerify
		}

		txblock.Header.Signature = multisig
		txblock.Header.MultiPubKey = multiPub
		txblock.Header.BoolMap = boolMap
		txblock.Header.DSCoinBase = [][]byte{coinbases[0]}
		finalPubkey.Deserialize(txblock.Header.MultiPubKey)
		finalSign.Deserialize(txblock.Header.Signature)
		ret = finalSign.BLSVerify(finalPubkey, txblock.MHash().Bytes())
		if !ret {
			loggerBackup.Errorf("bls multi signature verify failed")
			return ErrVerifyMessage
		} else {
			txblock.Header.DSCoinBase = coinbases
			cb.role.SetCurrentFinalBlock(txblock)
		}
	case pb.ConsensusType_MicroBlockConsensus:
		mcblock := &pb.MicroBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, mcblock)
		if err != nil {
			loggerBackup.Errorf("micro block unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		loggerBackup.Debugf("microblock is %+v", mcblock)
		cb.role.GetCurrentMicroBlock().Header.Signature = nil

		multisig = mcblock.Header.Signature
		multiPub = mcblock.Header.MultiPubKey
		boolMap = mcblock.Header.BoolMap

		mcblock.Header.Signature = nil
		mcblock.Header.MultiPubKey = nil
		mcblock.Header.BoolMap = nil
		if !reflect.DeepEqual(cb.role.GetCurrentMicroBlock(), mcblock) {
			loggerBackup.Errorf("micro block not match")
			loggerBackup.Errorf("my micro block is %+v", cb.role.GetCurrentMicroBlock())
			loggerBackup.Errorf("msg micro block is %+v", mcblock)
			return ErrVerifyBlock
		}
		err = cb.peerServer.MsgVerify(msg, cb.role.GetCurrentMicroBlock().Hash().Bytes())
		if err != nil {
			loggerBackup.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}

		err = pb.BlsMultiPubkeyVerify(boolMap, cb.peerServer.ShardingNodes, multiPub)
		if err != nil {
			loggerBackup.Errorf("bls multi pubkey verify failed with error: %s", err.Error())
			return ErrBLSPubKeyVerify
		}

		mcblock.Header.Signature = multisig
		mcblock.Header.MultiPubKey = multiPub
		mcblock.Header.BoolMap = boolMap
		finalPubkey.Deserialize(mcblock.Header.MultiPubKey)
		finalSign.Deserialize(mcblock.Header.Signature)
		ret = finalSign.BLSVerify(finalPubkey, mcblock.Hash().Bytes())
		if !ret {
			loggerBackup.Errorf("bls multi signature verify failed")
			return ErrVerifyMessage
		} else {
			cb.role.SetCurrentMicroBlock(mcblock)
		}
	case pb.ConsensusType_ViewChangeConsensus:
		vcblock := &pb.VCBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, vcblock)
		if err != nil {
			loggerBackup.Errorf("micro block unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		loggerBackup.Debugf("vcblock is %+v", vcblock)
		cb.role.GetCurrentVCBlock().Header.Signature = nil

		multisig = vcblock.Header.Signature
		multiPub = vcblock.Header.MultiPubKey
		boolMap = vcblock.Header.BoolMap

		vcblock.Header.Signature = nil
		vcblock.Header.MultiPubKey = nil
		vcblock.Header.BoolMap = nil
		if !reflect.DeepEqual(cb.role.GetCurrentVCBlock(), vcblock) {
			loggerBackup.Errorf("viewchange block not match")
			loggerBackup.Errorf("my vcblock is %+v", cb.role.GetCurrentVCBlock())
			loggerBackup.Errorf("msg vcblock is %+v", vcblock)
			return ErrVerifyBlock
		}
		err = cb.peerServer.MsgVerify(msg, cb.role.GetCurrentVCBlock().Hash().Bytes())
		if err != nil {
			loggerBackup.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}

		// different multi pubkey verify parameters for different viewchange stage
		if vcblock.Header.Stage == "MicroBlockConsensus" {
			err = pb.BlsMultiPubkeyVerify(boolMap, cb.peerServer.ShardingNodes, multiPub)
		} else {
			err = pb.BlsMultiPubkeyVerify(boolMap, committeeSort, multiPub)
		}
		if err != nil {
			loggerBackup.Errorf("bls multi pubkey verify failed with error: %s", err.Error())
			return ErrBLSPubKeyVerify
		}

		vcblock.Header.Signature = multisig
		vcblock.Header.MultiPubKey = multiPub
		vcblock.Header.BoolMap = boolMap
		finalPubkey.Deserialize(vcblock.Header.MultiPubKey)
		finalSign.Deserialize(vcblock.Header.Signature)
		ret = finalSign.BLSVerify(finalPubkey, vcblock.Hash().Bytes())
		if !ret {
			loggerBackup.Errorf("bls multi signature verify failed")
			return ErrVerifyMessage
		} else {
			cb.role.SetCurrentVCBlock(vcblock)
		}
	}

	for i := 0; i < len(boolMap); i++ {
		if boolMap[i] {
			counter++
		}
	}

	if counter < expected {
		loggerBackup.Errorf("multi signer less than expected number, expected %d, actual %d", expected, counter)
		return ErrVerifyBlock
	}

	return nil
}
