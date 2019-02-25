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
	"bytes"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/protos"
)

var log = bcLogger

// ReadHeader retrieves the block header corresponding to the hash.
func ReadHeader(db rawdb.DatabaseReader, hash common.Hash, number uint64) *protos.TxBlockHeader {
	data := rawdb.ReadHeaderRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	header := new(protos.TxBlockHeader)
	if err := rlp.Decode(bytes.NewReader(data), header); err != nil {
		log.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	return header
}

// WriteHeader stores a block header into the database and also stores the hash-
// to-number mapping.
func WriteHeader(db rawdb.DatabaseWriter, header *protos.TxBlockHeader) {
	// Write the hash -> number mapping
	var (
		hash    = header.Hash().Bytes()
		number  = header.BlockNumber
		encoded = rawdb.EncodeBlockNumber(number)
	)
	key := rawdb.GetHeaderNumberKey(hash)
	if err := db.Put(key, encoded); err != nil {
		log.Critical("Failed to store hash to number mapping", "err", err)
	}
	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		log.Critical("Failed to RLP encode header", "err", err)
	}
	key = rawdb.GetFullHeaderKey(number, hash)
	if err := db.Put(key, data); err != nil {
		log.Critical("Failed to store header", "err", err)
	}
}

// ReadBody retrieves the block body corresponding to the hash.
func ReadBody(db rawdb.DatabaseReader, hash common.Hash, number uint64) *protos.TxBlockBody {
	data := rawdb.ReadBodyRLP(db, hash, number)
	if len(data) == 0 {
		return nil
	}
	body := new(protos.TxBlockBody)
	if err := rlp.Decode(bytes.NewReader(data), body); err != nil {
		log.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}
	return body
}

// WriteBody storea a block body into the database.
func WriteBody(db rawdb.DatabaseWriter, hash common.Hash, number uint64, body *protos.TxBlockBody) {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		log.Critical("Failed to RLP encode body", "err", err)
	}
	rawdb.WriteBodyRLP(db, hash, number, data)
}

// ReadBlock retrieves an entire block corresponding to the hash, assembling it
// back from the stored header and body. If either the header or body could not
// be retrieved nil is returned.
//
// Note, due to concurrent download of header and block body the header and thus
// canonical hash can be stored in the database but the body data not (yet).
func ReadBlock(db rawdb.DatabaseReader, hash common.Hash, number uint64) *protos.TxBlock {
	header := ReadHeader(db, hash, number)
	if header == nil {
		return nil
	}
	body := ReadBody(db, hash, number)
	if body == nil {
		return nil
	}

	ptrBlk := &protos.TxBlock{
		Header: header,
		Body:   body,
	}

	return ptrBlk
}

// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db rawdb.DatabaseWriter, block *protos.TxBlock) {
	WriteBody(db, block.Hash(), block.Header.BlockNumber, block.GetBody())
	WriteHeader(db, block.GetHeader())
}

// FindCommonAncestor returns the last common ancestor of two block headers
func FindCommonAncestor(db rawdb.DatabaseReader, a, b *protos.TxBlockHeader) *protos.TxBlockHeader {
	for bn := b.GetBlockNumber(); a.GetBlockNumber() > bn; {
		previousBlockHash := common.BytesToHash(a.GetPreviousBlockHash())
		a = ReadHeader(db, previousBlockHash, a.GetBlockNumber()-1)
		if a == nil {
			return nil
		}
	}
	for an := a.GetBlockNumber(); an < b.GetBlockNumber(); {
		previousBlockHash := common.BytesToHash(b.GetPreviousBlockHash())
		b = ReadHeader(db, previousBlockHash, b.GetBlockNumber()-1)
		if b == nil {
			return nil
		}
	}

	for a.Hash() != b.Hash() {

		aPreviousBlockHash := common.BytesToHash(a.GetPreviousBlockHash())
		a = ReadHeader(db, aPreviousBlockHash, a.GetBlockNumber()-1)
		if a == nil {
			return nil
		}

		bPreviousBlockHash := common.BytesToHash(b.GetPreviousBlockHash())
		b = ReadHeader(db, bPreviousBlockHash, b.GetBlockNumber()-1)
		if b == nil {
			return nil
		}
	}
	return a
}
