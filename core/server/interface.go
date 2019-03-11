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

package server

import (
	"context"

	pb "github.com/ok-chain/okchain/protos"
)

type IState interface {
	ProcessMsg(r IRole, msg *pb.Message, from *pb.PeerEndpoint) error
	AwaitDone()
	GetName() string
}

type DsMessageHandler interface {
	ProcessSetPrimary(msg *pb.Message, from *pb.PeerEndpoint) error
	ProcessPoWSubmission(msg *pb.Message, from *pb.PeerEndpoint) error
	ProcessMicroBlockSubmission(msg *pb.Message, from *pb.PeerEndpoint) error
}

type ShardingMessageHandler interface {
	ProcessStartPoW(msg *pb.Message, from *pb.PeerEndpoint) error
	ProcessDSBlock(msg *pb.Message, from *pb.PeerEndpoint) error
	ProcessFinalBlock(msg *pb.Message, from *pb.PeerEndpoint) error
}

type LookupMessageHandler interface {
	ProcessRegister(msg *pb.Message, from *pb.PeerEndpoint) error
}

type ConsensusHandler interface {
	VerifyBlock(msg *pb.Message, from *pb.PeerEndpoint, consensusType pb.ConsensusType) error
	ComposeResponse(consensusType pb.ConsensusType) (*pb.Message, error)
	ComposeFinalResponse(consensusType pb.ConsensusType) (*pb.Message, error)
	StartViewChange(currentStage, lastStage pb.ConsensusType) error
	GetCurrentDSBlock() *pb.DSBlock
	GetCurrentFinalBlock() *pb.TxBlock
	GetCurrentMicroBlock() *pb.MicroBlock
	GetCurrentVCBlock() *pb.VCBlock
	SetCurrentDSBlock(block *pb.DSBlock)
	SetCurrentFinalBlock(block *pb.TxBlock)
	SetCurrentMicroBlock(block *pb.MicroBlock)
	SetCurrentVCBlock(block *pb.VCBlock)
}

type iConsensusBase interface {
	ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint, r IRole) error
	GetCurrentLeader() *pb.PeerEndpoint
	GetCurrentConsensusStage() pb.ConsensusType
	SetCurrentConsensusStage(consensusType pb.ConsensusType)
	WaitForViewChange()
	SetIRole(r IRole)
}

type IConsensusLead interface {
	iConsensusBase
	InitiateConsensus(msg *pb.Message, to []*pb.PeerEndpoint, r IRole) error
	GetToleranceSize() int
}

type IConsensusBackup interface {
	iConsensusBase
	UpdateLeader(leader *pb.PeerEndpoint)
}

type IRole interface {
	ProcessMsg(msg *pb.Message, from *pb.PeerEndpoint) error
	ProcessTransaction(tx *pb.Transaction)
	ProcessConsensusMsg(msg *pb.Message, from *pb.PeerEndpoint) error
	Wait4PoWSubmission(ctx context.Context, cancle context.CancelFunc)
	OnConsensusCompleted(err error) error
	OnMircoBlockConsensusStarted(peer *pb.PeerEndpoint) error
	OnViewChangeConsensusStarted() error
	VerifyVCBlock(msg *pb.Message, from *pb.PeerEndpoint) (*pb.VCBlock, error)

	GetName() string
	GetStateName() string
	ChangeState(state StateType) error
	GetPeerServer() *PeerServer
	DumpCurrentState()
	//GetCurrentLeader() *pb.PeerEndpoint
	ProcessNewAdd()

	DsMessageHandler
	ShardingMessageHandler
	ConsensusHandler
	LookupMessageHandler
}

// Peer provides interface for a peer
type Peer interface {
	GetPeerEndpoint() (*pb.PeerEndpoint, error)
	NewOpenchainDiscoveryHello() (*pb.Message, error)
}

// MessageHandler standard interface for handling Openchain messages.
type IP2PHandler interface {
	HandleMessage(msg *pb.Message) error
	SendMessage(msg *pb.Message) error
	To() (pb.PeerEndpoint, error)
	Stop(bool) error
	Registered() bool
}

type ChatStream interface {
	Send(*pb.Message) error
	Recv() (*pb.Message, error)
}

type INode interface {
	StartSynchronization()
	GetBroadcastList()
	Execute()
}
