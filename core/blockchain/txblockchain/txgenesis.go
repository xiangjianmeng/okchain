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
 * txgenesis.go - generate the gensis block in db
 * briefs:
 * 	- 2018/9/19. niwei create
 */

package txblockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ok-chain/okchain/common/hexutil"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/protos"

	"github.com/ok-chain/okchain/common"
	params "github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/rawdb"
)

var txLogger = bcLogger

type Genesis struct {
	Config    *params.ChainConfig `json:"config"`
	Version   uint32              `json:"version"`
	Nonce     uint64              `json:"nonce"`
	Timestamp uint64              `json:"timestamp"`
	ExtraData []byte              `json:"extraData"`
	GasLimit  uint64              `json:"gasLimit"   gencodec:"required"`
	// Difficulty *big.Int       `json:"difficulty" gencodec:"required"`
	Mixhash  common.Hash    `json:"mixHash"`
	Coinbase common.Address `json:"coinbase"`
	Alloc    GenesisAlloc   `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number            uint64      `json:"number"`
	PreviousBlockHash common.Hash `json:"parentHash"`
}

type GenesisAlloc map[common.Address]GenesisAccount

type GenesisMismatchError struct {
	Stored, New common.Hash
}

func (e *GenesisMismatchError) Error() string {
	return fmt.Sprintf("database already contains an incompatible genesis block (have %x, new %x)", e.Stored[:8], e.New[:8])
}

type GenesisAccount struct {
	Code    []byte                      `json:"code,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	Balance *big.Int                    `json:"balance" gencodec:"required"`
	Nonce   uint64                      `json:"nonce,omitempty"`
}

//// SetupGenesisBlock writes or updates the genesis block in db.
//// The block that will be used is:
////
////                              genesis == nil       genesis != nil
////                       +----------------------------------------------------
////     db has no genesis |  C1: main-net default  |  C2: genesis
////     db has genesis    |  C3: from DB           |  C4 genesis (if compatible)
////
//// The stored chain configuration will be updated if it is compatible (i.e. does not
//// specify a fork block below the local head block). In case of a conflict, the
//// error is a *params.ConfigCompatError and the new, unwritten config is returned.
////
//// The returned chain configuration is never nil.
////
//// FLT. 2018/09/27. Eth support C1+C2+C3+C4, and we support C2+C3 only.
////
func setupDefaultGenesisBlock(db okcdb.Database) (*params.ChainConfig, common.Hash, error) {
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		genesis := DefaultGenesisBlock()

		block, err := genesis.Commit(db)
		if err != nil {
			return nil, common.Hash{}, err
		} else {
			return genesis.Config, block.Hash(), err
		}
	} else {
		gBlk := ReadBlock(db, stored, 0)
		calHash := gBlk.Hash()
		txLogger.Infof("Read TxGenesisBlock from db, expected hash: %s, actully hash: %s\n", hexutil.Encode(stored[:]), hexutil.Encode(calHash[:]))

		readConf := rawdb.ReadChainConfig(db, stored)
		if gBlk == nil || readConf == nil {
			return nil, stored, errors.New(fmt.Sprintf(
				"TxGenesisBlock load failed or No ChainConfig found in DB, Check why. gBlk: %s, readCnf: %s", gBlk, readConf))
		} else {
			return readConf, stored, nil
		}
	}
}

func setupGenesisBlockByCustom(db okcdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	if genesis == nil || genesis.Config == nil {
		return nil, common.Hash{}, errors.New(fmt.Sprintf("Setup Custom TX genesis fail. genesis: %v", genesis))
	}

	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		blk, err := genesis.Commit(db)
		return genesis.Config, blk.Hash(), err
	} else {
		ghash := genesis.ToBlock(db).Hash()
		txLogger.Infof("Read TxGenesisBlock from db, expected hash: %s, actully hash: %s\n", hexutil.Encode(stored[:]), hexutil.Encode(ghash[:]))

		if ghash != stored {
			return genesis.Config, ghash, &GenesisMismatchError{stored, ghash}
		} else {
			return genesis.Config, ghash, nil
		}
	}
}

