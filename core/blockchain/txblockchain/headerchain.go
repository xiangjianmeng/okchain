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

/**
 * Copyright 2015 OKCoin Inc.
 * headerchain.go - headers process
 * briefs:
 * 	- 2018/9/18. niwei create
 */

package txblockchain

import (
	"sync/atomic"

	"github.com/hashicorp/golang-lru"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/protos"
)

const (
	headerCacheLimit = 512
	tdCacheLimit     = 1024
	numberCacheLimit = 2048
)

type HeaderChain struct {
	config *config.ChainConfig

	chainDb       okcdb.Database
	genesisHeader *protos.TxBlockHeader

	currentHeader     atomic.Value
	currentHeaderHash common.Hash

	procInterrupt func() bool

	headerCache *lru.Cache
	numberCache *lru.Cache
	//rand   *mrand.Rand
}

func NewHeaderChain(chainDb okcdb.Database, config *config.ChainConfig, procInterrupt func() bool) (*HeaderChain, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	numberCache, _ := lru.New(numberCacheLimit)

	hc := &HeaderChain{
		config:        config,
		chainDb:       chainDb,
		headerCache:   headerCache,
		numberCache:   numberCache,
		procInterrupt: procInterrupt,
	}

	hc.genesisHeader = hc.GetHeaderByNumber(0)
	if hc.genesisHeader == nil {
		// TODO. 20181205. return Blockchain.ErrNoGenesis
		return nil, nil
	}

	hc.currentHeader.Store(hc.genesisHeader)
	if head := core.GetHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			hc.currentHeader.Store(chead)
		}
	}
	hc.currentHeaderHash = hc.CurrentHeader().Hash()

	return hc, nil
}

func (hc *HeaderChain) GetHeaderByNumber(number uint64) *protos.TxBlockHeader {
	hash := rawdb.ReadCanonicalHash(hc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash, number)
}

func (hc *HeaderChain) GetHeader(hash common.Hash, number uint64) *protos.TxBlockHeader {
	if header, ok := hc.headerCache.Get(hash); ok {
		return header.(*protos.TxBlockHeader)
	}

	header := GetHeader(hc.chainDb, hash, number)
	if header == nil {
		return nil
	}

	hc.headerCache.Add(hash, header)
	return header
}

func (hc *HeaderChain) SetCurrentHeader(head *protos.TxBlockHeader) {
	rawdb.WriteHeadHeaderHash(hc.chainDb, head.Hash())
	hc.currentHeader.Store(head)
	hc.currentHeaderHash = head.Hash()
}

func (hc *HeaderChain) GetHeaderByHash(hash common.Hash) *protos.TxBlockHeader {
	return hc.GetHeader(hash, hc.GetBlockNumber(hash))
}

func (hc *HeaderChain) GetBlockNumber(hash common.Hash) uint64 {
	if cached, ok := hc.numberCache.Get(hash); ok {
		return cached.(uint64)
	}
	number := GetBlockNumber(hc.chainDb, hash)
	if number == nil {
		return missingNumber
	}

	if *number != missingNumber {
		hc.numberCache.Add(hash, *number)
	}
	return *number
}

func (hc *HeaderChain) CurrentHeader() *protos.TxBlockHeader {
	return hc.currentHeader.Load().(*protos.TxBlockHeader)
}

// HasHeader checks if a block header is present in the database or not.
func (hc *HeaderChain) HasHeader(hash common.Hash, number uint64) bool {
	if hc.numberCache.Contains(hash) || hc.headerCache.Contains(hash) {
		return true
	}
	ok, _ := hc.chainDb.Has(rawdb.GetHeaderKey(hash, number))
	return ok
}

// WriteHeader writes a header into the local chain, given that its parent is
// already known. If the total difficulty of the newly inserted header becomes
// greater than the current known TD, the canonical chain is re-routed.
//
// Note: This method is not concurrent-safe with inserting blocks simultaneously
// into the chain, as side effects caused by reorganisations cannot be emulated
// without the real blocks. Hence, writing headers directly should only be done
// in two scenarios: pure-header mode of operation (light clients), or properly
// separated header/block phases (non-archive clients).
func (hc *HeaderChain) WriteHeader(header *protos.TxBlockHeader) (err error) {

	hash := header.Hash()

	WriteHeader(hc.chainDb, header)
	rawdb.WriteCanonicalHash(hc.chainDb, hash, header.BlockNumber)
	rawdb.WriteHeadHeaderHash(hc.chainDb, hash)

	hc.headerCache.Add(hash, header)
	hc.numberCache.Add(hash, header.BlockNumber)

	return nil
}
