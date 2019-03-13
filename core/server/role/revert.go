// Copyright The go-okchain Authors 2019,  All rights reserved.

package role

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/gogo/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/crypto/multibls"
	pb "github.com/ok-chain/okchain/protos"
)

func RevertRole(srv *ps.PeerServer) error {
	// The latest dsblock and the latest txblock should be processed again.
	curDsBlock := srv.DsBlockChain().CurrentBlock().(*pb.DSBlock)
	curTxBlock := srv.TxBlockChain().CurrentBlock().(*pb.TxBlock)

	printBlock(curDsBlock, curTxBlock)

	onDsBlockReady(srv, curDsBlock)
	onFinalBlockReady(srv, curTxBlock)
	if srv.GetCurrentRole().GetName() == "Lookup" {
		return nil
	}
	pubkey := srv.PublicKey

	shardingSize := uint64(srv.GetShardingSize())
	shardingNodeNum := uint64(len(curDsBlock.Body.ShardingNodes))
	if shardingNodeNum < shardingSize {
		shardingSize = shardingNodeNum
	}
	//curTxDsBlockNumber:=(curTxBlock.NumberU64()+shardingSize-1)/shardingSize
	//curTxDsBlock:=srv.DsBlockChain().GetBlockByNumber(curTxDsBlockNumber)

	if curTxBlock.NumberU64() < curDsBlock.NumberU64()*shardingSize { //should build txblock
		if bytes.Equal(pubkey, curDsBlock.Header.WinnerPubKey) { //ds leader
			srv.ChangeRole(ps.PeerRole_DsLead, ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
		} else if srv.Committee.Has(srv.SelfNode) {
			// ds backup
			// 2. go to backup role and next state
			srv.ChangeRole(ps.PeerRole_DsBackup, ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
			loggerDsLead.Infof("I am next round DS Backup")
			srv.GetCurrentRole().(*RoleDsBackup).dsConsensusBackup.UpdateLeader(curDsBlock.Header.NewLeader)
		} else if hashNode(curDsBlock.Body.ShardingNodes, srv.SelfNode) {
			initShardingNodeInfo(srv, curDsBlock)
			leaderNumber := curTxBlock.NumberU64() % shardingSize
			logger.Debugf("my sharding nodes are %+v", srv.ShardingNodes)
			if reflect.DeepEqual(srv.ShardingNodes[leaderNumber].Pubkey, srv.SelfNode.Pubkey) { //sharding leader
				targetRole := ps.PeerRole_ShardingLead
				targetState := ps.STATE_MICROBLOCK_CONSENSUS
				srv.ChangeRole(targetRole, targetState)

			} else { //sharding backup
				targetRole := ps.PeerRole_ShardingBackup
				targetState := ps.STATE_MICROBLOCK_CONSENSUS
				srv.ChangeRole(targetRole, targetState)
				srv.GetCurrentRole().(*RoleShardingBackup).consensusBackup.UpdateLeader(srv.ShardingNodes[leaderNumber])
			}
			time.Sleep(time.Second * 5)
			srv.GetCurrentRole().OnMircoBlockConsensusStarted(srv.ShardingNodes[leaderNumber])
		} else {
			srv.ChangeRole(ps.PeerRole_Idle, ps.STATE_IDLE)
			fmt.Println("unknown role")
		}
	} else if curTxBlock.NumberU64() == curDsBlock.NumberU64()*shardingSize { //should build dsblock
		if bytes.Equal(pubkey, curDsBlock.Header.WinnerPubKey) { //ds leader
			srv.ChangeRole(ps.PeerRole_DsLead, ps.STATE_WAIT4_POW_SUBMISSION)
			ctx, cancle := context.WithTimeout(context.Background(), time.Duration(srv.GetWait4PoWTime())*time.Second)
			loggerDsLead.Debugf("waiting for POW_SUBMISSION")
			go srv.GetCurrentRole().(*RoleDsLead).Wait4PoWSubmission(ctx, cancle)
		} else if srv.Committee.Has(srv.SelfNode) {
			// ds backup
			// 2. go to backup role and next state
			srv.ChangeRole(ps.PeerRole_DsBackup, ps.STATE_WAIT4_POW_SUBMISSION)
			srv.GetCurrentRole().(*RoleDsBackup).dsConsensusBackup.UpdateLeader(curDsBlock.Header.NewLeader)
			loggerDsLead.Infof("I am next round DS Backup")
			ctx, cancle := context.WithTimeout(context.Background(), time.Duration(srv.GetWait4PoWTime())*time.Second)
			loggerDsLead.Debugf("waiting for POW_SUBMISSION")
			go srv.GetCurrentRole().(*RoleDsBackup).Wait4PoWSubmission(ctx, cancle)
		} else if hashNode(curDsBlock.Body.ShardingNodes, srv.SelfNode) {
			initShardingNodeInfo(srv, curDsBlock)
			leaderNumber := curTxBlock.NumberU64() % shardingSize
			time.Sleep(time.Second * 5)
			if reflect.DeepEqual(srv.ShardingNodes[leaderNumber].Pubkey, srv.SelfNode.Pubkey) { //sharding leader
				targetRole := ps.PeerRole_ShardingLead
				targetState := ps.STATE_WAITING_DSBLOCK
				srv.ChangeRole(targetRole, targetState)
				startPowLead(srv.GetCurrentRole().(*RoleShardingLead), curTxBlock)
			} else {
				// sharding backup
				targetRole := ps.PeerRole_ShardingBackup
				targetState := ps.STATE_WAITING_DSBLOCK
				srv.ChangeRole(targetRole, targetState)
				srv.GetCurrentRole().(*RoleShardingBackup).consensusBackup.UpdateLeader(srv.ShardingNodes[leaderNumber])
				startPowBackup(srv.GetCurrentRole().(*RoleShardingBackup), curTxBlock)
			}

		} else {
			srv.ChangeRole(ps.PeerRole_Idle, ps.STATE_IDLE)
			fmt.Println("unknown role")
		}
	} else {
		fmt.Println("error relationship about dsblocknumber and txblocknumber")
	}
	return nil
}

func onDsBlockReady(srv *ps.PeerServer, dsblock *pb.DSBlock) {
	srv.GetCurrentRole().SetCurrentDSBlock(dsblock)
	srv.Committee = dsblock.Body.Committee
	srv.ShardingNodes = dsblock.Body.ShardingNodes
	//srv.GetCurrentRole().ResetShardingId2ShardingNodeListMap(dsblock)
	peerServer := srv
	shardingSize := srv.GetShardingSize()
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
	for i, nodes := range peerServer.ShardingMap {
		fmt.Println(i, nodes)
	}

	//for k, v := range dsblock.Body.PubKeyToCoinBaseMap {
	//	srv.ConsensusData.PubKeyToCoinBaseMap[k] = v
	//}
}

func onFinalBlockReady(peerServer *ps.PeerServer, txblock *pb.TxBlock) error {
	peerServer.GetCurrentRole().SetCurrentFinalBlock(txblock)
	str, err := txblock.ToReadable()
	if err != nil {
		logger.Debugf("current finalblock is %+v", txblock)
	} else {
		logger.Debugf("current finalblock is %s", str)
	}
	return nil
}

func initShardingNodeInfo(peerServer *ps.PeerServer, dsblock *pb.DSBlock) {
	var shardingID uint32
	peerServer.ShardingNodes = []*pb.PeerEndpoint{}
	pk := peerServer.SelfNode.Pubkey

	shardingSize := peerServer.GetShardingSize()
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
				peerServer.ShardingNodes = append(peerServer.ShardingNodes, dsblock.Body.ShardingNodes[i])
			}
		} else {
			peerServer.ShardingNodes = append(peerServer.ShardingNodes, dsblock.Body.ShardingNodes[i])
		}
	}

	peerServer.ShardingID = shardingID
	sort.Sort(pb.ByPubkey{PeerEndpointList: peerServer.ShardingNodes})

	logger.Debugf("my sharding ID is %d", shardingID)
	logger.Debugf("my sharding peers are %+v", peerServer.ShardingNodes)
}

