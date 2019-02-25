// Copyright The go-okchain Authors 2018,  All rights reserved.

// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package blockchain

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/event"
	"github.com/ok-chain/okchain/core"
	"github.com/ok-chain/okchain/core/consensus"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
)

var log = logging.MustGetLogger("BlockChain")

type BlockChain struct {
	db            database.Database
	chainFeed     event.Feed
	chainHeadFeed event.Feed
	scope         event.SubscriptionScope
	genesisBlock  protos.IBlock

	mu      sync.RWMutex // global mutex for locking chain operations
	chainmu sync.RWMutex // BlockChain insertion lock

	currentBlock atomic.Value // Current head of the block chain
	numberCache  *lru.Cache
	blockCache   *lru.Cache
	futureBlocks *lru.Cache // future blocks are blocks added for later processing

	quit          chan struct{}  // BlockChain quit channel
	running       int32          // running must be called atomically
	procInterrupt int32          // interrupt signaler for block processing, procInterrupt must be atomically called
	wg            sync.WaitGroup // chain processing wait group for shutting down
	engine        consensus.Engine
	impl          IBasicBlockChain
}

func NewBlockChain(db database.Database, engine consensus.Engine) (*BlockChain, error) {
	blockCache, _ := lru.New(BlockCacheLimit)
	futureBlocks, _ := lru.New(FutureBlkCacheLimit)
	numberCache, _ := lru.New(NumberCacheLimit)
	bc := &BlockChain{
		db:           db,
		quit:         make(chan struct{}),
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		numberCache:  numberCache,
		engine:       engine,
	}
	return bc, nil
}

func (bc *BlockChain) Start() error {
	if bc.genesisBlock = bc.GetBlockByNumber(0); bc.genesisBlock == nil {
		return ErrNoGenesis
	}
	if err := bc.impl.loadLastState(); err != nil {
		return err
	}
	go bc.update()

	return nil
}

func (bc *BlockChain) GetDatabase() database.Database {
	return bc.db
}

