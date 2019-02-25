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
	"errors"
	"sync"

	"github.com/ok-chain/okchain/util"
)

type StateContainerFactoryType func() *StateContainer
type StateContainer struct {
	StateMap map[StateType]IState
}

var StateContainerFactory StateContainerFactoryType = defaultStateFactory
var once sync.Once
var stateContainer *StateContainer

func newStateContainer(factory StateContainerFactoryType) {
	once.Do(func() {
		stateContainer = factory.new()
	})
}

func GetStateContainer() *StateContainer {
	return stateContainer
}

func (factory StateContainerFactoryType) new() *StateContainer {
	return factory()
}

func (sc *StateContainer) GetState(targetState StateType) (IState, error) {
	state, ok := sc.StateMap[targetState]
	if !ok {
		return nil, errors.New("invalid target state")
	}
	return state, nil
}

func defaultStateFactory() *StateContainer {
	util.OkChainPanic("Invalid defaultStateFactory")
	return nil
}
