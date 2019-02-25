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

package dsblockchain

import (
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
)

var log = logging.MustGetLogger("DSBlockchain")

// WriteDSBlock serializes a dsblock into the database.
func WriteDsBlock(db database.Putter, dsblock *protos.DSBlock) (err error) {
	if err = WriteDsHeader(db, dsblock.Header); err != nil {
		return err
	}

	if err = WriteDsBody(db, dsblock.Hash(), dsblock.Header.BlockNumber, dsblock.Body); err != nil {
		return err
	}

	return nil
}

// WriteHeader serializes a block header into the database.
func WriteDsHeader(db database.Putter, header *protos.DSBlockHeader) (err error) {
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}
	hash := header.Hash().Bytes()
	encNum := rawdb.EncodeBlockNumber(header.BlockNumber)
	key := rawdb.GetHeaderNumberKey(hash)

	if err = db.Put(key, encNum); err != nil {
		log.Critical("Failed to store hash to number mapping", "err", err)
	}

	key = rawdb.GetFullHeaderKey(header.GetBlockNumber(), hash)
	if err = db.Put(key, data); err != nil {
		log.Critical("Failed to store header", "err", err)
	}
	return nil
}

// WriteBody serializes the body of a block into the database.
func WriteDsBody(db database.Putter, hash common.Hash, number uint64, body *protos.DSBlockBody) (err error) {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		log.Critical("Fail to Encode DSBlockHeader.", "err", err)
		return err
	}
	rawdb.WriteBodyRLP(db, hash, number, data)
	return nil
}

// GetHeader retrieves the block, nil if none found.
func GetDsBlock(db rawdb.DatabaseReader, hash common.Hash, number uint64) *protos.DSBlock {
	header := GetDsHeader(db, hash, number)
	body := GetDsBody(db, hash, number)
	if header == nil || body == nil {
		return nil
	}

	return &protos.DSBlock{Header: header, Body: body}
}

// GetHeader retrieves the block header corresponding to the hash, nil if none found.
func GetDsHeader(db rawdb.DatabaseReader, hash common.Hash, number uint64) *protos.DSBlockHeader {

	data := rawdb.ReadHeaderRLP(db, hash, number)

	if data == nil || len(data) == 0 {
		return nil
	}

	header := new(protos.DSBlockHeader)
	time := new(protos.Timestamp)
	peerend := new(protos.PeerEndpoint)

	header.Timestamp = time
	header.NewLeader = peerend

	if err := rlp.DecodeBytes(data, header); err != nil {
		log.Error("Invalid block header RLP", "hash", hash, "err", err)
		return nil
	}
	return header
}

// GetBody retrieves the block body (transactons, uncles) corresponding to the hash, nil if none found.
func GetDsBody(db rawdb.DatabaseReader, hash common.Hash, number uint64) *protos.DSBlockBody {

	data := rawdb.ReadBodyRLP(db, hash, number)
	if data == nil || len(data) == 0 {
		return nil
	}

	body := new(protos.DSBlockBody)
	if err := rlp.DecodeBytes(data, body); err != nil {
		log.Error("Invalid block body RLP", "hash", hash, "err", err)
		return nil
	}

	return body
}
