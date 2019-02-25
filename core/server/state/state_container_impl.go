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
* state machine类实现
* 每个类都没有成员变量，是无状态的
 */

package state

import (
	ps "github.com/ok-chain/okchain/core/server"
)

func newStateContainer() *ps.StateContainer {
	c := &ps.StateContainer{}
	c.StateMap = make(map[ps.StateType]ps.IState)

	{
		s := &STATE_DsIdle{}
		s.imp = s
		c.StateMap[ps.STATE_DS_IDLE] = s
	}
	{
		s := &STATE_Wait4POW_SUBMISSION{}
		s.imp = s
		c.StateMap[ps.STATE_WAIT4_POW_SUBMISSION] = s
	}
	{
		s := &STATE_DSBLOCK_CONSENSUS_PREP{}
		s.imp = s
		c.StateMap[ps.STATE_DSBLOCK_CONSENSUS_PREP] = s
	}
	{
		s := &STATE_DSBLOCK_CONSENSUS{}
		s.imp = s
		c.StateMap[ps.STATE_DSBLOCK_CONSENSUS] = s
	}
	{
		s := &STATE_WAIT4_MICROBLOCK_SUBMISSION{}
		s.imp = s
		c.StateMap[ps.STATE_WAIT4_MICROBLOCK_SUBMISSION] = s
	}
	{
		s := &STATE_FINALBLOCK_CONSENSUS_PREP{}
		s.imp = s
		c.StateMap[ps.STATE_FINALBLOCK_CONSENSUS_PREP] = s
	}
	{
		s := &STATE_FINALBLOCK_CONSENSUS{}
		s.imp = s
		c.StateMap[ps.STATE_FINALBLOCK_CONSENSUS] = s
	}
	{
		s := &STATE_VIEWCHANGE_CONSENSUS_PREP{}
		s.imp = s
		c.StateMap[ps.STATE_VIEWCHANGE_CONSENSUS_PREP] = s
	}
	{
		s := &STATE_VIEWCHANGE_CONSENSUS{}
		s.imp = s
		c.StateMap[ps.STATE_VIEWCHANGE_CONSENSUS] = s
	}
	{
		s := &STATE_ShardingIdle{}
		s.imp = s
		c.StateMap[ps.STATE_SHARDING_IDLE] = s
	}
	{
		s := &STATE_POW_SUBMISSION{}
		s.imp = s
		c.StateMap[ps.STATE_POW_SUBMISSION] = s
	}
	{
		s := &STATE_WAITING_DSBLOCK{}
		s.imp = s
		c.StateMap[ps.STATE_WAITING_DSBLOCK] = s
	}
	{
		s := &STATE_MICROBLOCK_CONSENSUS_PREP{}
		s.imp = s
		c.StateMap[ps.STATE_MICROBLOCK_CONSENSUS_PREP] = s
	}
	{
		s := &STATE_MICROBLOCK_CONSENSUS{}
		s.imp = s
		c.StateMap[ps.STATE_MICROBLOCK_CONSENSUS] = s
	}
	{
		s := &STATE_WAITING_FINALBLOCK{}
		s.imp = s
		c.StateMap[ps.STATE_WAITING_FINALBLOCK] = s
	}
	{
		s := &STATE_FALLBACK_CONSENSUS_PREP{}
		s.imp = s
		c.StateMap[ps.STATE_FALLBACK_CONSENSUS_PREP] = s
	}
	{
		s := &STATE_FALLBACK_CONSENSUS{}
		s.imp = s
		c.StateMap[ps.STATE_FALLBACK_CONSENSUS] = s
	}
	{
		s := &STATE_WAITING_FALLBACKBLOCK{}
		s.imp = s
		c.StateMap[ps.STATE_WAITING_FALLBACKBLOCK] = s
	}
	{
		s := &STATE_Idle{}
		s.imp = s
		c.StateMap[ps.STATE_IDLE] = s
	}
	{
		s := &STATE_Lookup{}
		s.imp = s
		c.StateMap[ps.STATE_LOOKUP] = s
	}

	return c
}
