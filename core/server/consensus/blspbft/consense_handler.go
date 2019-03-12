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
* 共识handler，与Role通信
 */

package blspbft

import (
	"reflect"
	"sync"

	ps "github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	pb "github.com/ok-chain/okchain/protos"
)

var (
	loggerCHandler   = logging.MustGetLogger("consensusHandler")
	once             sync.Once
	consensusHandler ConsensusHandler
)

// store current and last consensus type
type ConsensusHandler struct {
	currentType   pb.ConsensusType
	lastStageType pb.ConsensusType
}

func init() {
	newConsensusHandler()
}

func newConsensusHandler() {
	once.Do(func() {
		consensusHandler = ConsensusHandler{}
	})
}

// verify block message when backup nodes received announcement message
func (ch *ConsensusHandler) VerifyMessage(msg *pb.Message, from *pb.PeerEndpoint, r ps.IRole) error {
	loggerCHandler.Debugf("begin to verify block in %s", pb.ConsensusType_name[int32(ch.currentType)])
	// call role method to handle different type block verify
	err := r.VerifyBlock(msg, from, ch.currentType)

	if err != nil {
		loggerCHandler.Debugf("vefiry block failed with error: %s", err.Error())
		return ErrVerifyBlock
	}

	return nil
}

// compose response message according to current consensus type
func (ch *ConsensusHandler) ComposedResponse(r ps.IRole) (*pb.Message, error) {
	loggerCHandler.Debugf("compose response message in %s, CurRole<%s>", pb.ConsensusType_name[int32(ch.currentType)], reflect.TypeOf(r))
	msg, err := r.ComposeResponse(ch.currentType)
	if err != nil {
		loggerCHandler.Debugf("compose response message error: %s", err.Error())
		return nil, ErrComposeMessage
	}
	return msg, nil
}

// compose final response message according to current consensus type
func (ch *ConsensusHandler) ComposedFinalResponse(r ps.IRole) (*pb.Message, error) {
	loggerCHandler.Debugf("compose final response message in %s", pb.ConsensusType_name[int32(ch.currentType)])
	msg, err := r.ComposeFinalResponse(ch.currentType)
	if err != nil {
		loggerCHandler.Debugf("compose final response message error: %s", err.Error())
		return nil, ErrComposeMessage
	}
	return msg, nil
}

func (ch *ConsensusHandler) GetCurrentConsensusType() pb.ConsensusType {
	return consensusHandler.currentType
}

func (ch *ConsensusHandler) SetCurrentConsensusType(consensusType pb.ConsensusType) {
	consensusHandler.currentType = consensusType
}

func (ch *ConsensusHandler) StartViewChange(r ps.IRole) error {
	// init last stage type, lastStageType cannot be pb.ConsensusType_ViewChangeConsensus
	if ch.currentType != pb.ConsensusType_ViewChangeConsensus {
		ch.lastStageType = ch.currentType
	}

	// call StartViewChange function in rolebase
	err := r.StartViewChange(ch.currentType, ch.lastStageType)
	if err != nil {
		return err
	}
	return nil
}
