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
	"context"

	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
)

func (r *RoleBase) VerifyBlock(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ComposeResponse(consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) ComposeFinalResponse(consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) produceAnnounce(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) produceFinalCollectiveSig(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) produceResponse(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) produceFinalResponse(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) produceBroadCastBlock(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error) {
	util.OkChainPanic("Invalid invoke")
	return nil, nil
}

func (r *RoleBase) VerifyConsensusMsg(err error) { util.OkChainPanic("Invalid invoke") }

func (r *RoleBase) OnMircoBlockConsensusStarted(peer *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) OnViewChangeConsensusStarted() error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) resetShardingNodes(dsblock *pb.DSBlock) { util.OkChainPanic("Invalid invoke") }

func (r *RoleBase) OnConsensusCompleted(err error, boolMapSign2 *pb.BoolMapSignature) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) Wait4PoWSubmission(ctx context.Context, cancle context.CancelFunc) {
	util.OkChainPanic("Invalid invoke")
}
func (r *RoleBase) ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}
func (r *RoleBase) onWait4PoWSubmissionDone() error { util.OkChainPanic("Invalid invoke"); return nil }

func (r *RoleBase) onWait4MicroBlockSubmissionDone() error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) onDsBlockConsensusCompleted(err error, boolMapSign2 *pb.BoolMapSignature) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) onFinalBlockConsensusCompleted(err error, boolMapSign2 *pb.BoolMapSignature) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) onViewChangeConsensusCompleted(err error) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) onMircoBlockConsensusCompleted(err error) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessSetPrimary(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessDSBlock(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessPoWSubmission(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}
func (r *RoleBase) ProcessMicroBlockSubmission(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessStartPoW(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessFinalBlock(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessRegister(msg *pb.Message, from *pb.PeerEndpoint) error {
	util.OkChainPanic("Invalid invoke")
	return nil
}

func (r *RoleBase) ProcessNewAdd() { util.OkChainPanic("Invalid invoke") }
