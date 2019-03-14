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
	"encoding/hex"
	"math/big"
	"reflect"
	"sort"
	"sync"

	"github.com/golang/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	gossip_common "github.com/ok-chain/okchain/p2p/gossip/common"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

var logger = logging.MustGetLogger("baseRole")

func init() {
	ps.RoleContainerFactory = newRoleContainer
}

const (
	CONSENSUS_START_WAITTIME       int64 = 5
	ToleranceFraction                    = 0.667
	MICROBLOCK_MAXCOMPOSETX_NUMBER int   = 10000
)

type runConsensus func(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error)
type Consensus2FuncMapType map[pb.Message_Type]runConsensus

type RoleBase struct {
	imp              iRoleInternel // pointing to myself for polymorphic, works as 'this' pointer in C++/Java
	peerServer       *ps.PeerServer
	lock             sync.Mutex
	state            ps.IState
	name             string
	consensusFuncMap Consensus2FuncMapType
}

func (r *RoleBase) initBase(imp iRoleInternel) {
	r.imp = imp
	r.buildConsensusFuncMap()
}

func (r *RoleBase) GetPeerServer() *ps.PeerServer {
	return r.peerServer
}

func (r *RoleBase) getConsensusFunc(consensusType pb.ConsensusType, messageType pb.Message_Type) (runConsensus, error) {
	consensusFunc, ok := r.consensusFuncMap[messageType]

	if !ok {
		logger.Errorf("Invalid consensusType or messageType")
		return nil, ErrInvalidMessage
	}
	return consensusFunc, nil
}

func (r *RoleBase) buildConsensusFuncMap() {
	r.consensusFuncMap = make(map[pb.Message_Type]runConsensus)
	r.consensusFuncMap[pb.Message_Consensus_Announce] =
		func(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
			return r.imp.produceAnnounce(envelope, consensusType)
		}

	r.consensusFuncMap[pb.Message_Consensus_FinalCollectiveSig] =
		func(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
			return r.imp.produceFinalCollectiveSig(envelope, consensusType)
		}

	r.consensusFuncMap[pb.Message_Consensus_Response] =
		func(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
			return r.imp.produceResponse(envelope, consensusType)
		}

	r.consensusFuncMap[pb.Message_Consensus_FinalResponse] =
		func(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
			return r.imp.produceFinalResponse(envelope, consensusType)
		}

	r.consensusFuncMap[pb.Message_Consensus_BroadCastBlock] =
		func(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
			return r.imp.produceBroadCastBlock(envelope, consensusType)
		}
}

func (r *RoleBase) GetName() string {
	return r.name
}

func (r *RoleBase) GetStateName() string {
	return r.state.GetName()
}

func (r *RoleBase) ProcessTransaction(tx *pb.Transaction) {
	logger.Debugf("gossip_tx_msg seq_num: %d", tx.Nonce)
	r.peerServer.GetTxPool().AddRemote(tx)
}

func (r *RoleBase) ProcessMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	logger.Debugf("%s: roleType<%s>, roleValue<%v>", util.GId, reflect.TypeOf(r.imp), r)
	logger.Debugf("%s: stateType<%s>, stateValue<%v>", util.GId, reflect.TypeOf(r.state), r.state)

	r.state.ProcessMsg(r.imp, msg, from)
	return nil
}

func (r *RoleBase) DumpCurrentState() {
	logger.Debugf("%s: current role <%s>, state <%s>",
		util.GId, reflect.TypeOf(r.imp), reflect.TypeOf(r.state))
}

func (r *RoleBase) transitionCheck(target ps.StateType) error {
	return nil
}

func (r *RoleBase) ChangeState(target ps.StateType) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	// TODO: add state transition check for each chile role
	err := r.imp.transitionCheck(target)

	if err == nil {
		state, err := ps.GetStateContainer().GetState(target)
		if err == nil {
			r.state = state
		} else {
			logger.Errorf("get container state failed with error: %s", err.Error())
			return ErrTargetState
		}
	}
	return nil
}

