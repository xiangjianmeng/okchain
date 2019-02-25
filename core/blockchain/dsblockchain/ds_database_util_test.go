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
	"os"
	"sync/atomic"
	"testing"
)

import (
	"fmt"

	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/protos"
	"github.com/stretchr/testify/assert"
)

func getDefaultDsBlock() *protos.DSBlock {
	rawDsHeader := protos.DSBlockHeader{
		Version:           100,
		PreviousBlockHash: hexutil.MustDecode("0x0000001111111100000000000000000000000000000000000000000000000000"),
		WinnerPubKey:      hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		WinnerNonce:       0,
		BlockNumber:       12,
		PowDifficulty:     0,
	}
	rawDsHeader.NewLeader = new(protos.PeerEndpoint)
	rawDsHeader.NewLeader.Id = new(protos.PeerID)
	rawDsHeader.Timestamp = new(protos.Timestamp)
	rawDsHeader.Miner = rawDsHeader.NewLeader
	dsHeader := rawDsHeader
	hash := dsHeader.Hash()
	h := atomic.Value{}
	h.Store(hash)

	dsBody := protos.DSBlockBody{
		ShardingNodes: []*protos.PeerEndpoint{rawDsHeader.NewLeader},
		Committee:     []*protos.PeerEndpoint{rawDsHeader.NewLeader},
	}

	dsblock := protos.DSBlock{}
	dsblock.Header = &dsHeader
	dsblock.Body = &dsBody

	return &dsblock
}

func TestReadWriteDsBlock(t *testing.T) {
	/*
		var a int64 = 500000
		data,err:=rlp.EncodeToBytes(a)
		fmt.Println(data,err)
		var b int64
		err=rlp.DecodeBytes(data,&b)
		fmt.Println(b,err)
		return*/
	fmt.Println("hello?")
	dbDir := "chaindata"
	os.RemoveAll(dbDir)
	defer os.RemoveAll(dbDir)
	db, err := database.NewRDBDatabase(dbDir, 16, 1024)
	assert.Equal(t, nil, err)

	dsblock := getDefaultDsBlock()
	err = WriteDsBlock(db, dsblock)
	assert.Equal(t, nil, err)
	block2 := GetDsBlock(db, dsblock.Hash(), dsblock.GetHeader().BlockNumber)

	fmt.Println("??")
	fmt.Println(block2.NumberU64())
	db.Close()
}

func TestPartialWrite(t *testing.T) {

	db, _ := database.NewMemDatabase()
	dsblock := getDefaultDsBlock()
	emptyHeader := GetDsHeader(db, dsblock.Hash(), dsblock.NumberU64())
	assert.True(t, emptyHeader == nil)

	err := WriteDsHeader(db, dsblock.GetHeader())
	assert.True(t, err == nil)

	partitialBlk := GetDsBlock(db, dsblock.Hash(), dsblock.NumberU64())
	assert.True(t, partitialBlk == nil)
}

func TestErrorRLPEncode(t *testing.T) {

	db, _ := database.NewMemDatabase()
	dsblock := getDefaultDsBlock()
	dsblock.GetHeader().Timestamp = nil

	err := WriteDsBlock(db, dsblock)
	assert.True(t, err == nil)

	errblock := GetDsBlock(db, dsblock.Hash(), dsblock.NumberU64())
	assert.True(t, errblock == nil)

}
