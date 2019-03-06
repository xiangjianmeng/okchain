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
	"bytes"
	"context"
	"encoding/hex"
	"reflect"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/core/blockchain"
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/crypto/multibls"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
)

type RoleDsBackup struct {
	*RoleDsBase
	dsConsensusBackup ps.IConsensusBackup
}

var loggerDsBackup = logging.MustGetLogger("dsBackupRole")

func newRoleDsBackup(peer *ps.PeerServer) ps.IRole {
	dsBase := newRoleDsBase(peer)
	r := &RoleDsBackup{RoleDsBase: dsBase}
	r.initBase(r)
	r.name = "DsBackup"
	r.dsConsensusBackup = peer.ProduceConsensusBackup(peer)
	r.dsConsensusBackup.SetIRole(r)
	return r
}

func (r *RoleDsBackup) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	err := r.dsConsensusBackup.ProcessConsensusMsg(msg, from, r)
	if err != nil {
		loggerDsBackup.Errorf("handle consensus message failed with error: %s", err.Error())
		return err
	}
	return nil
}

func (r *RoleDsBackup) onWait4PoWSubmissionDone() error {
	if r.peerServer.ConsensusData.PoWSubList.Len() == 0 {
		ctx, cancle := context.WithTimeout(context.Background(), time.Duration(r.peerServer.GetWait4PoWTime())*time.Second)
		go r.Wait4PoWSubmission(ctx, cancle)
		return nil
	}
	r.ChangeState(ps.STATE_DSBLOCK_CONSENSUS)
	r.dsConsensusBackup.SetCurrentConsensusStage(pb.ConsensusType_DsBlockConsensus)
	loggerDsBackup.Debugf("wait for POW_SUBMISSION finished")

	go r.dsConsensusBackup.WaitForViewChange()
	dsblock, err := r.composeDSBlock()
	if err != nil {
		return ErrComposeBlock
	}
	r.SetCurrentDSBlock(dsblock)
	return nil
}

func (r *RoleDsBackup) onWait4MicroBlockSubmissionDone() error {
	r.ChangeState(ps.STATE_FINALBLOCK_CONSENSUS)
	r.dsConsensusBackup.SetCurrentConsensusStage(pb.ConsensusType_FinalBlockConsensus)
	loggerDsBackup.Debugf("wait for MICROBLOCK_SUBMISSION finished")

	go r.dsConsensusBackup.WaitForViewChange()
	// viewchange may change leader, so update leader each time
	r.dsConsensusBackup.UpdateLeader(r.peerServer.Committee[0])
	txblock, err := r.composeFinalBlock()
	if err != nil {
		return ErrComposeBlock
	}

	ptocoin, ret := r.peerServer.DsBlockChain().(*blockchain.DSBlockChain).GetPubkey2CoinbaseCache().GetCoinbaseByPubkey(r.peerServer.Committee[0].Pubkey)
	if !ret {
		loggerDsBackup.Errorf("get coinbase failed, pubkey is %+v", hex.EncodeToString(r.peerServer.Committee[0].Pubkey))
		return ErrGetCoinbase
	}
	txblock.Header.DSCoinBase = append([][]byte{}, ptocoin)
	r.SetCurrentFinalBlock(txblock)
	return nil
}