func (r *RoleBase) dumpNetworkTopology() {
	logger.Debugf("============Dump=============")
	defer logger.Debugf("============Dump end=============")

	logger.Debugf("Committee:")
	r.peerServer.Committee.Dump()

	logger.Debugf("ShardingNodes:")
	r.peerServer.ShardingNodes.Dump()

	logger.Debugf("ShardingMap:")
	r.peerServer.ShardingMap.Dump()
}

func (r *RoleBase) resetShardingId2ShardingNodeListMap(dsblock *pb.DSBlock) {
	peerServer := r.peerServer
	shardingSize := r.peerServer.GetShardingSize()
	numOfSharding := len(dsblock.Body.ShardingNodes) / shardingSize
	shardingNodesTotal := numOfSharding * shardingSize

	//cal shardingMap
	peerServer.ShardingMap = make(pb.PeerEndpointMap)

	if shardingNodesTotal != 0 {
		for i := 1; i < shardingNodesTotal; i++ {
			peerServer.ShardingMap[uint32(i%numOfSharding)] =
				append(peerServer.ShardingMap[uint32(i%numOfSharding)], dsblock.Body.ShardingNodes[i])
		}
		peerServer.ShardingMap[uint32(0)] = append(peerServer.ShardingMap[uint32(0)], dsblock.Body.ShardingNodes[0])
	} else {
		for i := 0; i < shardingNodesTotal; i++ {
			peerServer.ShardingMap[uint32(0)] = append(peerServer.ShardingMap[uint32(0)], dsblock.Body.ShardingNodes[i])
		}
	}
}

func (r *RoleBase) onDsBlockReady(dsblock *pb.DSBlock) error {
	if r.peerServer.DsBlockChain().CurrentBlock().NumberU64() >= dsblock.Header.BlockNumber {
		return ErrDuplicatedBlock
	}

	if err := r.peerServer.DsBlockChain().InsertBlock(dsblock); err != nil {
		logger.Errorf("dsBlockChain insertChain error, %s", err.Error())
		return ErrInsertBlock
	}
	r.peerServer.Gossip.UpdateDsLedgerHeight(r.peerServer.DsBlockChain().Height(), gossip_common.ChainID("A"))

	r.SetCurrentDSBlock(dsblock)
	logger.Debugf("dsBlock insertChain success! :=)")
	logger.Debugf("****************************************************")
	logger.Debugf("current dsblock is %+v", r.peerServer.ConsensusData.CurrentDSBlock)
	logger.Debugf("****************************************************")

	// 1. reset all ds, head is the new ds lead
	r.peerServer.Committee = dsblock.Body.Committee
	logger.Debugf("current committee is %+v", r.peerServer.Committee)

	// 2. reset all shard nodes, sorted by public key
	// todo: use r.imp.resetShardingNodes(dsblock)
	r.peerServer.ShardingNodes = dsblock.Body.ShardingNodes

	// 3. reset ShardingMap
	r.resetShardingId2ShardingNodeListMap(dsblock)

	//for k, v := range dsblock.Body.PubKeyToCoinBaseMap {
	//	r.peerServer.ConsensusData.PubKeyToCoinBaseMap[k] = v
	//}

	//logger.Debugf("current pubkey to coinbase map is %+v", r.peerServer.ConsensusData.PubKeyToCoinBaseMap)
	//logger.Debugf("current pubkey to coinbase map len is %d", len(r.peerServer.ConsensusData.PubKeyToCoinBaseMap))

	return nil
}

