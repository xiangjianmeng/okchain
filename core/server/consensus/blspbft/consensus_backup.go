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
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/core/blockchain"
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/crypto/multibls"
	logging "github.com/ok-chain/okchain/log"
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
	case pb.Message_Consensus_BroadCastBlock:
		cb.processMessageBroadCastBlock(msg, from)
	default:
		return ErrInvalidMessage
	}
	return nil
}

func (cb *ConsensusBackup) send2Lead(res *pb.Message) error {
	loggerBackup.Debugf("send message to: %s", cb.leader)
	go func() {
		err := cb.peerServer.Multicast(res, pb.PeerEndpointList{cb.leader})
		if err != nil {
			loggerBackup.Errorf("send message to all nodes failed")
			// return ErrMultiCastMessage
		}
	}()
	return nil
}

func (cb *ConsensusBackup) processMessageAnnounce(msg *pb.Message, from *pb.PeerEndpoint) error {
	var err error
	// verify message.
	err = cb.peerServer.MessageVerify(msg)
	if err != nil {
		loggerBackup.Errorf("message signature verify failed with error: %s", err.Error())
		return ErrVerifyMessage
	}
	// 1. verify block
	err = consensusHandler.VerifyMessage(msg, from, cb.role)
	if err != nil {
		return ErrVerifyBlock
	} else {
		loggerBackup.Debugf("verify announce message passed")
	}

	// 2. compose final response message
	response, err := consensusHandler.ComposedResponse(cb.role)
	if err != nil {
		return ErrComposeMessage
	}

	loggerBackup.Debugf("response is %+v", response)

	// 3. send result to leader
	cb.send2Lead(response)
	return nil
}

func (cb *ConsensusBackup) processMessageFinalCollectiveSig(msg *pb.Message, from *pb.PeerEndpoint) error {
	var err error
	// check if message is changed
	err = cb.peerServer.MessageVerify(msg)
	if err != nil {
		loggerBackup.Errorf("verifying signature of final collective message failed with error: %s", err.Error())
		return ErrVerifyMessage
	}
	// verify multiple signature in round 1.
	err = cb.verifyBlockSig1(msg, from, consensusHandler.currentType)
	if err != nil {
		loggerBackup.Errorf("verifying payload of final collectivesig message failed, error: %s", err.Error())
		return ErrVerifyMessage
	} else {
		loggerBackup.Debugf("verifying payload of final collectivesig message pass")
	}

	// compose final response message
	finalResponse, err := consensusHandler.ComposedFinalResponse(cb.role)
	if err != nil {
		return ErrComposeMessage
	}

	loggerBackup.Debugf("final response is %+v", finalResponse)

	// send result to leader
	cb.send2Lead(finalResponse)
	return nil
}

