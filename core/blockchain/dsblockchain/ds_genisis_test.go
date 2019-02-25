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
	"fmt"
	"testing"
)

import (
	"bytes"
	"os"

	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/rawdb"
	"github.com/ok-chain/okchain/protos"
	"github.com/stretchr/testify/assert"
)

func defaultTestDSBlockHeader() *protos.DSBlockHeader {
	rawDsHeader := protos.DSBlockHeader{
		Version:           0,
		Timestamp:         getFixedUtcTimestamp(),
		PreviousBlockHash: hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		WinnerPubKey:      hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		WinnerNonce:       0,
		BlockNumber:       0,
		PowDifficulty:     0,
		NewLeader:         nil,
	}

	return &rawDsHeader
}

func TestSetupDefaultDSGenesisBlock(t *testing.T) {

	dbDir := "/tmp/unittest/okchain/ds-chaindata"
	os.RemoveAll(dbDir)

	db, err := database.NewLDBDatabase(dbDir, 16, 1024)
	assert.Equal(t, nil, err)

	for i := 0; i < 3; i++ {
		conf, hash, err := SetupDefaultDSGenesisBlock(db)
		fmt.Println(conf)
		fmt.Println(hash)
		fmt.Println(err)

		headHash := rawdb.ReadCanonicalHash(db, 0)
		assert.True(t, bytes.Compare(headHash.Bytes(), config.MainnetDSGenesisHash.Bytes()) == 0)

		storedHash := rawdb.ReadCanonicalHash(db, 0)
		dsblock := GetDsBlock(db, storedHash, 0)
		assert.True(t, dsblock.Hash() == storedHash)
	}
}
