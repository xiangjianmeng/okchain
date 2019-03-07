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
	"errors"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"sort"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/consensus/mine"
	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

const (
	GasLimitBoundDivisor uint64 = 1024 // The bound divisor of the gas limit, used in update calculations.
	MinGasLimit          uint64 = 5000 // Minimum the gas limit may ever be.
	MinerGasFloor        uint64 = 8000000
	MinerGasCeil         uint64 = 8000000
)

var roleDsBaseLogger = logging.MustGetLogger("roleDsBase")

type RoleDsBase struct {
	*RoleBase
	cancelFunc context.CancelFunc
}

func newRoleDsBase(peer *ps.PeerServer) *RoleDsBase {
	base := &RoleBase{peerServer: peer}
	r := &RoleDsBase{RoleBase: base}
	r.cancelFunc = nil
	return r
}

// callback function when consensus finished
func (r *RoleDsBase) OnConsensusCompleted(err error) error {
	stateType := reflect.TypeOf(r.state).String()
	loggerDsBackup.Debugf("Invoked: state<%s>", stateType)

	// notify channel to interrupt viewchange timer
	r.peerServer.VcChan <- "finished"
	loggerDsBackup.Debugf("send message to channel to stop viewchange")
	// todo: use state machine to handle OnConsensusCompleted in each state

	switch stateType {
	case "*state.STATE_DSBLOCK_CONSENSUS":
		loggerDsBackup.Debugf("dsblock consensus completed")
		err = r.imp.onDsBlockConsensusCompleted(err)
	case "*state.STATE_FINALBLOCK_CONSENSUS":
		loggerDsBackup.Debugf("final block consensus completed")
		err = r.imp.onFinalBlockConsensusCompleted(err)
	case "*state.STATE_VIEWCHANGE_CONSENSUS":
		loggerDsBackup.Debugf("viewchange consensus completed")
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

func (r *RoleDsBase) ProcessPoWSubmission(msg *pb.Message, from *pb.PeerEndpoint) error {
	roleDsBaseLogger.Debugf("[%s]: RoleDsBase ProcessPoWSubmission", util.GId)
	pow := &pb.PoWSubmission{}

	err := proto.Unmarshal(msg.Payload, pow)
	if err != nil {
		roleDsBaseLogger.Errorf("unmarshal pow submission message error: %s", err.Error())
		return ErrUnmarshalMessage
	}
	roleDsBaseLogger.Debugf("pow message from %s", pow.Peer.Id)

	if r.peerServer.ConsensusData.PoWSubList.Has(pow.Peer) {
		roleDsBaseLogger.Errorf("received a duplicate pow submission message")
		return ErrDuplicatedMessage
	}

	if pow.Rand1 != r.updateDSBlockRand() {
		roleDsBaseLogger.Errorf("DSBlockRand number is not as expected")
		return ErrVerifyBlock
	}

	if pow.Rand2 != r.updateTXBlockRand() {
		roleDsBaseLogger.Errorf("TXBlockRand number is not as expected")
		return ErrVerifyBlock
	}

	err = r.peerServer.MsgVerify(msg, pow.Hash().Bytes())
	if err != nil {
		roleDsBaseLogger.Errorf("bls message signature check error: %s", err.Error())
		return ErrVerifyMessage
	}

	pubKeyStr, err := r.peerServer.MsgSinger.PubKeyBytes2String(pow.Peer.Pubkey)
	if err != nil {
		roleDsBaseLogger.Errorf("bls public key bytes to string error: %s", err.Error())
		return ErrPubKeyConvert
	}

	verifier, _ := mine.NewVerifier(config.GetMinerType(), pubKeyStr)
	difficulty := big.NewInt(r.peerServer.GetPowDifficulty())
	seed := r.updateDSBlockRand() + r.updateTXBlockRand() + pow.Peer.Address + pubKeyStr + hex.EncodeToString(pow.Coinbase)
	ret, err := verifier.Verify([]byte(seed), pow.BlockNum, difficulty, pow.Result)

	if err != nil {
		roleDsBaseLogger.Errorf("pow result verify failed with error: %s", err.Error())
		return ErrVerifyMine
	}

	roleDsBaseLogger.Debugf("[%s] PoW result verify %+v", util.GId, ret)
	roleDsBaseLogger.Debugf("pow submission message is %+v", pow)

	r.peerServer.ConsensusData.PoWSubList.Append(pow)

	r.peerServer.ShardingNodes = append(r.peerServer.ShardingNodes, msg.Peer)

	roleDsBaseLogger.Debugf("add <%s> powsubmission message to list, current list len is %d",
		pow.Peer.Id, r.peerServer.ConsensusData.PoWSubList.Len())

	return nil
}

func (r *RoleDsBase) Wait4PoWSubmission(ctx context.Context, cancle context.CancelFunc) {
	r.cancelFunc = cancle
	roleDsBaseLogger.Debugf("wait %ds to collect PoW submission", r.peerServer.GetWait4PoWTime())

	for {
		select {
		case <-ctx.Done():
			r.peerServer.ConsensusData.PoWSubList.Sort()
			r.peerServer.ConsensusData.PoWSubList.Dump()
			r.imp.onWait4PoWSubmissionDone()
			return
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (r *RoleDsBase) ProcessMicroBlockSubmission(msg *pb.Message, from *pb.PeerEndpoint) error {
	microBlock := &pb.MicroBlock{}
	err := proto.Unmarshal(msg.Payload, microBlock)
	if err != nil {
		roleDsBaseLogger.Debugf("unmarshal micro block submission message error: %s", err.Error())
		return ErrUnmarshalMessage
	}
	roleDsBaseLogger.Debugf("received micro block is: %+v", microBlock)
	if microBlock.Header.DSBlockNum != r.peerServer.DsBlockChain().CurrentBlock().NumberU64() {
		roleDsBaseLogger.Debugf("dsblock number is not as expect")
		return ErrVerifyBlock
	}
	if !bytes.Equal(microBlock.Header.DSBlockHash, r.peerServer.DsBlockChain().CurrentBlock().Hash().Bytes()) {
		roleDsBaseLogger.Debugf("dsblock hash is not equal")
		return ErrVerifyBlock
	}
	err = r.peerServer.MsgVerify(msg, microBlock.Hash().Bytes())
	if err != nil {
		roleDsBaseLogger.Debugf("bls message verify failed with error: %s", err.Error())
		return ErrVerifyMessage
	}

	sort.Sort(pb.ByPubkey{PeerEndpointList: r.peerServer.ShardingMap[microBlock.ShardID]})
	if len(microBlock.Header.BoolMap) != len(r.peerServer.ShardingMap[microBlock.ShardID]) {
		roleDsBaseLogger.Errorf("micro block boolmap and sharding nodes cannot match")
		roleDsBaseLogger.Errorf("micro block boolmap: %+v", microBlock.Header.BoolMap)
		roleDsBaseLogger.Errorf("sharding nodes: %+v", r.peerServer.ShardingMap[microBlock.ShardID])
		return ErrVerifyBlock
	}

	counter := 0
	expected := 0
	for i := 0; i < len(microBlock.Header.BoolMap); i++ {
		if microBlock.Header.BoolMap[i] {
			counter++
		}
	}

	if r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum == 1 {
		expected = int(math.Floor(float64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes))*ToleranceFraction)) + 1
	} else {
		expected = int(math.Floor(float64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)/
			int(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum))*ToleranceFraction)) + 1
	}

	if counter < expected {
		roleDsBaseLogger.Errorf("multi signer less than expected number, expected %d, actual %d", expected, counter)
		return ErrVerifyBlock
	}

	err = pb.BlsMultiPubkeyVerify(microBlock.Header.BoolMap, r.peerServer.ShardingMap[microBlock.ShardID], microBlock.Header.MultiPubKey)
	if err != nil {
		roleDsBaseLogger.Debugf("bls multi public key verify failed with error: %s", err.Error())
		return ErrBLSPubKeyVerify
	}

	roleDsBaseLogger.Debugf("microblock submission message check pass")
	if r.peerServer.ConsensusData.MicroBlockList.Len() > 0 {
		if r.peerServer.ConsensusData.MicroBlockList.Has(msg.Peer) {
			loggerDsBackup.Errorf("dumplicate micro block submission message")
			return ErrDuplicatedMessage
		}
	}

	r.peerServer.ConsensusData.MicroBlockList.Append(microBlock)

	if uint32(r.peerServer.ConsensusData.MicroBlockList.Len()) == r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Header.ShardingSum {
		err = r.imp.onWait4MicroBlockSubmissionDone()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RoleDsBase) composeDSBlock() (*pb.DSBlock, error) {
	dsblock := &pb.DSBlock{}
	dsblock.Header = &pb.DSBlockHeader{}
	dsblock.Body = &pb.DSBlockBody{}

	pow := r.peerServer.ConsensusData.PoWSubList.Get(0)

	if pow != nil {
		dsblock.Header.Version = 1
		dsblock.Header.BlockNumber = r.peerServer.DsBlockChain().CurrentBlock().NumberU64() + 1
		dsblock.Header.Timestamp = pb.CreateUtcTimestamp()
		dsblock.Header.PowDifficulty = uint64(r.peerServer.GetPowDifficulty())
		dsblock.Header.PowSubmissionNum = uint64(r.peerServer.ConsensusData.PoWSubList.Len())
		dsblock.Header.WinnerNonce = pow.Nonce
		dsblock.Header.WinnerPubKey = pow.PublicKey
		dsblock.Header.NewLeader = pow.Peer
		dsblock.Header.Miner = r.peerServer.SelfNode
		dsblock.Header.PreviousBlockHash = r.peerServer.DsBlockChain().CurrentBlock().Hash().Bytes()
	} else {
		return dsblock, fmt.Errorf("not enough pow submission results")
	}

	//dsblock.Body.PubKeyToCoinBaseMap = make(map[string][]byte)
	//// todo, set in genesis dsblock
	//if dsblock.Header.BlockNumber == 1 {
	//	for _, v := range r.peerServer.Committee {
	//		pkStr, err := r.peerServer.MsgSinger.PubKeyBytes2String(v.Pubkey)
	//		if err != nil {
	//			roleDsBaseLogger.Errorf("pubkey to string error: %s", err.Error())
	//			continue
	//		}
	//		dsblock.Body.PubKeyToCoinBaseMap[pkStr] = []byte("coinbase:" + v.Id.Name)
	//	}
	//}

	roleDsBaseLogger.Debugf("new committee leader is %+v", dsblock.Header.NewLeader.Id)

	shardingSize := r.peerServer.GetShardingSize()
	numOfSharding := r.peerServer.ConsensusData.PoWSubList.Len() / shardingSize
	shardingNodesTotal := numOfSharding * shardingSize

	if numOfSharding == 0 {
		dsblock.Header.ShardingSum = 1
	} else {
		dsblock.Header.ShardingSum = uint32(numOfSharding)
	}

	if shardingNodesTotal != 0 {
		roleDsBaseLogger.Debugf("total number of sharding nodes is %d", shardingNodesTotal)
		if shardingNodesTotal == r.peerServer.ConsensusData.PoWSubList.Len() {
			roleDsBaseLogger.Debugf("no nodes that submit pow result will be kicked off!")
		} else {
			roleDsBaseLogger.Debugf("%d nodes will be kicked off!", r.peerServer.ConsensusData.PoWSubList.Len()-shardingNodesTotal)
		}
		dsblock.Body.ShardingNodes = append(dsblock.Body.ShardingNodes, r.peerServer.Committee[len(r.peerServer.Committee)-1])
		for i := 1; i < shardingNodesTotal; i++ {
			dsblock.Body.ShardingNodes = append(dsblock.Body.ShardingNodes, r.peerServer.ConsensusData.PoWSubList.Get(i).Peer)
		}
	} else {
		roleDsBaseLogger.Debugf("the nodes that submit pow result is less than expected sharding size. "+
			"only one sharding generate and sharding size is %d", r.peerServer.ConsensusData.PoWSubList.Len())
		dsblock.Body.ShardingNodes = append(dsblock.Body.ShardingNodes, r.peerServer.Committee[len(r.peerServer.Committee)-1])
		for i := 1; i < r.peerServer.ConsensusData.PoWSubList.Len(); i++ {
			dsblock.Body.ShardingNodes = append(dsblock.Body.ShardingNodes, r.peerServer.ConsensusData.PoWSubList.Get(i).Peer)
		}
	}

	dsblock.Body.Committee = r.peerServer.Committee.EnqueueAndDequeueTail(dsblock.Header.NewLeader)

	dsblock.Body.CurrentBlockHash = dsblock.Hash().Bytes()

	//for i := 0; i < r.peerServer.ConsensusData.PoWSubList.Len(); i++ {
	//	pkStr, err := r.peerServer.MsgSinger.PubKeyBytes2String(r.peerServer.ConsensusData.PoWSubList.Get(i).PublicKey)
	//	if err != nil {
	//		continue
	//	}
	//	if _, ok := r.peerServer.ConsensusData.PubKeyToCoinBaseMap[pkStr]; ok {
	//		if !reflect.DeepEqual(r.peerServer.ConsensusData.PubKeyToCoinBaseMap[pkStr], r.peerServer.ConsensusData.PoWSubList.Get(i).Coinbase) {
	//			dsblock.Body.PubKeyToCoinBaseMap[pkStr] = r.peerServer.ConsensusData.PoWSubList.Get(i).Coinbase
	//		}
	//	} else {
	//		dsblock.Body.PubKeyToCoinBaseMap[pkStr] = r.peerServer.ConsensusData.PoWSubList.Get(i).Coinbase
	//	}
	//}
	//
	//if len(dsblock.Body.PubKeyToCoinBaseMap) == 0 {
	//	dsblock.Body.PubKeyToCoinBaseMap = nil
	//}

	return dsblock, nil
}

func (r *RoleDsBase) composeFinalBlock() (*pb.TxBlock, error) {
	finalBlock := &pb.TxBlock{}
	finalBlock.Header = &pb.TxBlockHeader{}
	finalBlock.Body = &pb.TxBlockBody{}
	finalBlock.Header.Timestamp = pb.CreateUtcTimestamp()
	if r.peerServer.ConsensusData.MicroBlockList.Len() > 0 {
		finalBlock.Header.DSBlockHash = r.peerServer.ConsensusData.MicroBlockList.Get(0).Header.DSBlockHash
		finalBlock.Header.DSBlockNum = r.peerServer.ConsensusData.MicroBlockList.Get(0).Header.DSBlockNum
		finalBlock.Header.BlockNumber = r.peerServer.ConsensusData.MicroBlockList.Get(0).Header.BlockNumber
		finalBlock.Header.Version = r.peerServer.ConsensusData.MicroBlockList.Get(0).Header.Version
		finalBlock.Header.DSCoinBase = append(finalBlock.Header.DSCoinBase, r.peerServer.CoinBase)
		finalBlock.Body.NumberOfMicroBlock = uint32(0)
		finalBlock.Header.TxNum = uint64(0)
		finalBlock.Header.GasUsed = uint64(0)
		finalBlock.Header.Miner = r.peerServer.SelfNode
		for i := 0; i < r.peerServer.ConsensusData.MicroBlockList.Len(); i++ {
			finalBlock.Header.TxNum += r.peerServer.ConsensusData.MicroBlockList.Get(i).Header.TxNum
			finalBlock.Header.GasUsed += r.peerServer.ConsensusData.MicroBlockList.Get(i).Header.GasUsed
			finalBlock.Header.ShardingLeadCoinBase = append(finalBlock.Header.ShardingLeadCoinBase, r.peerServer.ConsensusData.MicroBlockList.Get(i).ShardingLeadCoinBase)
			finalBlock.Body.NumberOfMicroBlock++
			finalBlock.Body.Transactions = append(finalBlock.Body.Transactions, r.peerServer.ConsensusData.MicroBlockList.Get(i).Transactions...)
			finalBlock.Body.MicroBlockHashes = append(finalBlock.Body.MicroBlockHashes, util.Hash(r.peerServer.ConsensusData.MicroBlockList.Get(i).Header).Bytes())
		}

		finalBlock.Header.GasLimit = uint64(0)
		for _, v := range finalBlock.Body.Transactions {
			finalBlock.Header.GasLimit += v.GasLimit
		}

		gasLimit := r.calcGasLimit(r.peerServer.TxBlockChain().CurrentBlock().(*pb.TxBlock))
		if gasLimit > finalBlock.Header.GasLimit {
			finalBlock.Header.GasLimit = gasLimit
		}

		finalBlock.Header.TxRoot = pb.ComputeMerkelRoot(finalBlock.Body.Transactions)
		//bc := r.peerServer.TxBlockChain()
		//statecatch, err := bc.State()
		//if err != nil {
		//	return nil, err
		//}
		//sc := statecatch.Copy()
		//_, _, gasUsed, err := bc.Processor().Process(finalBlock, sc, bc.VmConfig())
		//if err != nil {
		//	return nil, err
		//}
		//root := sc.IntermediateRoot(false)
		//finalBlock.Header.StateRoot = root.Bytes()
		//finalBlock.Header.GasUsed = gasUsed
		finalBlock.Header.PreviousBlockHash = r.peerServer.TxBlockChain().CurrentBlock().Hash().Bytes()
		fstr, err := finalBlock.ToReadable()
		if err != nil {
			roleDsBaseLogger.Debugf("composed final block is %+v", finalBlock)
		} else {
			roleDsBaseLogger.Debugf("composed final block is %s", fstr)
		}
	} else {
		roleDsBaseLogger.Error("no microblock in list")
		return finalBlock, errors.New("no microblock in list")
	}

	finalBlock.Body.CurrentBlockHash = finalBlock.Hash().Bytes()

	return finalBlock, nil
}

func (r *RoleDsBase) composeVCBlock() (*pb.VCBlock, error) {
	vcblock := &pb.VCBlock{}
	vcblock.Header = &pb.VCBlockHeader{}
	vcblock.Header.Timestamp = pb.CreateUtcTimestamp()
	vcblock.Header.ViewChangeDSNo = r.peerServer.DsBlockChain().CurrentBlock().NumberU64()
	vcblock.Header.ViewChangeTXNo = r.peerServer.TxBlockChain().CurrentBlock().NumberU64()
	vcblock.Header.Stage = pb.ConsensusType_name[int32(r.peerServer.ConsensusData.LastStage)]
	if r.peerServer.Committee.Len() < 2 {
		logger.Errorf("not enough backup nodes to choose for next consensus leader")
		return nil, errors.New("not enough backup nodes to choose for next consensus leader")
	}

	vcblock.Header.NewLeader = r.GetNewLeader(r.peerServer.ConsensusData.CurrentStage, r.peerServer.ConsensusData.LastStage)
	r.SetCurrentVCBlock(vcblock)
	return vcblock, nil
}

func (r *RoleDsBase) calcGasLimit(parent *pb.TxBlock) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.Header.GasUsed + parent.Header.GasUsed/2) / GasLimitBoundDivisor
	// decay = parentGasLimit / 1024 -1
	decay := parent.Header.GasLimit/GasLimitBoundDivisor - 1
	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parent.Header.GasLimit - decay + contrib
	if limit < MinGasLimit {
		limit = MinGasLimit
	}
	// If we're outside our allowed gas range, we try to hone towards them
	if limit < MinerGasFloor {
		limit = parent.Header.GasLimit + decay
		if limit > MinerGasFloor {
			limit = MinerGasFloor
		}
	} else if limit > MinerGasCeil {
		limit = parent.Header.GasLimit - decay
		if limit < MinerGasCeil {
			limit = MinerGasCeil
		}
	}
	return limit
}
