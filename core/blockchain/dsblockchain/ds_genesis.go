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

/*
Author : lingting.fu@okcoin.com

*/

package dsblockchain

import (
	"bytes"
	"time"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/protos"
)

var dsLogger = log

// createFixUtcTimestamp returns a fixed in UTC
func getFixedUtcTimestamp() *protos.Timestamp {
	fixedDate := time.Date(2018, time.September, 20, 0, 0, 0, 0, time.UTC)
	return &(protos.Timestamp{Second: uint64(fixedDate.Unix()), Nanosecond: uint64(fixedDate.Nanosecond())})
}

// DSGenesis specifies the header fields, state of a genesis block. It also defines hard
// fork switch-over blocks through the chain configuration.
type DSGenesis struct {
	Config *config.ChainConfig `json:"config"`
}

// Commit writes the block and state of a genesis specification to the database.
// The block is committed as the canonical head block.
func (g *DSGenesis) Commit(db database.Database) (*protos.DSBlock, error) {
	block := g.ToBlock(db)
	err := WriteDsBlock(db, block)

	writtenHash := block.Hash()
	log.Debugf("Committed Block Hash: %s", hexutil.Encode(writtenHash[:]))

	rawdb.WriteCanonicalHash(db, block.Hash(), block.GetHeader().BlockNumber)
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())

	return block, err
}

// ToBlock creates the genesis block and writes state of a genesis specification
// to the given database (or discards it if nil).
func (g *DSGenesis) ToBlock(db database.Database) *protos.DSBlock {

	fixPeerId := protos.PeerID{Name: "NoPeer"}
	fixPeerLeader := protos.PeerEndpoint{}
	fixPeerLeader.Id = &fixPeerId

	rawDsHeader := protos.DSBlockHeader{
		Version:           0,
		Timestamp:         getFixedUtcTimestamp(),
		PreviousBlockHash: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		WinnerPubKey:      hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		WinnerNonce:       0,
		BlockNumber:       0,
		PowDifficulty:     0,
		NewLeader:         &fixPeerLeader,
		Miner:             &fixPeerLeader,
	}

	dsBody := protos.DSBlockBody{}

	block := protos.DSBlock{
		Header: &rawDsHeader,
		Body:   &dsBody,
	}
	return &block
}

// DefaultGenesisBlock returns the OKChain main net ds genesis block.
func DefaultDSGenesisBlock() *DSGenesis {
	return &DSGenesis{
		Config: config.MainnetDSChainConfig,
	}
}

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                              genesis == nil       genesis != nil
//                       +----------------------------------------------------
//     db has no genesis |  C1: main-net default  |  C2: genesis
//     db has genesis    |  C3: from DB           |  C4 genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *config.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
// 2018.09.28. Flt. Support C1/C3 only.
func SetupDefaultDSGenesisBlock(db database.Database) (*config.ChainConfig, common.Hash, error) {

	genesis := DefaultDSGenesisBlock()
	storedHash := rawdb.ReadCanonicalHash(db, 0)

	if (storedHash == common.Hash{}) {

		dsblock, err := genesis.Commit(db)
		return genesis.Config, dsblock.Hash(), err

	} else {
		dsblock := GetDsBlock(db, storedHash, 0)
		calHash := dsblock.Hash()
		dsLogger.Infof("Read ds genesis block from db, expected hash: %s, actully hash: %s\n", hexutil.Encode(storedHash[:]), hexutil.Encode(calHash[:]))

		if dsblock != nil {

			if bytes.Compare(calHash[:], storedHash[:]) != 0 || bytes.Compare(storedHash[:], config.MainnetDSGenesisHash[:]) != 0 {
				dsLogger.Error("DSGenisis hash in DB is different from DefaultGenesisBlock, " +
					"strongly recommend u CHECK WHY and cleanup old file to fix this problem.")
				return genesis.Config, dsblock.Hash(), &core.GenesisMismatchError{Stored: config.MainnetDSGenesisHash, New: calHash}
			}

			return genesis.Config, dsblock.Hash(), nil

		} else {
			dsLogger.Infof("No genesis block found in db(dsblock == nil), expected hash: %s\n", hexutil.Encode(storedHash[:]))
			return genesis.Config, common.Hash{}, nil
		}
	}
}