func (g *Genesis) configOrDefault(ghash common.Hash) *params.ChainConfig {
	switch {
	case g != nil:
		return g.Config
	case ghash == params.MainnetGenesisHash:
		return params.MainnetChainConfig
	case ghash == params.TestnetGenesisHash:
		return params.TestnetChainConfig
	default:
		return params.AllEthashProtocolChanges
	}
}

func DefaultGenesisBlock() *Genesis {
	return &Genesis{
		Version:   1,
		Config:    params.MainnetChainConfig,
		Nonce:     66,
		ExtraData: hexutil.MustDecode("0x11bbe8db4e347b4e8c937c1c8370e4b5ed33adb3db69cbdb7a38e1e50b1b82fa"),
		GasLimit:  4712388,
	}
}

func (g *Genesis) ToBlock(db okcdb.Database) *protos.TxBlock {
	if db == nil {
		db, _ = okcdb.NewMemDatabase()
	}
	//var addr1 common.Address
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	statedb.Prepare(common.Hash{}, common.Hash{}, uint64(0), 0)
	for addr, account := range g.Alloc {
		txLogger.Debugf("genesis account alloc: %x, %d", addr.Bytes(), account.Balance)
		statedb.AddBalance(addr, account.Balance)
		statedb.SetCode(addr, account.Code)
		statedb.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			statedb.SetState(addr, key, value)
		}
	}

	root := statedb.IntermediateRoot(false)

	statedb.Commit(false)
	statedb.Database().TrieDB().Commit(root, true)

	block := new(protos.TxBlock)

	fixedDate := time.Date(2018, time.September, 20, 0, 0, 0, 0, time.UTC)
	secs := uint64(fixedDate.Second())
	nanos := uint64(fixedDate.Nanosecond())

	fixPeerId := protos.PeerID{Name: "NoPeer"}
	fixPeerLeader := protos.PeerEndpoint{}
	fixPeerLeader.Id = &fixPeerId

	block.Header = &protos.TxBlockHeader{
		Version:           g.Version,
		BlockNumber:       g.Number,
		PreviousBlockHash: common.Hash{}.Bytes(),
		GasLimit:          g.GasLimit,
		StateRoot:         root.Bytes(),
		Timestamp:         &protos.Timestamp{Second: secs, Nanosecond: nanos},
		Miner:             &fixPeerLeader,
	}

	block.Body = &protos.TxBlockBody{
		NumberOfMicroBlock: 0,
	}

	return block
}

func (g *Genesis) Commit(db okcdb.Database) (*protos.TxBlock, error) {
	block := g.ToBlock(db)

	if block.Header.BlockNumber != 0 {
		return nil, fmt.Errorf("can't commit TxGenesisBlock with number > 0")
	}

	config := g.Config
	if config == nil {
		config = params.AllEthashProtocolChanges
	}

	WriteBlock(db, block)
	rawdb.WriteCanonicalHash(db, block.Hash(), 0)
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	rawdb.WriteChainConfig(db, block.Hash(), config)

	// 2018/09/27. Fore. Set & Get When doing commit job of Genesis.
	stordBlock := ReadBlock(db, block.Hash(), 0)
	if stordBlock == nil {
		return stordBlock, errors.New("TxGenesis is not committed into DB.")
	} else {
		return stordBlock, nil
	}
}

func initGenesis(gensisPath string) (*Genesis, error) {
	if len(gensisPath) == 0 {
		return nil, errors.New("Must supply path to genesis JSON file")
	}

	file, err := os.Open(gensisPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	genesis := new(Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		return nil, err
	}

	return genesis, nil
}

func SetupTxBlock(db okcdb.Database, genesisPath string) (*params.ChainConfig, common.Hash, error) {
	if len(genesisPath) == 0 {
		return setupDefaultGenesisBlock(db)
	}
	genesis, err := initGenesis(genesisPath)
	if err != nil {
		return nil, common.Hash{}, err
	}

	blkConfig, gblkHash, err := setupGenesisBlockByCustom(db, genesis)
	if err != nil {
		txLogger.Error(err)
	} else {
		txLogger.Infof("TxGenesisBlock Hash: %s, genesisPath: %s", hexutil.Encode(gblkHash[:]), genesisPath)
	}

	return blkConfig, gblkHash, err
}
