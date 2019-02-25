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

package txblockchain

import (
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/protos"
)

// LightChain encapsulates functions required to synchronise a light chain.
type TxLightChain interface {
	// HasHeader verifies a header's presence in the local chain.
	HasHeader(common.Hash, uint64) bool

	// GetHeaderByHash retrieves a header from the local chain.
	GetHeaderByHash(common.Hash) *protos.TxBlockHeader

	// CurrentHeader retrieves the head header from the local chain.
	CurrentHeader() *protos.TxBlockHeader

	// GetTd returns the total difficulty of a local block.
	//GetTd(common.Hash, uint64) *big.Int

	// InsertHeaderChain inserts a batch of headers into the local chain.
	InsertHeaderChain([]*protos.TxBlockHeader, int) (int, error)

	// Rollback removes a few recently added elements from the local chain.
	Rollback([]common.Hash)
}

// TxBlockChain encapsulates functions required to sync a (full or fast) blockchain.
type TxBlockChain interface {
	TxLightChain

	// HasBlock verifies a block's presence in the local chain.
	HasBlock(common.Hash, uint64) bool

	// GetBlockByHash retrieves a block from the local chain.
	GetBlockByHash(common.Hash) *protos.TxBlock

	// CurrentBlock retrieves the head block from the local chain.
	CurrentBlock() *protos.TxBlock

	// CurrentFastBlock retrieves the head fast block from the local chain.
	CurrentFastBlock() *protos.TxBlock

	// FastSyncCommitHead directly commits the head block to a certain entity.
	FastSyncCommitHead(common.Hash) error

	// InsertChain inserts a batch of blocks into the local chain.
	InsertChain(blocks protos.TxBlocks) (int, error)

	// InsertReceiptChain inserts a batch of receipts into the local chain.
	//InsertReceiptChain(types.Blocks, []types.Receipts) (int, error)
}
