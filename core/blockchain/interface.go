// Copyright The go-okchain Authors 2018,  All rights reserved.

package blockchain

import (
	"time"

	"math"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/event"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/core/vm"
	"github.com/ok-chain/okchain/protos"
)

type WriteStatus byte

const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SideStatTy
)

const (
	ToleranceFraction    = 0.667
	BlockCacheLimit      = 256
	BadBlockCacheLimit   = 128
	FutureBlkCacheLimit  = 1024
	NumberCacheLimit     = 2048
	MaxFutureBlkTry      = 30
	TriesInMemory        = 128
	AllowedFutureBlkTime = 15 * time.Second
	MissingNumber        = uint64(0xffffffffffffffff)
	StatsReportLimit     = 8 * time.Second // statsReportLimit is the time limit during import after which we always print
	// out progress. This avoids the user wondering what's going on.
)

func GetToleranceSize(group protos.PeerEndpointList) int {
	return int(math.Floor(float64(len(group)-1)*ToleranceFraction)) + 1
}

type IBasicBlockChain interface {
	//ConsensusEngine()
	CurrentBlock() protos.IBlock
	Height() uint64
	Genesis() protos.IBlock

	ReadBlock(hash common.Hash, number uint64) protos.IBlock
	WriteBlock(blk protos.IBlock, putter okcdb.Putter) error
	ValidateBlock(blk protos.IBlock) error
	IsFutureBlockErr(err error) (bool, error)

	GetBlock(hash common.Hash, number uint64) protos.IBlock
	GetBlockByHash(hash common.Hash) protos.IBlock
	GetBlockByNumber(number uint64) protos.IBlock
	GetBlockNumber(hash common.Hash) uint64
	GetDatabase() okcdb.Database
	batchInsert(block protos.IBlock) error
	loadLastState() error
	InsertBlock(block protos.IBlock) (err error)
	InsertChain(chain []protos.IBlock) (int, error)
	Rollback(chain []common.Hash) (int, error)

	Start() error
	Quit()
	GetProcInterrupt() bool
	ProcFutureBlocks()

	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

type ITxBlockchain interface {
	IBasicBlockChain
	Config() *config.ChainConfig
	VmConfig() vm.Config
	Processor() Processor
	Validator() Validator
	StateDB() state.Database

	SetProcessor(processor Processor)
	SetValidator(validator Validator)
	State() (*state.StateDB, error)
	StateAt(root common.Hash) (*state.StateDB, error)
}

type Processor interface {
	Process(block *protos.TxBlock, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error)
	SetPeerServer(server core.StateProcessorPeer)
}

// CacheConfig contains the configuration values for the trie caching/pruning
// that's resident in a blockchain.
type CacheConfig struct {
	Disabled      bool          // Whether to disable trie write caching (archive node)
	TrieNodeLimit int           // Memory limit (MB) at which to flush the current in-memory trie to disk
	TrieTimeLimit time.Duration // Time limit after which to flush the current in-memory trie to disk
}

func GetDefaultCacheConfig() *CacheConfig {
	cc := CacheConfig{
		Disabled:      true,
		TrieNodeLimit: 1024,
		TrieTimeLimit: time.Second,
	}
	return &cc
}