func (r *RoleBase) onFinalBlockReady(txblock *pb.TxBlock) error {
	r.SetCurrentFinalBlock(txblock)
	if err := r.peerServer.TxBlockChain().InsertBlock(txblock); err != nil {
		logger.Errorf("txBlockChain insertChain error, %s", err.Error())
		// tmp := r.peerServer.TxBlockChain().CurrentBlock()
		// tmpstr, err := tmp.ToReadable()
		// if err != nil {
		// 	logger.Debugf("latest txblock in chain %+v", tmp)
		// } else {
		// 	logger.Debugf("latest txblock in chain %s", tmpstr)
		// }
		// curstr, err := txblock.ToReadable()
		// if err != nil {
		// 	logger.Debugf("current txblock in chain %+v", txblock)
		// } else {
		// 	logger.Debugf("current txblock in chain %s", curstr)
		// }
		return ErrInsertBlock
	}
	r.peerServer.Gossip.UpdateTxLedgerHeight(r.peerServer.TxBlockChain().Height(), gossip_common.ChainID("A"))

	logger.Debugf("finalBlock insertChain success! :=)")
	logger.Debugf("=========================================================")
	logger.Debugf("current finalblock is %+v", txblock)
	logger.Debugf("=========================================================")
	return nil
}

func (r *RoleBase) produceConsensusPayload(input proto.Message, consensusType pb.ConsensusType) ([]byte, error) {
	blockdata, err := proto.Marshal(input)
	if err != nil {
		logger.Errorf("marshal block message failed with error: %s", err.Error())
		return nil, ErrMarshalMessage
	}

	payload := &pb.ConsensusPayload{}
	payload.Msg = blockdata
	payload.Type = consensusType

	data, err := proto.Marshal(payload)
	if err != nil {
		logger.Errorf("marshal consensus payload message failed with error: %s", err.Error())
		return nil, ErrMarshalMessage
	}

	return data, nil
}

func (r *RoleBase) produceConsensusMessage(consensusType pb.ConsensusType, msgType pb.Message_Type) (*pb.Message, error) {
	envelope := &pb.Message{Type: msgType}
	envelope.Timestamp = pb.CreateUtcTimestamp()
	envelope.Peer = r.peerServer.SelfNode

	produceFunc, err := r.getConsensusFunc(consensusType, msgType)
	if err != nil {
		return nil, ErrInvalidMessage
	}

	// TODO: alternative: generate input and payload in produceFunc, rather than here
	return produceFunc(envelope, consensusType)
}

func (r *RoleBase) updateDSBlockRand() string {
	ret := r.peerServer.DsBlockChain().CurrentBlock().Hash().String()
	return ret
}

func (r *RoleBase) updateTXBlockRand() string {
	ret := r.peerServer.TxBlockChain().CurrentBlock().Hash().String()
	return ret
}

func (r *RoleBase) GetCurrentLeader() *pb.PeerEndpoint {
	return r.GetCurrentLeader()
}

func (r *RoleBase) GetCurrentDSBlock() *pb.DSBlock {
	return r.peerServer.ConsensusData.CurrentDSBlock
}

func (r *RoleBase) GetCurrentFinalBlock() *pb.TxBlock {
	return r.peerServer.ConsensusData.CurrentFinalBlock
}

func (r *RoleBase) GetCurrentMicroBlock() *pb.MicroBlock {
	return r.peerServer.ConsensusData.CurrentMicroBlock
}

func (r *RoleBase) GetCurrentVCBlock() *pb.VCBlock {
	return r.peerServer.ConsensusData.CurrentVCBlock
}

func (r *RoleBase) SetCurrentDSBlock(block *pb.DSBlock) {
	r.peerServer.ConsensusData.CurrentDSBlock = block
}

func (r *RoleBase) SetCurrentFinalBlock(block *pb.TxBlock) {
	r.peerServer.ConsensusData.CurrentFinalBlock = block
}

func (r *RoleBase) SetCurrentMicroBlock(block *pb.MicroBlock) {
	r.peerServer.ConsensusData.CurrentMicroBlock = block
}

func (r *RoleBase) SetCurrentVCBlock(block *pb.VCBlock) {
	r.peerServer.ConsensusData.CurrentVCBlock = block
}

