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
 * txgenesis_test.go - generate the gensis block test
 * briefs:
 * 	- 2018/9/19. niwei create
 */

package txblockchain

import (
	"testing"

	okcdb "github.com/ok-chain/okchain/core/database"
)

func TestDefaultGenesis(t *testing.T) {
	db, _ := okcdb.NewMemDatabase()

	_, hash, err := SetupTxBlock(db, "")
	if err != nil {
		t.Log(err)
	}

	t.Log(hash)
}

//func TestGenesisToJson(t *testing.T) {
//	memdb, _ := okcdb.NewMemDatabase()
//
//	path := "/Users/lingting.fu/go/src/github.com/ok-chain/okchain/core/txblockchain/data/genesis.json"
//	config, hash, err := SetupTxBlock(memdb, path)
//	if err != nil {
//		t.Log(err)
//	}
//
//	hash1 := rawdb.ReadCanonicalHash(memdb, 0)
//
//	if (hash1 == common.Hash{}) {
//		t.Fatal(hash1)
//	}
//
//	t.Log(hash1)
//
//	bc, err := NewTxBlockChain(config, memdb, GetDefaultCacheConfig())
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	state, err := bc.State()
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	//statedb, _ := state.New(common.Hash{}, state.NewDatabase(memdb))
//	b := state.GetBalance(common.HexToAddress("0x9b175d69a0A3236A1e6992d03FA0be0891D8D023"))
//	t.Log(hash)
//	t.Log(b.String())
//}