func startPowLead(r *RoleShardingLead, txblock *pb.TxBlock) error {
	pk := multibls.PubKey{}
	pk.Deserialize(r.peerServer.SelfNode.Pubkey)
	miningResult, diff, err := r.mining(pk.GetHexString(), txblock.Header.BlockNumber)
	if err != nil {
		return err
	}

	powSubmission := r.composePoWSubmission(miningResult, txblock.Header.BlockNumber, r.peerServer.SelfNode.Pubkey, diff)

	loggerShardingBase.Debugf("powsubmission is %+v", powSubmission)

	data, err := proto.Marshal(powSubmission)
	if err != nil {
		loggerShardingBase.Errorf("PoWSubmission marshal error %s", err.Error())
		return err
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
	return nil
}

func startPowBackup(r *RoleShardingBackup, txblock *pb.TxBlock) error {
	pk := multibls.PubKey{}
	pk.Deserialize(r.peerServer.SelfNode.Pubkey)
	miningResult, diff, err := r.mining(pk.GetHexString(), txblock.Header.BlockNumber)
	if err != nil {
		return err
	}

	powSubmission := r.composePoWSubmission(miningResult, txblock.Header.BlockNumber, r.peerServer.SelfNode.Pubkey, diff)

	loggerShardingBase.Debugf("powsubmission is %+v", powSubmission)

	data, err := proto.Marshal(powSubmission)
	if err != nil {
		loggerShardingBase.Errorf("PoWSubmission marshal error %s", err.Error())
		return err
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
	return nil
}

func printBlock(dsBlock *pb.DSBlock, txBlock *pb.TxBlock) error {
	txBlockString, err := txBlock.ToReadable()
	if err != nil {
		return err
	}
	dsBlockString, err := dsBlock.ToReadable()
	if err != nil {
		return err
	}
	fmt.Println("curtxBlock", txBlockString)
	fmt.Println("curDsBlock", dsBlockString)
	logger.Info("curtxBlock", txBlockString)
	logger.Info("curDsBlock", dsBlockString)
	return nil
}

func hashNode(list []*pb.PeerEndpoint, e *pb.PeerEndpoint) bool {
	res := false
	for _, peer := range list {
		if reflect.DeepEqual(peer.Pubkey, e.Pubkey) {
			res = true
			break
		}
	}

	return res
}