func (r *RoleBase) getConsensusData(consensusType pb.ConsensusType) (proto.Message, error) {
	switch consensusType {
	case pb.ConsensusType_DsBlockConsensus:
		return r.GetCurrentDSBlock(), nil
	case pb.ConsensusType_FinalBlockConsensus:
		return r.GetCurrentFinalBlock(), nil
	case pb.ConsensusType_MicroBlockConsensus:
		return r.GetCurrentMicroBlock(), nil
	case pb.ConsensusType_ViewChangeConsensus:
		return r.GetCurrentVCBlock(), nil
	default:
		return nil, ErrHandleInCurrentState
	}
}

func (r *RoleBase) initShardingNodeInfo(dsblock *pb.DSBlock) {
	var shardingID uint32
	r.peerServer.ShardingNodes = []*pb.PeerEndpoint{}
	pk := r.peerServer.SelfNode.Pubkey

	shardingSize := r.peerServer.GetShardingSize()
	numOfSharding := len(dsblock.Body.ShardingNodes) / shardingSize
	shardingNodesTotal := numOfSharding * shardingSize

	for i := 0; i < len(dsblock.Body.ShardingNodes); i++ {
		if reflect.DeepEqual(dsblock.Body.ShardingNodes[i].Pubkey, pk) {
			if shardingNodesTotal != 0 {
				shardingID = uint32(i % numOfSharding)
				break
			} else {
				shardingID = uint32(0)
				break
			}
		}
	}

	for i := 0; i < len(dsblock.Body.ShardingNodes); i++ {
		if shardingNodesTotal != 0 {
			if uint32(i%numOfSharding) == shardingID {
				r.peerServer.ShardingNodes = append(r.peerServer.ShardingNodes, dsblock.Body.ShardingNodes[i])
			}
		} else {
			r.peerServer.ShardingNodes = append(r.peerServer.ShardingNodes, dsblock.Body.ShardingNodes[i])
		}
	}

	r.peerServer.ShardingID = shardingID
	sort.Sort(pb.ByPubkey{PeerEndpointList: r.peerServer.ShardingNodes})

	logger.Debugf("my sharding ID is %d", shardingID)
	logger.Debugf("my sharding peers are %+v", r.peerServer.ShardingNodes)
}

func (r *RoleBase) GetNewLeader(currentStage, lastStage pb.ConsensusType) *pb.PeerEndpoint {
	newLeader := &pb.PeerEndpoint{}
	newLeader = r.getNewLeaderWhenNotViewChange(currentStage)
	if currentStage == pb.ConsensusType_ViewChangeConsensus {
		if r.GetCurrentVCBlock() != nil && r.GetCurrentVCBlock().Header != nil && r.GetCurrentVCBlock().Header.NewLeader != nil {
			switch lastStage {
			case pb.ConsensusType_DsBlockConsensus, pb.ConsensusType_FinalBlockConsensus:
				for i := 0; i < len(r.peerServer.Committee); i++ {
					if reflect.DeepEqual(r.peerServer.Committee[i].Pubkey, r.GetCurrentVCBlock().Header.NewLeader.Pubkey) {
						newLeader = r.peerServer.Committee[(i+1)%len(r.peerServer.Committee)]
						break
					}
				}
			case pb.ConsensusType_MicroBlockConsensus:
				for i := 0; i < len(r.peerServer.ShardingNodes); i++ {
					if reflect.DeepEqual(r.peerServer.ShardingNodes[i].Pubkey, r.GetCurrentVCBlock().Header.NewLeader.Pubkey) {
						newLeader = r.peerServer.ShardingNodes[(i+1)%len(r.peerServer.ShardingNodes)]
						break
					}
				}
			}
		} else {
			logger.Debugf("vcblock is misssing, try to use default leader")
			newLeader = r.getNewLeaderWhenNotViewChange(currentStage)
		}
	}
	logger.Debugf("current stage is %+v, last stage is %+v", currentStage, lastStage)
	logger.Debugf("new leader is %+v", newLeader)
	return newLeader
}