// DsBlock Consensus 结束后被 Consensus backup engine 回调
func (r *RoleDsBackup) onDsBlockConsensusCompleted(err error) error {

	err = r.onDsBlockReady(r.GetCurrentDSBlock())
	if err != nil {
		return err
	}
	// clear powsublist
	r.peerServer.ConsensusData.PoWSubList.Clear()

	if r.peerServer.ShardingNodes.Has(r.peerServer.SelfNode) {
		// change to a sharding node
		r.initShardingNodeInfo(r.GetCurrentDSBlock())

		//r.peerServer.ShardingNodes.Dump()

		nextRole := ps.PeerRole_ShardingBackup
		if reflect.DeepEqual(r.peerServer.ShardingNodes[0].Pubkey, r.peerServer.SelfNode.Pubkey) {
			loggerDsBackup.Infof("I am this round ShardingLead")
			nextRole = ps.PeerRole_ShardingLead
		} else {
			loggerDsBackup.Infof("I am this round ShardingBackup")
		}
		newRole := r.peerServer.ChangeRole(nextRole, ps.STATE_MICROBLOCK_CONSENSUS)

		// use new role to call consensus start function
		err = newRole.OnMircoBlockConsensusStarted(r.peerServer.ShardingNodes[0])
	} else {
		r.ChangeState(ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
	}

	if err != nil {
		return err
	}
	r.DumpCurrentState()
	return nil
}

func (r *RoleDsBackup) onFinalBlockConsensusCompleted(err error) error {

	err = r.onFinalBlockReady(r.GetCurrentFinalBlock())
	if err != nil {
		return err
	}
	// clear microblock list
	r.peerServer.ConsensusData.MicroBlockList.Clear()

	numOfSharding := len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes) / r.peerServer.GetShardingSize()

	if (numOfSharding != 0 && r.GetCurrentFinalBlock().Header.BlockNumber%uint64(r.peerServer.GetShardingSize()) == 0) ||
		(numOfSharding == 0 && r.GetCurrentFinalBlock().Header.BlockNumber%uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)) == 0) {
		r.ChangeState(ps.STATE_WAIT4_POW_SUBMISSION)
		r.peerServer.ShardingNodes = []*pb.PeerEndpoint{}
		ctx, cancle := context.WithTimeout(context.Background(), time.Duration(r.peerServer.GetWait4PoWTime())*time.Second)
		go r.Wait4PoWSubmission(ctx, cancle)
	} else {
		r.ChangeState(ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
	}

	r.DumpCurrentState()
	return nil
}

func (r *RoleDsBackup) onViewChangeConsensusCompleted(err error) error {
	go r.dsConsensusBackup.WaitForViewChange()
	switch r.GetCurrentVCBlock().Header.Stage {
	case "DsBlockConsensus":
		r.dsConsensusBackup.SetCurrentConsensusStage(pb.ConsensusType_DsBlockConsensus)
		r.ChangeState(ps.STATE_DSBLOCK_CONSENSUS)
		r.DumpCurrentState()
	case "FinalBlockConsensus":
		r.dsConsensusBackup.SetCurrentConsensusStage(pb.ConsensusType_FinalBlockConsensus)
		r.ChangeState(ps.STATE_FINALBLOCK_CONSENSUS)
		r.DumpCurrentState()
	default:
		return ErrHandleInCurrentState
	}
	return nil
}

