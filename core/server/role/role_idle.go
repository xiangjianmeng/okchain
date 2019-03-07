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
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/config"
	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/crypto/multibls"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
)

type RoleIdle struct {
	*RoleShardingBase
	idlePoWFlag bool // new node PoW flag
	registered  bool
}

var idleLogger = logging.MustGetLogger("idleRole")

func newRoleIdle(peer *ps.PeerServer) ps.IRole {
	base := newRoleShardingBase(peer)
	r := &RoleIdle{RoleShardingBase: base}
	r.initBase(r)
	r.name = "Idle"

	return r
}

func (r *RoleIdle) ProcessDSBlock(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
	peerServer := r.peerServer
	dsblock := &pb.DSBlock{}
	err := proto.Unmarshal(pbMsg.Payload, dsblock)
	if err != nil {
		idleLogger.Errorf("unmarshal dsblock message failed with error: %s", err.Error())
		return err
	}

	err = r.onDsBlockReady(dsblock)
	if err != nil {
		idleLogger.Errorf("handle dsblock error: %s", err.Error())
		return err
	}

	if !r.idlePoWFlag {
		// no pow
		return nil
	}

	if bytes.Equal(dsblock.Header.WinnerPubKey, peerServer.GetPubkey()) {
		idleLogger.Infof("I am new DS committee leader!")
		r.peerServer.ChangeRole(ps.PeerRole_DsLead, ps.STATE_WAIT4_MICROBLOCK_SUBMISSION)
	} else {
		r.initShardingNodeInfo(dsblock)
		index := r.peerServer.ShardingNodes.Index(peerServer.SelfNode)

		if index == 0 {
			idleLogger.Infof("I am this round ShardingLead")
			newRole := r.peerServer.ChangeRole(ps.PeerRole_ShardingLead, ps.STATE_MICROBLOCK_CONSENSUS)
			err = newRole.OnMircoBlockConsensusStarted(r.peerServer.ShardingNodes[0])
		} else if index == -1 {
			idleLogger.Infof("I am this round Idle")
			r.idlePoWFlag = false
		} else {
			idleLogger.Infof("I am this round ShardingBackup")
			newRole := r.peerServer.ChangeRole(ps.PeerRole_ShardingBackup, ps.STATE_MICROBLOCK_CONSENSUS)
			err = newRole.OnMircoBlockConsensusStarted(r.peerServer.ShardingNodes[0])
		}
	}
	if err != nil {
		return err
	}
	r.peerServer.DumpRoleAndState()
	return nil
}

func (r *RoleIdle) ProcessFinalBlock(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
	txblock := &pb.TxBlock{}
	err := proto.Unmarshal(pbMsg.Payload, txblock)
	if err != nil {
		idleLogger.Errorf("txblock unmarshal failed with error: %s", err.Error())
		return err
	}

	r.onFinalBlockReady(txblock)

	curTxBlockNum := r.peerServer.TxBlockChain().CurrentBlock().NumberU64()
	maxHeight := r.peerServer.Gsp.MaxAvailableLedgerHeight(false)
	// height=num+1
	if curTxBlockNum < maxHeight-1 {
		return nil
	}

	curDsBlock := r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock)
	numOfSharding := len(curDsBlock.Body.ShardingNodes) / r.peerServer.GetShardingSize()
	shardingSize := r.peerServer.GetShardingSize()
	shardingNodes := len(curDsBlock.Body.ShardingNodes)
	if (numOfSharding != 0 && shardingSize > 0 && txblock.Header.BlockNumber%uint64(shardingSize) == 0) ||
		(numOfSharding == 0 && shardingNodes > 0 && txblock.Header.BlockNumber%uint64(shardingNodes) == 0) {
		// start pow
		idleLogger.Debugf("start pow, waiting for find a results")

		// only for local testing, sleep 1s and start mine
		time.Sleep(1 * time.Second)

		// call miner function
		pk := multibls.PubKey{}
		pk.Deserialize(r.peerServer.SelfNode.Pubkey)
		miningResult, diff, err := r.mining(pk.GetHexString(), txblock.Header.BlockNumber)
		if err != nil {
			return err
		}

		// issue#9  whether the block is updated after mining
		newDsBlock := r.peerServer.DsBlockChain().CurrentBlock().(*pb.DSBlock)
		if curDsBlock.NumberU64() != newDsBlock.NumberU64() {
			idleLogger.Warningf("dsblock is updated to %d, i will ignore the PoWSubmission", newDsBlock.NumberU64())
			return nil
		}

		powSubmission := r.composePoWSubmission(miningResult, txblock.Header.BlockNumber, r.peerServer.SelfNode.Pubkey, diff)
		data, err := proto.Marshal(powSubmission)
		if err != nil {
			idleLogger.Errorf("PoWSubmission marshal error %s", err.Error())
			return err
		}
		res := &pb.Message{}
		res.Type = pb.Message_DS_PoWSubmission
		res.Timestamp = pb.CreateUtcTimestamp()
		res.Payload = data
		res.Peer = r.peerServer.SelfNode
		sig, err := r.peerServer.MsgSinger.SignHash(powSubmission.Hash().Bytes(), nil)
		if err != nil {
			idleLogger.Errorf("bls message sign failed with error: %s", err.Error())
			return err
		}
		res.Signature = sig

		err = r.peerServer.Multicast(res, curDsBlock.Body.Committee)
		if err != nil {
			idleLogger.Errorf("send message to all DS nodes failed")
			return err
		}

		// r.ChangeState(ps.STATE_WAITING_DSBLOCK)
		idleLogger.Debugf("waiting dsblock...")
		r.idlePoWFlag = true
	}

	return nil
}