func (r *RoleBase) getNewLeaderWhenNotViewChange(currentStage pb.ConsensusType) *pb.PeerEndpoint {
	newLeader := &pb.PeerEndpoint{}
	switch currentStage {
	case pb.ConsensusType_DsBlockConsensus, pb.ConsensusType_FinalBlockConsensus:
		if r.peerServer.Committee.Len() < 2 {
			logger.Errorf("not enough backup nodes to choose for next consensus leader!")
			return nil
		}
		newLeader = r.peerServer.Committee[1]
	case pb.ConsensusType_MicroBlockConsensus:
		shardingSize := r.peerServer.GetShardingSize()
		numOfSharding := len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes) / shardingSize
		for i := 0; i < r.peerServer.ShardingNodes.Len(); i++ {
			if (numOfSharding != 0 && uint64(i) == r.peerServer.TxBlockChain().CurrentBlock().NumberU64()%uint64(shardingSize)) ||
				(numOfSharding == 0 && r.peerServer.TxBlockChain().CurrentBlock().NumberU64()%uint64(len(r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock).Body.ShardingNodes)) == uint64(i)) {
				newLeader = r.peerServer.ShardingNodes[(i+1)%r.peerServer.ShardingNodes.Len()]
				break
			}
		}
	}
	return newLeader
}

