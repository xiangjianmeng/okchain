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
	//"github.com/ethereum/go-ethereum/core/types"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/protos"
)

// WriteTxLookupEntries stores a positional metadata for every transaction from
// a block, enabling hash based transaction and receipt lookups.
func WriteTxLookupEntries(db rawdb.DatabaseWriter, block *protos.TxBlock) {
	for i, tx := range block.GetBody().GetTransactions() {
		entry := rawdb.TxLookupEntry{
			BlockHash:  block.Hash(),
			BlockIndex: block.GetHeader().GetBlockNumber(),
			Index:      uint64(i),
		}
		data, err := rlp.EncodeToBytes(entry)
		if err != nil {
			log.Critical("Failed to encode transaction lookup entry", "err", err)
		}
		if err := db.Put(rawdb.GetTxLookupKey(tx.Hash().Bytes()), data); err != nil {
			log.Critical("Failed to store transaction lookup entry", "err", err)
		}
	}
}

// ReadTransaction retrieves a specific transaction from the database, along with
// its added positional metadata.
func ReadTransaction(db rawdb.DatabaseReader, hash common.Hash) (*protos.Transaction, common.Hash, uint64, uint64) {
	blockHash, blockNumber, txIndex := rawdb.ReadTxLookupEntry(db, hash)
	if blockHash == (common.Hash{}) {
		return nil, common.Hash{}, 0, 0
	}
	body := ReadBody(db, blockHash, blockNumber)
	if body == nil || len(body.Transactions) <= int(txIndex) {
		log.Error("Transaction referenced missing", "number", blockNumber, "hash", blockHash, "index", txIndex)
		return nil, common.Hash{}, 0, 0
	}
	return body.Transactions[txIndex], blockHash, blockNumber, txIndex
}