func (bc *BlockChain) GetProcInterrupt() bool {
	return atomic.LoadInt32(&bc.procInterrupt) == 1
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (bc *BlockChain) GetBlockByNumber(number uint64) protos.IBlock {
	hash := rawdb.ReadCanonicalHash(bc.db, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return bc.GetBlock(hash, number)
}
func (bc *BlockChain) GetBlock(hash common.Hash, number uint64) protos.IBlock {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := bc.blockCache.Get(hash); ok {
		if cachedNO := bc.impl.GetBlockNumber(hash); cachedNO == number && cachedNO != MissingNumber {
			return block.(protos.IBlock)
		}
	}
	block := bc.impl.ReadBlock(hash, number)
	if block == nil {
		return nil
	}

	// Cache the found block for next time and return
	bc.blockCache.Add(hash, block)
	return block
}

func (bc *BlockChain) loadLastState() error {
	head := rawdb.ReadHeadBlockHash(bc.db)
	if head == (common.Hash{}) {
		return bc.Reset()
	}
	currentBlock := bc.GetBlockByHash(head)
	if currentBlock == nil {
		return bc.Reset()
	}
	bc.currentBlock.Store(currentBlock)
	return nil
}

func (bc *BlockChain) Quit() {
	bc.quit <- struct{}{}
}

func (bc *BlockChain) update() {
	futureTimer := time.NewTicker(5 * time.Second)
	defer futureTimer.Stop()
	for {
		select {
		case <-futureTimer.C:
			bc.impl.ProcFutureBlocks()
		case <-bc.quit:
			return
		}
	}
}

// GetBlockByHash retrieves a block from the database by hash, caching it if found.
func (bc *BlockChain) GetBlockByHash(hash common.Hash) protos.IBlock {
	return bc.GetBlock(hash, bc.GetBlockNumber(hash))
}

// GetBlockNumber retrieves the block number belonging to the given hash
// from the cache or database
func (bc *BlockChain) GetBlockNumber(hash common.Hash) uint64 {
	if cached, ok := bc.numberCache.Get(hash); ok {
		return cached.(uint64)
	}

	number := rawdb.ReadHeaderNumber(bc.db, hash)
	if number != nil && *number != MissingNumber {
		bc.numberCache.Add(hash, *number)
		return *number
	}

	return MissingNumber
}

func (bc *BlockChain) InsertBlock(block protos.IBlock) (err error) {

	log.Infof("CurrentBlockChain: Height_%d, InsertedBlockNO_%d", bc.CurrentBlock().NumberU64(), block.NumberU64())
	err = bc.impl.ValidateBlock(block)
	isFutureBlk, _ := bc.impl.IsFutureBlockErr(err)

	if isFutureBlk {
		bc.futureBlocks.Add(block.Hash(), block)
		log.Infof("BlockNO_%d is suppose to be future block", block.NumberU64())
	}

	if err != nil {
		bc.reportBadBlock(block, err)
		return err
	}

	if err = bc.impl.batchInsert(block); err == nil {
		bc.futureBlocks.Remove(block.Hash())
		bc.currentBlock.Store(block)
	}

	return err
}

func (bc *BlockChain) reportBadBlock(blk protos.IBlock, err error) {
	msg := fmt.Sprintf("Bad Block Found. BlkNO: %d, Hash: %s, Error: %s", blk.NumberU64(),
		common.Bytes2Hex(blk.Hash().Bytes()), err.Error())
	log.Warning(msg)
}

func (bc *BlockChain) InsertChain(chain []protos.IBlock) (int, error) {

	for i := 1; i < len(chain); i++ {
		if chain[i].NumberU64() != chain[i-1].NumberU64()+1 || chain[i].ParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			return 0, fmt.Errorf("Not contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, chain[i-1].NumberU64(),
				chain[i-1].Hash().Bytes()[:4], i, chain[i].NumberU64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash().Bytes()[:4])
		}
	}

	for i, block := range chain {
		if err := bc.impl.InsertBlock(block); err != nil {
			log.Errorf("ERROR: %s, CurrentBlockChain: Height_%d, InsertedBlockNO_%d", err.Error(), bc.CurrentBlock().NumberU64(), block.NumberU64())
			return i, err
		}
	}
	return 0, nil
}

func (bc *BlockChain) CurrentBlock() protos.IBlock {
	return bc.currentBlock.Load().(protos.IBlock)
}

func (bc *BlockChain) Height() uint64 {
	return bc.CurrentBlock().NumberU64() + 1
}

// Genesis retrieves the chain's genesis block.
func (bc *BlockChain) Genesis() protos.IBlock {
	return bc.genesisBlock
}

// insert injects a new head block into the current block chain. This method
// assumes that the block is indeed a true head. It will also reset the head
// header and the head fast sync block to this very same block if they are older
// or if they are on a different side chain.
//
// Note, this function assumes that the `mu` mutex is held!
func (bc *BlockChain) batchInsert(block protos.IBlock) error {
	batch := bc.GetDatabase().NewBatch()
	bc.impl.WriteBlock(block, batch)
	rawdb.WriteCanonicalHash(batch, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(batch, block.Hash())
	err := batch.Write()
	return err
}

func (bc *BlockChain) ProcFutureBlocks() {
	blocks := make([]protos.IBlock, 0, bc.futureBlocks.Len())
	for _, hash := range bc.futureBlocks.Keys() {
		if block, exist := bc.futureBlocks.Peek(hash); exist {
			blocks = append(blocks, block.(protos.IBlock))
		}
	}
	if len(blocks) > 0 {
		protos.IBlockBy(protos.BlockNumberCompare).Sort(blocks)

		for i := range blocks {
			bc.InsertBlock(blocks[i])
		}
	}
}

func (bc *BlockChain) Rollback(chain []common.Hash) (int, error) {
	return 0, ErrNotImplemented
}

func (bc *BlockChain) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	panic(ErrNotImplemented)
}

func (bc *BlockChain) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	panic(ErrNotImplemented)
}

//Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *BlockChain) Reset() error {
	return nil
}

func (bc *BlockChain) WriteBlock(blk protos.IBlock, db database.Database) error {
	panic(ErrNotImplemented)
}

func (bc *BlockChain) ReadBlock(hash common.Hash, number uint64) protos.IBlock {
	panic(ErrNotImplemented)
}

func (bc *BlockChain) ValidateBlock(blk protos.IBlock) error {
	panic(ErrNotImplemented)
}

func (bc *BlockChain) IsFutureBlockErr(err error) (bool, error) {
	panic(ErrNotImplemented)
}
