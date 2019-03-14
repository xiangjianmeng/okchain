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
	"math/big"
	"math/rand"

	"github.com/golang/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/spf13/viper"
)

type RoleLookup struct {
	*RoleBase
	rps          []*pb.PeerEndpoint
	shardingSize int
	dsSize       int
}

var lookupLogger = logging.MustGetLogger("lookupRole")

func newRoleLookup(peer *ps.PeerServer) ps.IRole {
	base := &RoleBase{peerServer: peer}
	r := &RoleLookup{RoleBase: base}
	r.initBase(r)
	r.name = "Lookup"
	r.shardingSize = viper.GetInt("sharding.size")
	r.dsSize = viper.GetInt("ds.size")

	return r
}

func (r *RoleLookup) ProcessDSBlock(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
	// verify message signature.
	var err error
	err = r.peerServer.MessageVerify(pbMsg)
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
	err = r.peerServer.VerifyDSBlockMultiSign2(dsBlockSign2)
	if err != nil {
		idleLogger.Errorf("verify dsBlockSign2 failed with error: %s", err.Error())
		return nil
	}
	// verify and insert dsblock.
	dsblock := dsBlockSign2.Block

	err = r.onDsBlockReady(dsblock)
	if err != nil {
		lookupLogger.Errorf("handle dsblock error: %s", err.Error())
		return err
	}

	r.peerServer.DumpRoleAndState()

	return nil
}

func (r *RoleLookup) ProcessFinalBlock(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
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

	return r.onFinalBlockReady(txblock)
}

func (r *RoleLookup) ProcessRegister(pbMsg *pb.Message, from *pb.PeerEndpoint) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	i := pb.PeerEndpointList(r.rps).Index(from)
	if i == -1 {
		if len(r.rps) == r.shardingSize+r.dsSize {
			lookupLogger.Warningf("Lookup registered peer are enough: %d, it will be added this network as new node", len(r.rps))
			r.informNewNode(from)
			return nil
		}
		r.rps = append(r.rps, from)
	}

	i = pb.PeerEndpointList(r.rps).Index(from)
	if i >= 0 && i < r.dsSize {
		lookupLogger.Debugf("Peer %s is ds backup, inform it", from.Id.Name)
		return r.informDsBackup(from)
	} else if i >= r.dsSize && i < r.shardingSize+r.dsSize {
		lookupLogger.Debugf("Peer %s is sharding, inform it", from.Id.Name)
		err := r.informSharding(from)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *RoleLookup) informNewNode(to *pb.PeerEndpoint) error {
	return r.peerServer.SendMsg(&pb.Message{Type: pb.Message_Node_NewAdd}, to.Id.Name)
}

func (r *RoleLookup) informDsBackup(to *pb.PeerEndpoint) error {
	up := r.dsSize
	if r.dsSize > len(r.rps) {
		up = len(r.rps)
	}
	data, err := proto.Marshal(&pb.InformDs{
		Leader: r.rps[0],
		DsSize: int32(r.dsSize),
		DsList: r.rps[0:up],
	})
	if err != nil {
		return err
	}

	return r.peerServer.SendMsg(&pb.Message{Type: pb.Message_DS_SetPrimary, Payload: data}, to.Id.Name)
}

func (r *RoleLookup) informSharding(to *pb.PeerEndpoint) error {
	startPow := &pb.StartPoW{
		BlockNum:   0,
		Difficulty: 100000,
		Rand1:      big.NewInt(rand.Int63()).String(),
		Rand2:      big.NewInt(rand.Int63()).String(),
		Peer:       to,
	}

	data, err := proto.Marshal(&pb.InformSharding{
		Pow:          startPow,
		ShardingSize: int32(r.shardingSize),
		DsList:       r.rps[0:r.dsSize],
		ShardingList: r.rps[r.dsSize:],
	})
	if err != nil {
		return err
	}

	return r.peerServer.SendMsg(&pb.Message{Type: pb.Message_Node_ProcessStartPoW, Payload: data}, to.Id.Name)
}
