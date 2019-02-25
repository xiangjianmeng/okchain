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
* BLS 共识 lead 节点实现
 */

package blspbft

import (
	"math"
	"reflect"
	"sort"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/core/blockchain"
	"github.com/ok-chain/okchain/crypto/multibls"
	logging "github.com/ok-chain/okchain/log"
	ps "github.com/ok-chain/okchain/core/server"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

var (
	loggerLead   = logging.MustGetLogger("consensusLead")
	finalBLSSign multibls.Sig
	FinalPubkey  []byte
	GPubkey      multibls.PubKey
)

type ConsensusLead struct {
	*ConsensusBase
	counter              int
	toleranceSize        int
	mux                  sync.Mutex
	consensusBackupNodes pb.PeerEndpointList
	publicKey2MsgMap     map[string]bool
	signMap              map[string]*multibls.Sig
	pubKeyMap            map[string]*multibls.PubKey
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

func (cl *ConsensusLead) SetIRole(irole ps.IRole) {
	cl.role = irole
}

// ProcessConsensusMsg process consensus message
func (cl *ConsensusLead) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint, r ps.IRole) error {
	if !cl.consensusBackupNodes.Has(from) {
		loggerLead.Errorf("incorrect message source, actual %+v not in backup nodes", from.Id.Name)
		return ErrIncorrectMessageSource
	}
	cl.role = r

	switch msg.Type {
	case pb.Message_Consensus_FinalResponse:
		cl.processMessageFinalResponse(msg, from)
	default:
		return ErrInvalidMessage
	}
	return nil
}

// InitiateConsensus
func (cl *ConsensusLead) InitiateConsensus(msg *pb.Message, to []*pb.PeerEndpoint, r ps.IRole) error {
	cl.state = ANNOUNCE_DONE
	cl.counter = 0
	cl.consensusBackupNodes = to
	cl.toleranceSize = int(math.Floor(float64(cl.consensusBackupNodes.Length())*ToleranceFraction)) + 1

	loggerLead.Debugf("send announce messages to nodes")
	cl.sendMsg2Backups(msg)

	return nil
}

func (cl *ConsensusLead) sendMsg2Backups(msg *pb.Message) error {
	err := cl.peerServer.Multicast(msg, cl.consensusBackupNodes)
	if err != nil {
		loggerLead.Errorf("send message to all nodes failed")
		return ErrMultiCastMessage
	}

	return nil
}

func (cl *ConsensusLead) handleFinalResponse(curState, nextState ConsensusState_Type, msg *pb.Message, from *pb.PeerEndpoint) bool {
	send2Backups := false

	loggerLead.Debugf("%s: update counter<%s>", util.GId, msg.Type.String())
	defer loggerLead.Debugf("%s: update counter<%s> done", util.GId, msg.Type.String())
	cl.mux.Lock()
	defer cl.mux.Unlock()
	if cl.state == curState {

		loggerLead.Debugf("%s: verify<%s>", util.GId, msg.Type.String())
		loggerLead.Debugf("verify block in final response passed")
		// 1. verify block
		err := cl.verifyAndUpdate(msg, from, consensusHandler.currentType)
		if err != nil {
			return false
		}

		// update counter
		cl.counter++
		if cl.counter == cl.toleranceSize {
			// 1. generate boolmap according to current consensus type
			err = cl.generateBoolMap(consensusHandler.currentType)
			if err != nil {
				loggerLead.Errorf("generate bit map error: %s", err.Error())
				return false
			}
			loggerLead.Debugf("generate bit map passed")

			loggerLead.Debugf("current consensus type is %+v", consensusHandler.currentType)

			//// testing for ds consensus viewchange
			//if cl.role.GetCurrentDSBlock().Header != nil {
			//	if cl.role.GetCurrentDSBlock().Header.BlockNumber == 1 && consensusHandler.currentType == pb.ConsensusType_DsBlockConsensus && cl.peerServer.Flag == false {
			//		ipPort := viper.GetString("peer.listenAddress")
			//		if strings.Contains(ipPort, "15001") {
			//			cl.peerServer.Flag = true
			//			loggerLead.Debugf("*****************************")
			//			loggerLead.Debugf("waiting!!!!!!!!!!!!!!!!!!")
			//			loggerLead.Debugf("*****************************")
			//			time.Sleep(30 * time.Second)
			//			return false
			//		}
			//	}
			//}

			// testing for final block consensus
			//if cl.role.GetCurrentFinalBlock().Header != nil {
			//	if cl.role.GetCurrentFinalBlock().Header.BlockNumber == 1 && consensusHandler.currentType == pb.ConsensusType_FinalBlockConsensus && cl.peerServer.Flag == false {
			//		ipPort := viper.GetString("peer.listenAddress")
			//		if strings.Contains(ipPort, strconv.Itoa(int(cl.peerServer.DsBlockChain().GetBlockByNumber(1).(*pb.DSBlock).Header.NewLeader.Port))) {
			//			cl.peerServer.Flag = true
			//			loggerLead.Debugf("*****************************")
			//			loggerLead.Debugf("waiting!!!!!!!!!!!!!!!!!!")
			//			loggerLead.Debugf("*****************************")
			//			time.Sleep(30 * time.Second)
			//			return false
			//		}
			//	}
			//}

			// testing for micro block consensus
			//if cl.role.GetCurrentMicroBlock().Header != nil {
			//	if cl.role.GetCurrentMicroBlock().Header.BlockNumber == 1 && consensusHandler.currentType == pb.ConsensusType_MicroBlockConsensus && cl.peerServer.Flag == false {
			//		ipPort := viper.GetString("peer.listenAddress")
			//		if strings.Contains(ipPort, "15004") {
			//			cl.peerServer.Flag = true
			//			loggerLead.Debugf("*****************************")
			//			loggerLead.Debugf("waiting!!!!!!!!!!!!!!!!!!")
			//			loggerLead.Debugf("*****************************")
			//			time.Sleep(30 * time.Second)
			//			return false
			//		}
			//	}
			//}

			// 2. compose final collectivesig message
			msg, err := cl.composeFinalCollectiveSig(consensusHandler.currentType)
			if err != nil {
				loggerLead.Errorf("compose final collectivesig message error: %s", err.Error())
				return false
			}

			loggerLead.Debugf("compose final collectivesig message successed")
			// reset counter
			cl.counter = 0
			cl.state = nextState
			// flag is true only when counter is equal to toleranceSize and compose message success
			send2Backups = true
			loggerLead.Debugf("send final collectivesig message to nodes")
			// 3. send message to backup nodes
			cl.sendMsg2Backups(msg)
		}
	}

	return send2Backups
}

func (cl *ConsensusLead) processMessageFinalResponse(msg *pb.Message, from *pb.PeerEndpoint) error {
	res := cl.handleFinalResponse(ANNOUNCE_DONE, FINALCOLLECTIVESIG_DONE, msg, from)

	if res {
		// reset counter and all relevant map
		cl.counter = 0
		cl.publicKey2MsgMap = make(map[string]bool)
		cl.pubKeyMap = make(map[string]*multibls.PubKey)
		cl.signMap = make(map[string]*multibls.Sig)
		// notify sharding lead or ds lead
		err := cl.role.OnConsensusCompleted(nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cl *ConsensusLead) GetCurrentLeader() *pb.PeerEndpoint {
	return cl.leader
}

func (cl *ConsensusLead) verifyAndUpdate(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error {
	var err error
	var ret bool
	var multiSig []byte

	tempPubKey := &multibls.PubKey{}
	tempPubKey.Deserialize(msg.Peer.Pubkey)
	tempSign := &multibls.Sig{}

	consensusPayload := &pb.ConsensusPayload{}
	err = proto.Unmarshal(msg.Payload, consensusPayload)
	if err != nil {
		loggerLead.Errorf("unmarshalling final response message error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	// init map
	if cl.publicKey2MsgMap == nil {
		cl.publicKey2MsgMap = make(map[string]bool)
		cl.signMap = make(map[string]*multibls.Sig)
		cl.pubKeyMap = make(map[string]*multibls.PubKey)
	}

	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		dsblock := &pb.DSBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, dsblock)
		if err != nil {
			loggerLead.Errorf("dsblock unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		multiSig = dsblock.Header.Signature
		dsblock.Header.Signature = nil
		if !reflect.DeepEqual(cl.role.GetCurrentDSBlock(), dsblock) {
			loggerLead.Errorf("dsblock not match")
			loggerLead.Errorf("msg is from peer %+v", from)
			loggerLead.Errorf("my dsblock is %+v", cl.role.GetCurrentDSBlock())
			loggerLead.Errorf("msg dsblock is %+v", dsblock)
			return ErrVerifyBlock
		}
		err = cl.peerServer.MsgVerify(msg, cl.role.GetCurrentDSBlock().Hash().Bytes())
		if err != nil {
			loggerLead.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}
		dsblock.Header.Signature = multiSig
		tempSign.Deserialize(multiSig)
		ret = tempSign.BLSVerify(tempPubKey, dsblock.MHash().Bytes())

	case pb.ConsensusType_FinalBlockConsensus:
		txblock := &pb.TxBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, txblock)
		if err != nil {
			loggerLead.Errorf("txblock unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		multiSig = txblock.Header.Signature
		txblock.Header.Signature = nil

		// temporary store coinbase, and reset according to consensus message
		coinbase := cl.role.GetCurrentFinalBlock().Header.DSCoinBase
		//pubStr := tempPubKey.GetHexString()
		//cl.role.GetCurrentFinalBlock().Header.DSCoinBase = append([][]byte{coinbase[0]}, cl.peerServer.ConsensusData.PubKeyToCoinBaseMap[pubStr])
		ptocoin, res := cl.peerServer.DsBlockChain().(*blockchain.DSBlockChain).GetPubkey2CoinbaseCache().GetCoinbaseByPubkey(tempPubKey.Serialize())
		if !res {
			loggerLead.Errorf("cannot get coinbase of pubkey: %s", tempPubKey.GetHexString())
			return ErrGetCoinBase
		}

		cl.role.GetCurrentFinalBlock().Header.DSCoinBase = append([][]byte{coinbase[0]}, ptocoin)
		if !reflect.DeepEqual(cl.role.GetCurrentFinalBlock(), txblock) {
			loggerLead.Errorf("final block not match")
			loggerLead.Errorf("my txblock is %+v", cl.role.GetCurrentFinalBlock())
			loggerLead.Errorf("msg txblock is %+v", txblock)
			return ErrVerifyBlock
		}
		err = cl.peerServer.MsgVerify(msg, cl.role.GetCurrentFinalBlock().Hash().Bytes())
		if err != nil {
			loggerLead.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}
		txblock.Header.Signature = multiSig
		tempSign.Deserialize(multiSig)
		txblock.Header.DSCoinBase = [][]byte{coinbase[0]}
		ret = tempSign.BLSVerify(tempPubKey, txblock.MHash().Bytes())

		//cl.role.GetCurrentFinalBlock().Header.DSCoinBase = append(coinbase, cl.peerServer.ConsensusData.PubKeyToCoinBaseMap[pubStr])
		cl.role.GetCurrentFinalBlock().Header.DSCoinBase = append(coinbase, ptocoin)
	case pb.ConsensusType_MicroBlockConsensus:
		mcblock := &pb.MicroBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, mcblock)
		if err != nil {
			loggerLead.Errorf("micro block unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		multiSig = mcblock.Header.Signature
		mcblock.Header.Signature = nil
		if !reflect.DeepEqual(cl.role.GetCurrentMicroBlock(), mcblock) {
			loggerLead.Errorf("micro block not match")
			loggerLead.Errorf("my micro block is %+v", cl.role.GetCurrentMicroBlock())
			loggerLead.Errorf("msg micro block is %+v", mcblock)
			return ErrVerifyBlock
		}
		err = cl.peerServer.MsgVerify(msg, cl.role.GetCurrentMicroBlock().Hash().Bytes())
		if err != nil {
			loggerLead.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}
		mcblock.Header.Signature = multiSig
		tempSign.Deserialize(multiSig)
		ret = tempSign.BLSVerify(tempPubKey, mcblock.Hash().Bytes())
	case pb.ConsensusType_ViewChangeConsensus:
		vcblock := &pb.VCBlock{}
		err = proto.Unmarshal(consensusPayload.Msg, vcblock)
		if err != nil {
			loggerLead.Errorf("viewchange block unmarshal failed with error: %s", err.Error())
			return ErrUnmarshalMessage
		}
		multiSig = vcblock.Header.Signature
		vcblock.Header.Signature = nil
		if !reflect.DeepEqual(cl.role.GetCurrentVCBlock(), vcblock) {
			loggerLead.Errorf("viewchange block not match")
			loggerLead.Errorf("my vcblock is %+v", cl.role.GetCurrentVCBlock())
			loggerLead.Errorf("msg vcblock is %+v", vcblock)
			return ErrVerifyBlock
		}
		err = cl.peerServer.MsgVerify(msg, cl.role.GetCurrentVCBlock().Hash().Bytes())
		if err != nil {
			loggerLead.Errorf("message signature verify failed with error: %s", err.Error())
			return ErrVerifyMessage
		}
		vcblock.Header.Signature = multiSig
		tempSign.Deserialize(multiSig)
		ret = tempSign.BLSVerify(tempPubKey, vcblock.Hash().Bytes())
	}

	if ret {
		loggerLead.Debugf("bls signature verify passed")
		cl.signMap[tempPubKey.GetHexString()] = tempSign
		cl.pubKeyMap[tempPubKey.GetHexString()] = tempPubKey
		cl.publicKey2MsgMap[tempPubKey.GetHexString()] = true
	} else {
		loggerLead.Errorf("bls signature verify failed")
		return ErrVerifyMessage
	}
	return nil
}

func (cl *ConsensusLead) generateBoolMap(consensusType pb.ConsensusType) error {
	var boolMap []bool

	if consensusType == pb.ConsensusType_FinalBlockConsensus || consensusType == pb.ConsensusType_DsBlockConsensus ||
		(consensusType == pb.ConsensusType_ViewChangeConsensus && cl.role.GetCurrentVCBlock().Header.Stage != "MicroBlockConsensus") {
		var committeeSort pb.PeerEndpointList
		for i := 0; i < cl.peerServer.Committee.Len(); i++ {
			committeeSort = append(committeeSort, cl.peerServer.Committee[i])
		}
		sort.Sort(pb.ByPubkey{PeerEndpointList: committeeSort})
		for i := 0; i < len(committeeSort); i++ {
			pubKeyStr, err := cl.peerServer.MsgSinger.PubKeyBytes2String(committeeSort[i].Pubkey)
			if err != nil {
				return err
			}
			if cl.publicKey2MsgMap[pubKeyStr] {
				boolMap = append(boolMap, true)
				finalBLSSign.Add(&cl.signMap[pubKeyStr].Sign)
				GPubkey.Add(&cl.pubKeyMap[pubKeyStr].PublicKey)
			} else {
				boolMap = append(boolMap, false)
			}
		}
	}

	if consensusType == pb.ConsensusType_MicroBlockConsensus ||
		(consensusType == pb.ConsensusType_ViewChangeConsensus && cl.role.GetCurrentVCBlock().Header.Stage == "MicroBlockConsensus") {
		for i := 0; i < len(cl.peerServer.ShardingNodes); i++ {
			pubKeyStr, err := cl.peerServer.MsgSinger.PubKeyBytes2String(cl.peerServer.ShardingNodes[i].Pubkey)
			if err != nil {
				return err
			}
			if cl.publicKey2MsgMap[pubKeyStr] {
				boolMap = append(boolMap, true)
				finalBLSSign.Add(&cl.signMap[pubKeyStr].Sign)
				GPubkey.Add(&cl.pubKeyMap[pubKeyStr].PublicKey)
			} else {
				boolMap = append(boolMap, false)
			}

		}

	}

	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		cl.role.GetCurrentDSBlock().Header.BoolMap = boolMap
		cl.role.GetCurrentDSBlock().Header.Signature = finalBLSSign.Serialize()
		cl.role.GetCurrentDSBlock().Header.MultiPubKey = GPubkey.Serialize()
	case pb.ConsensusType_FinalBlockConsensus:
		cl.role.GetCurrentFinalBlock().Header.BoolMap = boolMap
		cl.role.GetCurrentFinalBlock().Header.Signature = finalBLSSign.Serialize()
		cl.role.GetCurrentFinalBlock().Header.MultiPubKey = GPubkey.Serialize()
	case pb.ConsensusType_MicroBlockConsensus:
		cl.role.GetCurrentMicroBlock().Header.BoolMap = boolMap
		cl.role.GetCurrentMicroBlock().Header.Signature = finalBLSSign.Serialize()
		cl.role.GetCurrentMicroBlock().Header.MultiPubKey = GPubkey.Serialize()
	case pb.ConsensusType_ViewChangeConsensus:
		cl.role.GetCurrentVCBlock().Header.BoolMap = boolMap
		cl.role.GetCurrentVCBlock().Header.Signature = finalBLSSign.Serialize()
		cl.role.GetCurrentVCBlock().Header.MultiPubKey = GPubkey.Serialize()
	}

	var zeroBLSSign multibls.Sig
	var zeroPubkey multibls.PubKey
	finalBLSSign = zeroBLSSign
	GPubkey = zeroPubkey

	return nil
}

func (cl *ConsensusLead) composeFinalCollectiveSig(consensusType pb.ConsensusType) (*pb.Message, error) {
	finalCollectiveSig := &pb.Message{Type: pb.Message_Consensus_FinalCollectiveSig}
	finalCollectiveSig.Timestamp = pb.CreateUtcTimestamp()
	payload := &pb.ConsensusPayload{}
	payload.Type = consensusType
	var err error
	var block []byte
	var sig []byte
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		loggerLead.Debugf("dsblock is %+v", cl.role.GetCurrentDSBlock())
		block, err = proto.Marshal(cl.role.GetCurrentDSBlock())
		sig, err = cl.peerServer.MsgSinger.SignHash(cl.role.GetCurrentDSBlock().Hash().Bytes(), nil)
	case pb.ConsensusType_FinalBlockConsensus:
		stateRoot, gasUsed, err := cl.peerServer.CalStateRoot(cl.role.GetCurrentFinalBlock())
		if err != nil {
			return nil, err
		}
		cl.role.GetCurrentFinalBlock().Header.StateRoot = stateRoot
		cl.role.GetCurrentFinalBlock().Header.GasUsed = gasUsed
		loggerLead.Debugf("txblock is %+v", cl.role.GetCurrentFinalBlock())
		block, err = proto.Marshal(cl.role.GetCurrentFinalBlock())
		sig, err = cl.peerServer.MsgSinger.SignHash(cl.role.GetCurrentFinalBlock().Hash().Bytes(), nil)
	case pb.ConsensusType_MicroBlockConsensus:
		loggerLead.Debugf("microblock is %+v", cl.role.GetCurrentMicroBlock())
		block, err = proto.Marshal(cl.role.GetCurrentMicroBlock())
		sig, err = cl.peerServer.MsgSinger.SignHash(cl.role.GetCurrentMicroBlock().Hash().Bytes(), nil)
	case pb.ConsensusType_ViewChangeConsensus:
		loggerLead.Debugf("vcblock is %+v", cl.role.GetCurrentVCBlock())
		block, err = proto.Marshal(cl.role.GetCurrentVCBlock())
		sig, err = cl.peerServer.MsgSinger.SignHash(cl.role.GetCurrentVCBlock().Hash().Bytes(), nil)
	}

	if err != nil {
		loggerLead.Errorf("block signature verify failed with error: %s", err.Error())
		return nil, err
	}

	payload.Msg = block
	data, err := proto.Marshal(payload)
	if err != nil {
		loggerLead.Errorf("block marshal failed with error: %s", err.Error())
		return nil, err
	}
	finalCollectiveSig.Signature = sig
	finalCollectiveSig.Payload = data
	finalCollectiveSig.Peer = cl.peerServer.SelfNode
	return finalCollectiveSig, nil
}