func (cb *ConsensusBackup) processMessageBroadCastBlock(msg *pb.Message, from *pb.PeerEndpoint) error {
	var err error
	// verify message
	err = cb.peerServer.MessageVerify(msg)
	if err != nil {
		loggerBackup.Errorf("verify signature of broadcast message failed, error: %s", err.Error())
		return ErrVerifyMessage
	}
	loggerBackup.Debugf("verify broad cast message passed")
	// verify broadcast block message
	err = cb.verifyBlock(msg, from, consensusHandler.currentType)
	if err != nil {
		loggerBackup.Errorf("verify payload of broadcast message failed, error: %s", err.Error())
		return ErrVerifyMessage
	} else {
		loggerBackup.Debugf("verify payload of broadcast message  pass")
	}
	loggerBackup.Debugf("verify broad cast block passed")
	// notify sharding backup or ds backup
	err = cb.role.OnConsensusCompleted(nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (cb *ConsensusBackup) GetCurrentLeader() *pb.PeerEndpoint {
	return cb.leader
}

func (cb *ConsensusBackup) verifyBlockSig1(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error {
	consensusPayload := &pb.ConsensusPayload{}
	blockSig1 := &pb.BoolMapSignature{}
	multiPubkey := &multibls.PubKey{}
	sig1 := &multibls.Sig{}
	var err error
	var committeeSort pb.PeerEndpointList
	expected := 0
	counter := 0
	if consensusType != pb.ConsensusType_FinalBlockConsensus && consensusType != pb.ConsensusType_DsBlockConsensus && consensusType != pb.ConsensusType_ViewChangeConsensus ||
		(consensusType == pb.ConsensusType_ViewChangeConsensus && cb.role.GetCurrentVCBlock().Header.Stage == "MicroBlockConsensus") {
		return fmt.Errorf("Can not process this type of message: %d", consensusType)
	}
	for i := 0; i < cb.peerServer.Committee.Len(); i++ {
		committeeSort = append(committeeSort, cb.peerServer.Committee[i])
	}
	sort.Sort(pb.ByPubkey{PeerEndpointList: committeeSort})
	expected = int(math.Floor(float64(len(committeeSort))*ToleranceFraction + 1))

	// deserialize payload to blockSig1.
	err = proto.Unmarshal(msg.Payload, consensusPayload)
	if err != nil {
		return ErrUnmarshalMessage
	}
	err = proto.Unmarshal(consensusPayload.Msg, blockSig1)
	if err != nil {
		return ErrUnmarshalMessage
	}
	for i := 0; i < len(blockSig1.BoolMap); i++ {
		pubKey := multibls.PubKey{}
		if blockSig1.BoolMap[i] {
			pubKey.Deserialize(committeeSort[i].Pubkey)
			multiPubkey.Add(&pubKey.PublicKey)
			counter++
		}
	}
	if counter < expected {
		return fmt.Errorf("signatures in round 1 are not enough")
	}
	err = sig1.Deserialize(blockSig1.Signature)
	if err != nil {
		return fmt.Errorf("deserializing signatures in round 1 failed")
	}
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		verified := sig1.BLSVerify(multiPubkey, cb.role.GetCurrentDSBlock().MHash().Bytes())
		if !verified {
			return fmt.Errorf("verifying signatures in round 1 failed.")
		}
		cb.role.GetCurrentDSBlock().Header.Signature = blockSig1.Signature
	case pb.ConsensusType_FinalBlockConsensus:
		DSCoinBase := cb.role.GetCurrentFinalBlock().Header.DSCoinBase
		cb.role.GetCurrentFinalBlock().Header.DSCoinBase = [][]byte{DSCoinBase[0]}
		verified := sig1.BLSVerify(multiPubkey, cb.role.GetCurrentFinalBlock().MHash().Bytes())
		cb.role.GetCurrentFinalBlock().Header.DSCoinBase = DSCoinBase
		if !verified {
			return fmt.Errorf("verifying signatures in round 1 failed")
		}
		cb.role.GetCurrentFinalBlock().Header.Signature = blockSig1.Signature
	case pb.ConsensusType_ViewChangeConsensus:
		verified := sig1.BLSVerify(multiPubkey, cb.role.GetCurrentVCBlock().Hash().Bytes())
		if !verified {
			return fmt.Errorf("verifying signatures in round 1 failed.")
		}
		cb.role.GetCurrentFinalBlock().Header.Signature = blockSig1.Signature
	}
	return nil
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
		expected = int(math.Floor(float64(len(committeeSort)-1)*ToleranceFraction + 1))
	} else {
		if cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum == 1 {
			expected = int(math.Floor(float64(len(cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)-1)*ToleranceFraction)) + 1
		} else {
			expected = int(math.Floor(float64(len(cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)/
				int(cb.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum)-1)*ToleranceFraction)) + 1
		}
	}

	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		dsBlockSig2 := &pb.DSBlockWithSig2{}
		err = proto.Unmarshal(consensusPayload.Msg, dsBlockSig2)
		if err != nil {
			loggerBackup.Errorf("txBlockSig2 unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		dsblock := dsBlockSig2.Block
		// verify multiple signature in round 2
		multiSig2 := &multibls.Sig{}
		err = multiSig2.Deserialize(dsBlockSig2.Sig2.Signature)
		if err != nil {
			return err
		}
		err = cb.peerServer.VerifyMultiSignByBoolMap(dsBlockSig2.Sig2.BoolMap, committeeSort, multiSig2, dsblock.Header.Signature, expected)
		if err != nil {
			return err
		}
		// verify DSBlock
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
		txBlockSig2 := &pb.TxBlockWithSig2{}
		var txblock *pb.TxBlock
		err = proto.Unmarshal(consensusPayload.Msg, txBlockSig2)
		if err != nil {
			loggerBackup.Errorf("txBlockSig2 unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		txblock = txBlockSig2.Block
		// verify multiple signature in round 2
		multiSig2 := &multibls.Sig{}
		err = multiSig2.Deserialize(txBlockSig2.Sig2.Signature)
		if err != nil {
			return err
		}
		err = cb.peerServer.VerifyMultiSignByBoolMap(txBlockSig2.Sig2.BoolMap, committeeSort, multiSig2, txblock.Header.Signature, expected)
		if err != nil {
			return err
		}
		// verify txblock
		loggerBackup.Debugf("txblock is %+v", txblock)
		cb.role.GetCurrentFinalBlock().Header.Signature = nil

		multisig = txblock.Header.Signature
		multiPub = txblock.Header.MultiPubKey
		boolMap = txblock.Header.BoolMap
		coinbases := txblock.Header.DSCoinBase

		err = txblock.CheckCoinBase2(committeeSort, cb.peerServer.DsBlockChain().(*blockchain.DSBlockChain).GetPubkey2CoinbaseCache())
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
		vcBlockSig2 := &pb.VCBlockWithSig2{}
		vcblock := &pb.VCBlock{}
		if cb.role.GetCurrentVCBlock().Header.Stage == "MicroBlockConsensus" {
			err = proto.Unmarshal(consensusPayload.Msg, vcBlockSig2)
			if err != nil {
				loggerBackup.Errorf("txBlockSig2 unmarshal failed with error: %s", err.Error())
				return ErrUnmarshalMessage
			}
			vcblock = vcBlockSig2.Block
			// verify multiple signature in round 2
			multiSig2 := &multibls.Sig{}
			err = multiSig2.Deserialize(vcBlockSig2.Sig2.Signature)
			if err != nil {
				return err
			}
			err = cb.peerServer.VerifyMultiSignByBoolMap(vcBlockSig2.Sig2.BoolMap, committeeSort, multiSig2, vcblock.Header.Signature, expected)
			if err != nil {
				return err
			}
		} else {
			err = proto.Unmarshal(consensusPayload.Msg, vcblock)
			if err != nil {
				loggerBackup.Errorf("micro block unmarshal failed with error: %s", err.Error())
				return ErrUnmarshalMessage
			}
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
