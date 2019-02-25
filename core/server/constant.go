// Copyright The go-okchain Authors 2018,  All rights reserved.

package server

import (
	"time"

	logging "github.com/ok-chain/okchain/log"
)

type StateType int32
type PeerRole_Type int32

const (
	txChanSize     = 4096
	defaultTimeout = time.Second * 16

	PeerRole_Idle           PeerRole_Type = 0
	PeerRole_DsLead         PeerRole_Type = 1
	PeerRole_DsBackup       PeerRole_Type = 2
	PeerRole_ShardingBackup PeerRole_Type = 3
	PeerRole_ShardingLead   PeerRole_Type = 4
	PeerRole_Lookup         PeerRole_Type = 5

	// ds state
	STATE_DS_IDLE                     StateType = 100
	STATE_WAIT4_POW_SUBMISSION        StateType = 101
	STATE_DSBLOCK_CONSENSUS_PREP      StateType = 102
	STATE_DSBLOCK_CONSENSUS           StateType = 103
	STATE_WAIT4_MICROBLOCK_SUBMISSION StateType = 105
	STATE_FINALBLOCK_CONSENSUS_PREP   StateType = 106
	STATE_FINALBLOCK_CONSENSUS        StateType = 107
	STATE_VIEWCHANGE_CONSENSUS_PREP   StateType = 108
	STATE_VIEWCHANGE_CONSENSUS        StateType = 109

	// sharding state
	STATE_SHARDING_IDLE             StateType = 210
	STATE_POW_SUBMISSION            StateType = 211
	STATE_WAITING_DSBLOCK           StateType = 212
	STATE_MICROBLOCK_CONSENSUS_PREP StateType = 213
	STATE_MICROBLOCK_CONSENSUS      StateType = 214
	STATE_WAITING_FINALBLOCK        StateType = 215
	STATE_FALLBACK_CONSENSUS_PREP   StateType = 216
	STATE_FALLBACK_CONSENSUS        StateType = 217
	STATE_WAITING_FALLBACKBLOCK     StateType = 218

	// idle role state
	STATE_IDLE StateType = 300

	// look role state
	STATE_LOOKUP StateType = 400
)

var (
	peerLogger        = logging.MustGetLogger("peer")
	rpcMessageSize    = 32 * 1024 * 1024 //32MB
	powDifficulty     int64
	wait4PoWTime      int64
	viewchangeTimeOut int64
	shardingSize      int
	dsSize            int
	luAddList         []string
	dsInitList        []string
)