func (r *RoleBase) StartViewChange(currentStage, lastStage pb.ConsensusType) error {
	// store consensus stage info in peerServer ConsensusData
	r.peerServer.ConsensusData.CurrentStage = currentStage
	r.peerServer.ConsensusData.LastStage = lastStage
	// get viewchange leader
	newLeader := r.GetNewLeader(currentStage, lastStage)
	var err error
	if reflect.DeepEqual(newLeader.Pubkey, r.peerServer.SelfNode.Pubkey) {
		logger.Infof("I am the viewchange and next consensus leader")
		logger.Infof("last consensus is %s", pb.ConsensusType_name[int32(lastStage)])
		switch lastStage {
		case pb.ConsensusType_DsBlockConsensus, pb.ConsensusType_FinalBlockConsensus:
			newRole := r.peerServer.ChangeRole(ps.PeerRole_DsLead, ps.STATE_VIEWCHANGE_CONSENSUS)
			err = newRole.OnViewChangeConsensusStarted()
		case pb.ConsensusType_MicroBlockConsensus:
			newRole := r.peerServer.ChangeRole(ps.PeerRole_ShardingLead, ps.STATE_VIEWCHANGE_CONSENSUS)
			err = newRole.OnViewChangeConsensusStarted()
		}
	} else {
		logger.Infof("I am the viewchange and next consensus backup")
		logger.Infof("last consensus is %s", pb.ConsensusType_name[int32(lastStage)])
		switch lastStage {
		case pb.ConsensusType_DsBlockConsensus, pb.ConsensusType_FinalBlockConsensus:
			newRole := r.peerServer.ChangeRole(ps.PeerRole_DsBackup, ps.STATE_VIEWCHANGE_CONSENSUS)
			err = newRole.OnViewChangeConsensusStarted()
		case pb.ConsensusType_MicroBlockConsensus:
			newRole := r.peerServer.ChangeRole(ps.PeerRole_ShardingBackup, ps.STATE_VIEWCHANGE_CONSENSUS)
			err = newRole.OnViewChangeConsensusStarted()
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (r *RoleBase) VerifyVCBlock(msg *pb.Message, from *pb.PeerEndpoint) (*pb.VCBlock, error) {
	announce := &pb.ConsensusPayload{}
	err := proto.Unmarshal(msg.Payload, announce)
	if err != nil {
		logger.Errorf("unmarshal consensus payload failed with error: %s", err.Error())
		return nil, ErrUnmarshalMessage
	}

	vcblock := &pb.VCBlock{}
	err = proto.Unmarshal(announce.Msg, vcblock)
	if err != nil {
		logger.Errorf("unmarshal vcblock failed with error: %s", err.Error())
		return nil, ErrUnmarshalMessage
	}
	if r.GetCurrentVCBlock() == nil {
		logger.Errorf("vcblock is nil")
		return nil, ErrEmptyBlock
	}
	if vcblock.Header.ViewChangeTXNo != r.peerServer.TxBlockChain().CurrentBlock().NumberU64() {
		logger.Errorf("txblock number is not equal, actual is %d, expected %d", vcblock.Header.ViewChangeTXNo, r.peerServer.TxBlockChain().CurrentBlock().NumberU64())
		return nil, ErrVerifyBlock
	}
	if vcblock.Header.ViewChangeDSNo != r.peerServer.DsBlockChain().CurrentBlock().NumberU64() {
		logger.Errorf("dsblock number is not equal, actual is %d, expected %d", vcblock.Header.ViewChangeDSNo, r.peerServer.DsBlockChain().CurrentBlock().NumberU64())
		return nil, ErrVerifyBlock
	}
	if vcblock.Header.Stage != pb.ConsensusType_name[int32(r.peerServer.ConsensusData.LastStage)] {
		logger.Errorf("last stage cannot match, actual is %+v, expected %+v", vcblock.Header.Stage, pb.ConsensusType_name[int32(r.peerServer.ConsensusData.LastStage)])
		return nil, ErrVerifyBlock
	}
	if !reflect.DeepEqual(vcblock.Header.NewLeader.Pubkey, r.GetNewLeader(r.peerServer.ConsensusData.CurrentStage, r.peerServer.ConsensusData.LastStage).Pubkey) {
		logger.Errorf("pubkey is not match, actual is %s, expected %s", hex.EncodeToString(vcblock.Header.NewLeader.Pubkey),
			hex.EncodeToString(r.GetNewLeader(r.peerServer.ConsensusData.CurrentStage, r.peerServer.ConsensusData.LastStage).Pubkey))
		return nil, ErrVerifyBlock
	}

	return vcblock, nil
}

func (r *RoleBase) mining(pk string, blockNumber uint64) (*pb.MiningResult, int64, error) {
	diff := r.peerServer.GetPowDifficulty()
	seed := r.updateDSBlockRand() + r.updateTXBlockRand() + r.peerServer.SelfNode.Address + pk + hex.EncodeToString(r.peerServer.CoinBase)
	miningResult, err := r.peerServer.Miner.Mine([]byte(seed), blockNumber+1, big.NewInt(diff))
	if err != nil {
		logger.Errorf("mine error with seed %+v, error: %s", seed, err.Error())
		return nil, diff, ErrMine
	}
	logger.Debugf("mine nonce is %d, hash is %s, resulthash is %s", miningResult.Nonce,
		hex.EncodeToString(miningResult.MixHash), hex.EncodeToString(miningResult.PowResult))
	return miningResult, diff, nil
}

func (r *RoleBase) composePoWSubmission(miningResult *pb.MiningResult, blockNumber uint64, pubkey []byte, diff int64) *pb.PoWSubmission {
	powSubmission := &pb.PoWSubmission{}
	powSubmission.BlockNum = blockNumber
	powSubmission.Nonce = miningResult.Nonce
	powSubmission.Result = miningResult
	powSubmission.Peer = r.peerServer.SelfNode
	powSubmission.PublicKey = pubkey
	powSubmission.Coinbase = r.peerServer.CoinBase
	powSubmission.Difficulty = uint64(diff)
	powSubmission.Rand1 = r.updateDSBlockRand()
	powSubmission.Rand2 = r.updateTXBlockRand()
	return powSubmission
}