func (r *RoleDsBackup) verifyFinalBlock(msg *pb.Message, from *pb.PeerEndpoint) error {
	announce := &pb.ConsensusPayload{}
	err := proto.Unmarshal(msg.Payload, announce)
	if err != nil {
		loggerDsBackup.Errorf("unmarshal consensus payload error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	txblock := &pb.TxBlock{}
	err = proto.Unmarshal(announce.Msg, txblock)
	if err != nil {
		loggerDsBackup.Errorf("unmarshal txblock failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	myhash := []string{}
	for i := 0; i < len(r.GetCurrentFinalBlock().Body.MicroBlockHashes); i++ {
		myhash = append(myhash, hex.EncodeToString(r.GetCurrentFinalBlock().Body.MicroBlockHashes[i]))
	}

	msghash := []string{}
	for i := 0; i < len(txblock.Body.MicroBlockHashes); i++ {
		msghash = append(msghash, hex.EncodeToString(txblock.Body.MicroBlockHashes[i]))
	}

	sort.Strings(myhash)
	sort.Strings(msghash)

	if !reflect.DeepEqual(myhash, msghash) {
		loggerDsBackup.Errorf("myhash is %+v", myhash)
		loggerDsBackup.Errorf("msghash is %+v", msghash)
		loggerDsBackup.Errorf("micro block hashes in finalblock are not equal")
		return ErrVerifyBlock
	}

	mycoinbase := []string{}
	for i := 0; i < len(r.GetCurrentFinalBlock().Header.ShardingLeadCoinBase); i++ {
		mycoinbase = append(mycoinbase, hex.EncodeToString(r.GetCurrentFinalBlock().Header.ShardingLeadCoinBase[i]))
	}

	msgcoinbase := []string{}
	for i := 0; i < len(txblock.Header.ShardingLeadCoinBase); i++ {
		msgcoinbase = append(msgcoinbase, hex.EncodeToString(txblock.Header.ShardingLeadCoinBase[i]))
	}

	sort.Strings(mycoinbase)
	sort.Strings(msgcoinbase)

	if !reflect.DeepEqual(mycoinbase, msgcoinbase) {
		loggerDsBackup.Errorf("mycoinbase is %+v", mycoinbase)
		loggerDsBackup.Errorf("msgcoinbase is %+v", msgcoinbase)
		loggerDsBackup.Errorf("micro block coinbases in finalblock are not equal")
		return ErrVerifyBlock
	}

	if txblock.Body.NumberOfMicroBlock != r.GetCurrentFinalBlock().Body.NumberOfMicroBlock {
		loggerDsBackup.Errorf("expected microblock number is %d, actual is %d", r.GetCurrentFinalBlock().Body.NumberOfMicroBlock, txblock.Body.NumberOfMicroBlock)
		loggerDsBackup.Errorf("number of micro block is not equal")
		return ErrVerifyBlock
	}

	if txblock.Header.TxNum != r.GetCurrentFinalBlock().Header.TxNum {
		loggerDsBackup.Errorf("expected tx number is %d, actual is %d", r.GetCurrentFinalBlock().Header.TxNum, txblock.Header.TxNum)
		loggerDsBackup.Errorf("transactions number is not equal")
		return ErrVerifyBlock
	}

	if txblock.Header.DSBlockNum != r.GetCurrentFinalBlock().Header.DSBlockNum {
		loggerDsBackup.Errorf("expected dsblock number is %d, actual is %d", r.GetCurrentFinalBlock().Header.DSBlockNum, txblock.Header.DSBlockNum)
		loggerDsBackup.Errorf("dsBlock number is not equal")
		return ErrVerifyBlock
	}

	if !bytes.Equal(txblock.Header.PreviousBlockHash, r.GetCurrentFinalBlock().Header.PreviousBlockHash) {
		loggerDsBackup.Errorf("expected block hash is %d, actual is %d", r.GetCurrentFinalBlock().Header.PreviousBlockHash, txblock.Header.PreviousBlockHash)
		loggerDsBackup.Errorf("previous block hash is not equal")
		return ErrVerifyBlock
	}

	//if !bytes.Equal(txblock.Header.StateRoot, r.GetCurrentFinalBlock().Header.StateRoot) {
	//	loggerDsBackup.Errorf("state root is not equal")
	//	return errors.New("state root is not equal")
	//}

	if !pb.VerifyMerkelRoot(txblock.Body.Transactions, txblock.Header.TxRoot) {
		loggerDsBackup.Errorf("final block root is not equal")
		return ErrVerifyBlock
	}

	r.SetCurrentFinalBlock(txblock)

	return nil
}

func (r *RoleDsBackup) verifyDSBlock(msg *pb.Message, from *pb.PeerEndpoint) error {
	announce := &pb.ConsensusPayload{}
	err := proto.Unmarshal(msg.Payload, announce)
	if err != nil {
		loggerDsBackup.Errorf("unmarshal consensus payload failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	dsblock := &pb.DSBlock{}
	err = proto.Unmarshal(announce.Msg, dsblock)
	if err != nil {
		loggerDsBackup.Errorf("unmarshal dsblock failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	if r.peerServer.ConsensusData.PoWSubList.Len() == 0 {
		loggerDsBackup.Errorf("current PoWsublist len is 0, cannot handle announce message")
		return ErrEmptyPoWList
	}

	if !bytes.Equal(r.peerServer.ConsensusData.PoWSubList.Get(0).PublicKey, dsblock.Header.WinnerPubKey) {
		loggerDsBackup.Errorf("my winner pubkey is %s", hex.EncodeToString(r.peerServer.ConsensusData.PoWSubList.Get(0).PublicKey))
		loggerDsBackup.Errorf("msg winner pubkey is %s", hex.EncodeToString(dsblock.Header.WinnerPubKey))
		loggerDsBackup.Errorf("winner public key cannot match")
		return ErrVerifyBlock
	}

	if r.peerServer.ConsensusData.PoWSubList.Get(0).Nonce != dsblock.Header.WinnerNonce {
		loggerDsBackup.Errorf("my winner nonce is %d", r.peerServer.ConsensusData.PoWSubList.Get(0).Nonce)
		loggerDsBackup.Errorf("msg winner nonce is %d", dsblock.Header.WinnerNonce)
		loggerDsBackup.Errorf("winner nonce cannot match")
		return ErrVerifyBlock
	}

	if r.peerServer.DsBlockChain().CurrentBlock().NumberU64()+1 != dsblock.Header.BlockNumber {
		loggerDsBackup.Errorf("expected block number is %d, actual is %d", r.peerServer.DsBlockChain().CurrentBlock().NumberU64()+1, dsblock.Header.BlockNumber)
		loggerDsBackup.Errorf("block number cannot match")
		return ErrVerifyBlock
	}

	if r.peerServer.ConsensusData.PoWSubList.Get(0).Difficulty != dsblock.Header.PowDifficulty {
		loggerDsBackup.Errorf("pow difficulty cannot match")
		return ErrVerifyBlock
	}

	if !bytes.Equal(dsblock.Header.PreviousBlockHash, r.peerServer.DsBlockChain().CurrentBlock().Hash().Bytes()) {
		loggerDsBackup.Errorf("expected dsblock previous hash is %+v, actual id %+v", r.peerServer.DsBlockChain().CurrentBlock().Hash().Hex(), hex.EncodeToString(dsblock.Header.PreviousBlockHash))
		loggerDsBackup.Errorf("previous block hash cannot match")
		return ErrVerifyBlock
	}

	shardingSize := r.peerServer.GetShardingSize()
	numOfSharding := r.peerServer.ConsensusData.PoWSubList.Len() / shardingSize
	shardingNodesTotal := numOfSharding * shardingSize
	if numOfSharding == 0 {
		if dsblock.Header.ShardingSum != 1 {
			loggerDsBackup.Errorf("sharding number cannot match")
			return ErrVerifyBlock
		}
	} else {
		if uint32(numOfSharding) != dsblock.Header.ShardingSum {
			loggerDsBackup.Errorf("sharding number cannot match,should %d,but %d", numOfSharding, dsblock.Header.ShardingSum)
			return ErrVerifyBlock
		}
	}

	if dsblock.Body.ShardingNodes[0].Id.Name != r.peerServer.Committee[len(r.peerServer.Committee)-1].Id.Name {
		loggerDsBackup.Errorf("the committee backup node kicked off is not match")
		return ErrVerifyBlock
	}

	if shardingNodesTotal != 0 {
		for i := 1; i < shardingNodesTotal; i++ {
			if r.peerServer.ConsensusData.PoWSubList.Get(i).Peer.Id.Name != dsblock.Body.ShardingNodes[i].Id.Name {
				loggerDsBackup.Errorf("sharding node is not match. locate %d, expect is %s, actual is %s",
					i, r.peerServer.ConsensusData.PoWSubList.Get(i).Peer.Id.Name, dsblock.Body.ShardingNodes[i].Id.Name)
				return ErrVerifyBlock
			}
		}
	} else {
		for i := 1; i < r.peerServer.ConsensusData.PoWSubList.Len(); i++ {
			if r.peerServer.ConsensusData.PoWSubList.Get(i).Peer.Id.Name != dsblock.Body.ShardingNodes[i].Id.Name {
				loggerDsBackup.Errorf("sharding node is not match. locate %d, expect is %s, actual is %s",
					i, r.peerServer.ConsensusData.PoWSubList.Get(i).Peer.Id.Name, dsblock.Body.ShardingNodes[i].Id.Name)
				return ErrVerifyBlock
			}
		}
	}

	if !reflect.DeepEqual(r.peerServer.ConsensusData.PoWSubList.Get(0).Peer, dsblock.Header.NewLeader) {
		loggerDsBackup.Errorf("expected new leader is %+v, actual is %+v", r.peerServer.ConsensusData.PoWSubList.Get(0).Peer, dsblock.Header.NewLeader)
		loggerDsBackup.Errorf("committee new leader cannot match")
		return ErrVerifyBlock
	}

	//var pubKeyToCoinBaseMap map[string][]byte
	//pubKeyToCoinBaseMap = make(map[string][]byte)

	// todo, set in genesis dsblock
	//if dsblock.Header.BlockNumber == 1 {
	//	for _, v := range r.peerServer.Committee {
	//		pkStr, err := r.peerServer.MsgSinger.PubKeyBytes2String(v.Pubkey)
	//		if err != nil {
	//			roleDsBaseLogger.Errorf("pubkey to string error: %s", err.Error())
	//			continue
	//		}
	//		pubKeyToCoinBaseMap[pkStr] = []byte("coinbase:" + v.Id.Name)
	//	}
	//}
	//
	//for i := 0; i < r.peerServer.ConsensusData.PoWSubList.Len(); i++ {
	//	pkStr, err := r.peerServer.MsgSinger.PubKeyBytes2String(r.peerServer.ConsensusData.PoWSubList.Get(i).PublicKey)
	//	if err != nil {
	//		continue
	//	}
	//	if _, ok := r.peerServer.ConsensusData.PubKeyToCoinBaseMap[pkStr]; ok {
	//		if !reflect.DeepEqual(r.peerServer.ConsensusData.PubKeyToCoinBaseMap[pkStr], r.peerServer.ConsensusData.PoWSubList.Get(i).Coinbase) {
	//			pubKeyToCoinBaseMap[pkStr] = r.peerServer.ConsensusData.PoWSubList.Get(i).Coinbase
	//		}
	//	} else {
	//		pubKeyToCoinBaseMap[pkStr] = r.peerServer.ConsensusData.PoWSubList.Get(i).Coinbase
	//	}
	//}
	//
	//if len(pubKeyToCoinBaseMap) == 0 {
	//	pubKeyToCoinBaseMap = nil
	//}
	//
	//if !reflect.DeepEqual(pubKeyToCoinBaseMap, dsblock.Body.PubKeyToCoinBaseMap) {
	//	loggerDsBackup.Errorf("my pubkey to coinbase map is %+v", pubKeyToCoinBaseMap)
	//	loggerDsBackup.Errorf("dsblock pubkey to coinbase map is %+v", dsblock.Body.PubKeyToCoinBaseMap)
	//	loggerDsBackup.Errorf("pubkey to coinbase map cannot match")
	//	return ErrVerifyBlock
	//}

	err = r.peerServer.MsgVerify(msg, dsblock.Hash().Bytes())
	if err != nil {
		loggerDsBackup.Errorf("bls message signature check failed with error: %s", err.Error())
		return ErrVerifyMessage
	}

	r.SetCurrentDSBlock(dsblock)

	return nil
}

func (r *RoleDsBackup) preConsensusProcessDsBlock(block proto.Message, response *pb.Message, payload *pb.ConsensusPayload) error {
	sign := r.peerServer.PrivateKey.BlSSign(r.GetCurrentDSBlock().MHash().Bytes())
	r.GetCurrentDSBlock().Header.Signature = sign.Serialize()

	sig, err := r.peerServer.MsgSinger.SignHash(r.GetCurrentDSBlock().Hash().Bytes(), nil)
	if err != nil {
		loggerDsBackup.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}
	response.Signature = sig

	data, err := r.produceConsensusPayload(r.GetCurrentDSBlock(), pb.ConsensusType_DsBlockConsensus)
	if err != nil {
		return nil
	}
	response.Payload = data
	return nil
}

func (r *RoleDsBackup) preConsensusProcessFinalBlock(block proto.Message, response *pb.Message, payload *pb.ConsensusPayload) error {
	sign := r.peerServer.PrivateKey.BlSSign(r.GetCurrentFinalBlock().MHash().Bytes())
	r.GetCurrentFinalBlock().Header.Signature = sign.Serialize()
	loggerDsBackup.Debugf("final block is %+v", r.GetCurrentFinalBlock())

	pub := &multibls.PubKey{}
	pub.Deserialize(r.peerServer.PublicKey)

	ret := sign.BLSVerify(pub, r.GetCurrentFinalBlock().MHash().Bytes())
	if !ret {
		loggerDsBackup.Errorf("bls verify failed")
	}

	r.GetCurrentFinalBlock().Header.DSCoinBase = append(r.GetCurrentFinalBlock().Header.DSCoinBase, r.peerServer.CoinBase)
	loggerDsBackup.Debugf("final block is %+v", r.GetCurrentFinalBlock())
	sig, err := r.peerServer.MsgSinger.SignHash(r.GetCurrentFinalBlock().Hash().Bytes(), nil)
	if err != nil {
		loggerDsBackup.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}
	response.Signature = sig

	data, err := r.produceConsensusPayload(r.GetCurrentFinalBlock(), pb.ConsensusType_FinalBlockConsensus)
	if err != nil {
		return err
	}
	response.Payload = data
	return nil
}

func (r *RoleDsBackup) preConsensusProcessVCBlock(block proto.Message, response *pb.Message, payload *pb.ConsensusPayload) error {
	sign := r.peerServer.PrivateKey.BlSSign(r.GetCurrentVCBlock().Hash().Bytes())
	r.GetCurrentVCBlock().Header.Signature = sign.Serialize()

	sig, err := r.peerServer.MsgSinger.SignHash(r.GetCurrentVCBlock().Hash().Bytes(), nil)
	if err != nil {
		loggerDsBackup.Errorf("bls message sign failed with error: %s", err.Error())
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

func (r *RoleDsBackup) VerifyBlock(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error {
	var err error
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		err = r.verifyDSBlock(msg, from)
	case pb.ConsensusType_FinalBlockConsensus:
		err = r.verifyFinalBlock(msg, from)
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
		return ErrVerifyBlock
	}

	return nil
}

func (r *RoleDsBackup) getConsensusData(consensusType pb.ConsensusType) (proto.Message, error) {
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		return r.GetCurrentDSBlock(), nil
	case pb.ConsensusType_FinalBlockConsensus:
		return r.GetCurrentFinalBlock(), nil
	case pb.ConsensusType_ViewChangeConsensus:
		return r.GetCurrentVCBlock(), nil
	default:
		return nil, ErrHandleInCurrentState
	}
}

func (r *RoleDsBackup) ComposeFinalResponse(consensusType pb.ConsensusType) (*pb.Message, error) {
	response, err := r.produceConsensusMessage(consensusType, pb.Message_Consensus_FinalResponse)

	if err != nil {
		loggerDsBackup.Errorf("compose consensus message failed with error : %s", err.Error())
		return nil, ErrComposeMessage
	}

	return response, nil
}

func (r *RoleDsBackup) produceFinalResponse(block proto.Message, envelope *pb.Message, payload *pb.ConsensusPayload, consensusType pb.ConsensusType) (*pb.Message, error) {
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

func (r *RoleDsBackup) OnViewChangeConsensusStarted() error {
	go r.dsConsensusBackup.WaitForViewChange()
	r.dsConsensusBackup.SetCurrentConsensusStage(pb.ConsensusType_ViewChangeConsensus)
	// update leader when consensus start
	r.dsConsensusBackup.UpdateLeader(r.GetNewLeader(r.peerServer.ConsensusData.CurrentStage, r.peerServer.ConsensusData.LastStage))
	r.peerServer.Committee = append(r.peerServer.Committee[1:len(r.peerServer.Committee)], r.peerServer.Committee[0])
	loggerDsBackup.Debugf("current committee is %+v", r.peerServer.Committee)
	vcblock, err := r.composeVCBlock()
	if err != nil {
		return ErrComposeBlock
	}
	r.SetCurrentVCBlock(vcblock)
	return nil
}
