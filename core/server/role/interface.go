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
	"github.com/golang/protobuf/proto"
	ps "github.com/ok-chain/okchain/core/server"
	pb "github.com/ok-chain/okchain/protos"
)

type iRoleInternel interface {
	ps.IRole
	onWait4PoWSubmissionDone() error
	onWait4MicroBlockSubmissionDone() error
	onDsBlockConsensusCompleted(err error) error
	onFinalBlockConsensusCompleted(err error) error
	onMircoBlockConsensusCompleted(err error) error
	onViewChangeConsensusCompleted(err error) error
	resetShardingNodes(dsblock *pb.DSBlock)
	transitionCheck(target ps.StateType) error
	updateDSBlockRand() string
	updateTXBlockRand() string

	consensusExecution
}

type consensusExecution interface {
	getConsensusData(consensusType pb.ConsensusType) (proto.Message, error)

	produceAnnounce(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error)
	produceResponse(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error)
	produceFinalCollectiveSig(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error)
	produceFinalResponse(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error)
	produceBroadCastBlock(envelope *pb.Message, consensusType pb.ConsensusType) (*pb.Message, error)
}
