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
)

import (
	"github.com/hashicorp/golang-lru"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/event"
	"github.com/ok-chain/okchain/common/mclock"
	"github.com/ok-chain/okchain/config"
	tcore "github.com/ok-chain/okchain/core"
	"github.com/ok-chain/okchain/core/blockchain/txblockchain"
	"github.com/ok-chain/okchain/core/consensus"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/core/vm"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

var txLogger = logging.MustGetLogger("TxBlockChain")

type TxBlockChain struct {
	*BlockChain
	chainConfig *config.ChainConfig
	cacheConfig *CacheConfig // Cache configuration for pruning

	triegc *prque.Prque // Priority queue mapping block numbers to tries to gc

	chainHeadFeed event.Feed
	chainFeed     event.Feed
	logsFeed      event.Feed

	procmu sync.RWMutex // block processor lock
	wg     sync.WaitGroup

	stateCache state.Database // State database to reuse between imports (contains state cache)
	badBlocks  *lru.Cache     // Bad block cache

	processor Processor
	validator Validator
	vmConfig  vm.Config

	hotState    atomic.Value // hot state generted by Processor().Process()
	hotReceipts atomic.Value // hot receipts generted by Processor().Process()
	hotLogs     atomic.Value // hot logs generted by Processor().Process()

}

func NewTxBlockChain(config *config.ChainConfig, db okcdb.Database, cacheConfig *CacheConfig) (*TxBlockChain, error) {

	badBlocks, _ := lru.New(BadBlockCacheLimit)
	bc, err := NewBlockChain(db, nil)
	if err != nil {
		txLogger.Errorf("Fail to initialize BaseBlockChain, Check why. %s", err.Error())
	}

	txBC := TxBlockChain{
		chainConfig: config,
		cacheConfig: cacheConfig,
		triegc:      prque.New(),
		badBlocks:   badBlocks,
		stateCache:  state.NewDatabase(db),
		BlockChain:  bc,
	}

	txBC.impl = &txBC
	txBC.SetProcessor(tcore.NewStateProcessor(config, &txBC))
	err = txBC.Start()

	return &txBC, nil
}

func (bc *TxBlockChain) loadLastState() error {
	if err := bc.BlockChain.loadLastState(); err != nil {
		return err
	}

	currentBlock := bc.CurrentBlock().(*protos.TxBlock)
	stateRoot := common.BytesToHash(currentBlock.Header.StateRoot)
	if _, err := state.New(stateRoot, bc.stateCache); err != nil {
		if err := bc.repair(&currentBlock); err != nil {
			return err
		}
	}

	log.Infof("Loaded most recent local full block, number %d, hash: %x, stateRoot: %s",
		currentBlock.GetHeader().GetBlockNumber(), currentBlock.Hash(), common.Bytes2Hex(stateRoot.Bytes()))
	return nil
}

func (bc *TxBlockChain) batchInsert(block protos.IBlock) error {
	batch := bc.GetDatabase().NewBatch()
	bc.impl.WriteBlock(block, batch)
	rawdb.WriteCanonicalHash(batch, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(batch, block.Hash())
	txblockchain.WriteTxLookupEntries(batch, block.(*protos.TxBlock))

	// TODO: add Bloom in future. FLT. 2018/12/05
	//if len(receipts) > 0 && block.GetHeader().Bloom == nil {
	//	block.Header.Bloom = types.CreateBloom(receipts).Bytes()
	//}

	if bc.hotReceipts.Load().(interface{}) != nil {
		receipts := bc.hotReceipts.Load().(types.Receipts)
		rawdb.WriteReceipts(bc.GetDatabase(), block.Hash(), block.NumberU64(), receipts)
	}

	if bc.hotState.Load().(interface{}) != nil {
		state := bc.hotState.Load().(*state.StateDB)
		root, err := state.Commit(false)
		if err != nil {
			return err
		} else {
			// TODO: 20181205. FLT. Add StateDB cache handling (GC, Trie Reference & Dereference) in future.
			if bc.cacheConfig.Disabled {
				triedb := bc.stateCache.TrieDB()
				if err = triedb.Commit(root, false); err != nil {
					return err
				}
			}
			//} else {
			//
			//}
		}
		txLogger.Debugf("TxBlock(%s) refresh stateRoot(%+v)", common.Bytes2Hex(block.Hash().Bytes()), root)
	}

	if bc.hotLogs.Load().(interface{}) != nil {
		logs := bc.hotLogs.Load().([]*types.Log)

		blk := block.(*protos.TxBlock)
		ev1 := tcore.ChainHeadEvent{Block: blk}
		ev2 := tcore.ChainEvent{Block: blk, Hash: blk.Hash(), Logs: logs}
		events := []interface{}{ev1, ev2}
		bc.PostChainEvents(events, logs)
		txLogger.Debugf("Step5: finish broadcast ChainHeadEvent.")
	}

	err := batch.Write()
	return err
}

func (bc *TxBlockChain) IsFutureBlockErr(err error) (bool, error) {
	isFuture := err == ErrFutureBlock || err == ErrUnknownAncestor ||
		err == ErrDSBlockNotExist || err == ErrPrunedAncestor || err == ErrCheckCoinBase
	return isFuture, nil
}

// GetReceiptsByHash retrieves the receipts for all transactions in a given block.
func (bc *TxBlockChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	num := bc.GetBlockNumber(hash)
	return rawdb.GetBlockReceipts(bc.GetDatabase(), hash, num)
}

func (bc *TxBlockChain) repair(head **protos.TxBlock) error {
	for {
		stateRoot := common.BytesToHash((*head).Header.StateRoot)
		if _, err := state.New(stateRoot, bc.stateCache); err == nil {
			log.Info("Rewound blockchain to past state", "number", (*head).NumberU64(), "hash", (*head).Hash())
			return nil
		}

		parentHash := common.BytesToHash((*head).Header.PreviousBlockHash)
		if pBlk := bc.GetBlock(parentHash, (*head).NumberU64()-1); pBlk != nil {
			(*head) = pBlk.(*protos.TxBlock)
		} else {
			return ErrUnknownAncestor
		}
	}
}

func (bc *TxBlockChain) StateDB() state.Database {
	return bc.stateCache
}

func (bc *TxBlockChain) SetProcessor(processor Processor) {
	bc.procmu.Lock()
	defer bc.procmu.Unlock()
	bc.processor = processor
}

func (bc *TxBlockChain) Processor() Processor {
	bc.procmu.Lock()
	defer bc.procmu.Unlock()
	return bc.processor
}

func (bc *TxBlockChain) VmConfig() vm.Config {
	return bc.vmConfig
}

func (bc *TxBlockChain) SetValidator(validator Validator) {
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()
	bc.validator = validator
}

// Validator returns the current validator.
func (bc *TxBlockChain) Validator() Validator {
	bc.procmu.RLock()
	defer bc.procmu.RUnlock()
	return bc.validator
}

func (bc *TxBlockChain) WriteBlock(blk protos.IBlock, db okcdb.Putter) (err error) {

	defer func() error {
		if e := recover(); e != nil {
			return fmt.Errorf("WriteTxBlock panic, %+v", e)
		} else {
			return err
		}
	}()
	txblockchain.WriteBlock(db, blk.(*protos.TxBlock))

	return nil
}

func (bc *TxBlockChain) syncBlockCache(blk protos.IBlock, db okcdb.Database) error {
	panic(ErrNotImplemented)
}

func (bc *TxBlockChain) ReadBlock(hash common.Hash, number uint64) protos.IBlock {
	if txBlk := txblockchain.ReadBlock(bc.GetDatabase(), hash, number); txBlk != nil {
		return txBlk
	}
	return nil
}

func (bc *TxBlockChain) ValidateBlock(blk protos.IBlock) (err error) {

	block := blk.(*protos.TxBlock)

	txLogger.Debugf("Step1.0 VerifyHeader")
	if err = bc.Validator().VerifyHeader(block.GetHeader()); err != nil {
		return err
	}
	txLogger.Debugf("Step1.1 ValidateBody")
	if err = bc.Validator().ValidateBody(block); err != nil {
		return err
	}

	txLogger.Debugf("Step1.2 VerifySeal")
	if err = bc.Validator().VerifySeal(block); err != nil {
		return err
	}

	txLogger.Debugf("Step1: Finish to verify header & body & signature, block hash: %s", common.Bytes2Hex(block.Hash().Bytes()))

	parentHash := common.BytesToHash(block.GetHeader().GetPreviousBlockHash())
	parent := bc.GetBlockByHash(parentHash).(*protos.TxBlock)
	stateRoot := common.BytesToHash(parent.GetHeader().StateRoot)
	state, err := state.New(stateRoot, bc.stateCache)
	if err != nil {
		return err
	}
	txLogger.Debugf("Step2: Finish to prepare startRoot: %s", common.Bytes2Hex(stateRoot.Bytes()))

	receipts, logs, usedGas, err := bc.processor.Process(block, state, bc.vmConfig)
	if err != nil {
		return err
	}
	txLogger.Debugf("Step3: Finish to process block with stateRoot.")

	if err = bc.Validator().ValidateState(block, parent, state, receipts, usedGas); err != nil {
		return err
	} else {
		bc.hotState.Store(state)
		bc.hotReceipts.Store(receipts)
		bc.hotLogs.Store(logs)
	}
	txLogger.Debugf("Step4: Finish to validate state.")

	return err
}

func (bc *TxBlockChain) InsertBlock(block protos.IBlock) (err error) {

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	if err = bc.BlockChain.InsertBlock(block); err != nil {
		return err
	}

	if block.NumberU64() != bc.CurrentBlock().NumberU64() {
		return fmt.Errorf("Cache Data out of syncronization, expectBlkNO: %d, cachedBlkNO: %d", block.NumberU64(), bc.CurrentBlock().NumberU64())
	}

	return nil
}

func (bc *TxBlockChain) State() (*state.StateDB, error) {
	root := common.BytesToHash(bc.CurrentBlock().(*protos.TxBlock).Header.StateRoot)
	return bc.StateAt(root)
}
func (bc *TxBlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	db, err := state.New(root, bc.stateCache)
	if err != nil {
		return nil, err
	}
	noAddr := common.Hash{}
	db.Prepare(noAddr, noAddr, bc.CurrentBlock().NumberU64(), 0)
	return db, nil
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (bc *TxBlockChain) SubscribeChainHeadEvent(ch chan<- tcore.ChainHeadEvent) event.Subscription {
	return bc.scope.Track(bc.chainHeadFeed.Subscribe(ch))
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (bc *TxBlockChain) SubscribeChainEvent(ch chan<- tcore.ChainEvent) event.Subscription {
	return bc.scope.Track(bc.chainFeed.Subscribe(ch))
}

// Config retrieves the blockchain's chain configuration.
func (bc *TxBlockChain) Config() *config.ChainConfig {
	return bc.chainConfig
}

// CurrentHeader retrieves the current header from the local chain.
func (bc *TxBlockChain) CurrentHeader() *protos.TxBlockHeader {
	if blk := bc.CurrentBlock(); blk != nil {
		return blk.(*protos.TxBlock).Header
	}
	return nil
}

// Engine retrieves the blockchain's consensus engine.
func (bc *TxBlockChain) Engine() consensus.Engine { return nil }

// GetHeader retrieves a block header from the database by hash and number.
func (bc *TxBlockChain) GetHeader(hash common.Hash, number uint64) *protos.TxBlockHeader {
	header := txblockchain.ReadHeader(bc.GetDatabase(), hash, number)
	return header
}

// GetHeaderByNumber retrieves a block header from the database by number.
func (bc *TxBlockChain) GetHeaderByNumber(number uint64) *protos.TxBlockHeader {
	hash := rawdb.ReadCanonicalHash(bc.GetDatabase(), number)
	return bc.GetHeader(hash, number)
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bc *TxBlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	// Unsubscribe all subscriptions registered from blockchain
	close(bc.quit)
	atomic.StoreInt32(&bc.procInterrupt, 1)

	bc.wg.Wait()

	// Ensure the state of a recent block is also stored to disk before exiting.
	// We're writing three different states to catch different restart scenarios:
	//  - HEAD:     So we don't need to reprocess any blocks in the general case
	//  - HEAD-1:   So we don't do large reorgs if our HEAD becomes an uncle
	//  - HEAD-127: So we have a hard limit on the number of blocks reexecuted
	if !bc.cacheConfig.Disabled {
		triedb := bc.stateCache.TrieDB()

		for _, offset := range []uint64{0, 1, TriesInMemory - 1} {
			if number := bc.CurrentBlock().NumberU64(); number > offset {
				recentBlk := bc.GetBlockByNumber(number - offset)
				recent := recentBlk.(*protos.TxBlock)

				log.Info("Writing cached state to disk", "block", recent.NumberU64(), "hash", recent.Hash(), "root", recent.Root())
				if err := triedb.Commit(recent.Root(), true); err != nil {
					log.Error("Failed to commit recent state trie", "err", err)
				}
			}
		}
		for !bc.triegc.Empty() {
			triedb.Dereference(bc.triegc.PopItem().(common.Hash), common.Hash{})
		}
		if size := triedb.Size(); size != 0 {
			log.Error("Dangling trie nodes after full cleanup")
		}
	}
	log.Info("Blockchain manager stopped")
}

// PostChainEvents iterates over the events generated by a chain insertion and
// posts them into the event feed.
// TODO: Should not expose PostChainEvents. The chain events should be posted in WriteBlock.
func (bc *TxBlockChain) PostChainEvents(events []interface{}, logs []*types.Log) {
	// post event logs for further processing
	if logs != nil {
		bc.logsFeed.Send(logs)
	}

	for _, event := range events {
		switch ev := event.(type) {
		case tcore.ChainEvent:
			bc.chainFeed.Send(ev)

		case tcore.ChainHeadEvent:
			bc.chainHeadFeed.Send(ev)
		}
	}
}

func countTransactions(chain []*protos.TxBlock) (c int) {
	for _, b := range chain {
		c += len(b.Transactions())
	}
	return c
}

// insertStats tracks and reports on block insertion.
type insertStats struct {
	queued, processed, ignored int
	usedGas                    uint64
	lastIndex                  int
	startTime                  mclock.AbsTime
}

// report prints statistics if some number of blocks have been processed
// or more than a few seconds have passed since the last message.
func (st *insertStats) report(chain []*protos.TxBlock, index int, cache common.StorageSize) {
	// Fetch the timings for the batch
	var (
		now     = mclock.Now()
		elapsed = time.Duration(now) - time.Duration(st.startTime)
	)
	// If we're at the last block of the batch or report period reached, log
	if index == len(chain)-1 || elapsed >= StatsReportLimit {
		var (
			end = chain[index]
			txs = countTransactions(chain[st.lastIndex : index+1])
		)
		context := []interface{}{
			"blocks", st.processed, "txs", txs, "mgas", float64(st.usedGas) / 1000000,
			"elapsed", common.PrettyDuration(elapsed), "mgasps", float64(st.usedGas) * 1000 / float64(elapsed),
			"number", end.NumberU64(), "hash", end.Hash(), "cache", cache,
		}
		if st.queued > 0 {
			context = append(context, []interface{}{"queued", st.queued}...)
		}
		if st.ignored > 0 {
			context = append(context, []interface{}{"ignored", st.ignored}...)
		}
		log.Info("Imported new chain segment", context)

		*st = insertStats{startTime: now, lastIndex: index + 1}
	}
}
