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

//// Copyright 2017 The go-ethereum Authors
//// This file is part of the go-ethereum library.
////
//// The go-ethereum library is free software: you can redistribute it and/or modify
//// it under the terms of the GNU Lesser General Public License as published by
//// the Free Software Foundation, either version 3 of the License, or
//// (at your option) any later version.
////
//// The go-ethereum library is distributed in the hope that it will be useful,
//// but WITHOUT ANY WARRANTY; without even the implied warranty of
//// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
//// GNU Lesser General Public License for more details.
////
//// You should have received a copy of the GNU Lesser General Public License
//// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.
//
package txblockchain

import (
	"time"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/bitutil"
	"github.com/ok-chain/okchain/core/bloombits"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/protos"
)

const (
	// bloomServiceThreads is the number of goroutines used globally by an Ethereum
	// instance to service bloombits lookups for all running filters.
	bloomServiceThreads = 16

	// bloomFilterThreads is the number of goroutines used locally per filter to
	// multiplex requests onto the global servicing goroutines.
	bloomFilterThreads = 3

	// bloomRetrievalBatch is the maximum number of bloom bit retrievals to service
	// in a single batch.
	bloomRetrievalBatch = 16

	// bloomRetrievalWait is the maximum time to wait for enough bloom bit requests
	// to accumulate request an entire batch (avoiding hysteresis).
	bloomRetrievalWait = time.Duration(0)
)

//// 20181030. FLT. TODO: Start up in full node. which might be part of peer server.
//// startBloomHandlers starts a batch of goroutines to accept bloom bit database
//// retrievals from possibly a range of filters and serving the data to satisfy.
//func (eth *Ok) startBloomHandlers() {
//	for i := 0; i < bloomServiceThreads; i++ {
//		go func() {
//			for {
//				select {
//				case <-eth.shutdownChan:
//					return
//
//				case request := <-eth.bloomRequests:
//					task := <-request
//					task.Bitsets = make([][]byte, len(task.Sections))
//					for i, section := range task.Sections {
//						head := core.GetCanonicalHash(eth.chainDb, (section+1)*params.BloomBitsBlocks-1)
//						if compVector, err := core.GetBloomBits(eth.chainDb, task.Bit, section, head); err == nil {
//							if blob, err := bitutil.DecompressBytes(compVector, int(params.BloomBitsBlocks)/8); err == nil {
//								task.Bitsets[i] = blob
//							} else {
//								task.Error = err
//							}
//						} else {
//							task.Error = err
//						}
//					}
//					request <- task
//				}
//			}
//		}()
//	}
//}

const (
	// bloomConfirms is the number of confirmation blocks before a bloom section is
	// considered probably final and its rotated bits are calculated.
	bloomConfirms = 256

	// bloomThrottling is the time to wait between processing two consecutive index
	// sections. It's useful during chain upgrades to prevent disk overload.
	bloomThrottling = 100 * time.Millisecond
)

// BloomIndexer implements a core.ChainIndexer, building up a rotated bloom bits index
// for the Ethereum header bloom filters, permitting blazing fast filtering.
type BloomIndexer struct {
	size uint64 // section size to generate bloombits for

	db  okcdb.Database       // database instance to write index data and metadata into
	gen *bloombits.Generator // generator to rotate the bloom bits crating the bloom index

	section uint64      // Section is the section number being processed currently
	head    common.Hash // Head is the hash of the last header processed
}

// NewBloomIndexer returns a chain indexer that generates bloom bits data for the
// canonical chain for fast logs filtering.
func NewBloomIndexer(db okcdb.Database, size uint64) *ChainIndexer {
	backend := &BloomIndexer{
		db:   db,
		size: size,
	}
	table := okcdb.NewTable(db, string(rawdb.BloomBitsIndexPrefix))

	return NewChainIndexer(db, table, backend, size, bloomConfirms, bloomThrottling, "bloombits")
}

// Reset implements core.ChainIndexerBackend, starting a new bloombits index
// section.
func (b *BloomIndexer) Reset(section uint64, lastSectionHead common.Hash) error {
	gen, err := bloombits.NewGenerator(uint(b.size))
	b.gen, b.section, b.head = gen, section, common.Hash{}
	return err
}

// Process implements core.ChainIndexerBackend, adding a new header's bloom into
// the index.
func (b *BloomIndexer) Process(header *protos.TxBlockHeader) {
	// TODO: add Bloom in future. FLT. 2018/11/05
	//bloom := types.BytesToBloom(header.Bloom)
	bloom := types.BytesToBloom(header.Hash().Bytes())
	b.gen.AddBloom(uint(header.BlockNumber-b.section*b.size), bloom)
	b.head = header.Hash()
}

// Commit implements core.ChainIndexerBackend, finalizing the bloom section and
// writing it out into the database.
func (b *BloomIndexer) Commit() error {
	batch := b.db.NewBatch()

	for i := 0; i < types.BloomBitLength; i++ {
		bits, err := b.gen.Bitset(uint(i))
		if err != nil {
			return err
		}

		rawdb.WriteBloomBits(batch, uint(i), b.section, b.head, bitutil.CompressBytes(bits))
	}
	return batch.Write()
}
