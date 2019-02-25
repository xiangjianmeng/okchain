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
	"github.com/ok-chain/okchain/util"
	"github.com/pkg/errors"
)

type RoleContainer struct {
	RoleMap map[PeerRole_Type]IRole
}

type RoleContainerFactoryType func(server *PeerServer) *RoleContainer

var RoleContainerFactory RoleContainerFactoryType = defaultRoleContainerFactory

func (factory RoleContainerFactoryType) produceRoleContainer(p *PeerServer) *RoleContainer {
	return factory(p)
}

func (sc *RoleContainer) GetRole(target PeerRole_Type) (IRole, error) {
	role, ok := sc.RoleMap[target]
	if !ok {
		return nil, errors.New("Invalid targetState")
	}
	return role, nil
}

func defaultRoleContainerFactory(server *PeerServer) *RoleContainer {
	util.OkChainPanic("Invalid defaultRoleContainerFactory")
	return nil
}