// set primary for ds committee when system startup
func (r *RoleIdle) ProcessSetPrimary(rawMsg *pb.Message, from *pb.PeerEndpoint) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.registered {
		idleLogger.Warning("I registered, will not process set primary")
		return nil
	}
	if r.peerServer.DsBlockChain().CurrentBlock().NumberU64() > 0 {
		return RevertRole(r.peerServer)
	}

	inform := &pb.InformDs{}
	err := proto.Unmarshal(rawMsg.Payload, inform)
	if err != nil {
		idleLogger.Errorf("message unmarshal failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	if int(inform.DsSize) != len(inform.DsList) {
		idleLogger.Warningf("I need %d dsnode, but now have %d dsnode, waiting more nodes...", inform.DsSize, len(inform.DsList))

		return nil
	}
	r.peerServer.SetDsInit(inform.DsList)
	r.peerServer.ChatWithDs()
	idleLogger.Info("Waiting for say hello between dsnodes...")
	for i := 1; i <= 5; i++ {
		r.peerServer.UpdateRoleList()
		if len(r.peerServer.Committee) < len(inform.DsList) {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}
	myAddr := config.GetListenAddress()
	idleLogger.Debugf("current ds leader is %s, myaddr: %s", inform.Leader, myAddr)
	nextRole := ps.PeerRole_DsLead
	if myAddr == inform.Leader.Id.Name {
	} else {
		nextRole = ps.PeerRole_DsBackup
	}
	newRole := r.peerServer.ChangeRole(nextRole, ps.STATE_WAIT4_POW_SUBMISSION)
	r.peerServer.Registered <- struct{}{}
	r.registered = true

	id := r.peerServer.Committee.Index(inform.Leader)
	if myAddr != inform.Leader.Id.Name {
		newRole.(*RoleDsBackup).dsConsensusBackup.UpdateLeader(r.peerServer.Committee[id])
	}
	ctx, cancle := context.WithTimeout(context.Background(), time.Duration(r.peerServer.GetWait4PoWTime())*time.Second)
	go newRole.Wait4PoWSubmission(ctx, cancle)

	r.peerServer.DumpRoleAndState()
	return nil
}

func (r *RoleIdle) ProcessStartPoW(powmsg *pb.Message, from *pb.PeerEndpoint) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.registered {
		idleLogger.Warning("I registered, will not process start pow")
		return nil
	}

	if r.peerServer.DsBlockChain().CurrentBlock().NumberU64() > 0 {
		return RevertRole(r.peerServer)
	}

	inform := &pb.InformSharding{}
	err := proto.Unmarshal(powmsg.Payload, inform)
	if err != nil {
		idleLogger.Errorf("unmarshal startPow message failed with error: %s", err.Error())
		return ErrUnmarshalMessage
	}

	if int(inform.ShardingSize) != len(inform.ShardingList) {
		idleLogger.Warningf("I need %d shardingnode, but now have %d shardingnode, waiting more nodes...", inform.ShardingSize, len(inform.ShardingList))

		return nil
	}

	r.peerServer.SetDsInit(inform.DsList)
	r.peerServer.ChatWithDs()
	idleLogger.Info("Waiting for say hello to dsnodes...")
	for i := 1; i <= 5; i++ {
		r.peerServer.UpdateRoleList()
		if len(r.peerServer.Committee) < len(inform.DsList) {
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	idleLogger.Debugf("pow calculate parameter is %+v", inform.Pow)
	time.Sleep(time.Second * 5)
	pk := multibls.PubKey{}
	pk.Deserialize(r.peerServer.PublicKey)

	miningResult, diff, err := r.mining(pk.GetHexString(), inform.Pow.BlockNum)
	if err != nil {
		return ErrMine
	}

	powSubmission := r.composePoWSubmission(miningResult, inform.Pow.BlockNum, r.peerServer.SelfNode.Pubkey, diff)
	data, err := proto.Marshal(powSubmission)
	if err != nil {
		idleLogger.Errorf("PoWSubmission marshal failed with error: %s", err.Error())
		return ErrMarshalMessage
	}

	idleLogger.Debugf("powsubmission is %+v", powSubmission)
	res := &pb.Message{}
	res.Type = pb.Message_DS_PoWSubmission
	res.Timestamp = pb.CreateUtcTimestamp()
	res.Payload = data
	res.Peer = r.peerServer.SelfNode

	sig, err := r.peerServer.MsgSinger.SignHash(powSubmission.Hash().Bytes(), nil)
	if err != nil {
		idleLogger.Errorf("bls message sign failed with error: %s", err.Error())
		return ErrSignMessage
	}

	res.Signature = sig
	// send to all ds node
	r.peerServer.ChangeRole(ps.PeerRole_ShardingBackup, ps.STATE_WAITING_DSBLOCK)
	r.peerServer.Registered <- struct{}{}
	r.registered = true

	err = r.peerServer.Multicast(res, r.peerServer.Committee)
	if err != nil {
		idleLogger.Errorf("send message to all DS node failed")
		return ErrMultiCastMessage
	}

	return nil
}

func (r *RoleIdle) ProcessNewAdd() {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.registered {
		idleLogger.Warning("I registered, will not process start pow")
		return
	}

	r.peerServer.Registered <- struct{}{}
	r.registered = true
	return
}
