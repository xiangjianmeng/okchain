// Copyright The go-okchain Authors 2018,  All rights reserved.

package blockchain

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/core/blockchain/dsblockchain"
	"github.com/ok-chain/okchain/core/consensus/mine/vrf"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/crypto/multibls"
	"github.com/ok-chain/okchain/protos"
	"github.com/stretchr/testify/assert"
)

func TestDSBlockChain_ValidateBlock(t *testing.T) {
	dbDir := "chaindata"
	os.RemoveAll(dbDir)
	defer os.RemoveAll(dbDir)
	db, err := database.NewRDBDatabase(dbDir, 16, 1024)
	_, _, err = dsblockchain.SetupDefaultDSGenesisBlock(db)
	dsbc, err := NewDSBlockChain(db, nil, nil)

	// 1. valid block
	dsblock := buildDSblock(dsbc)
	err = dsbc.ValidateBlock(dsblock)
	assert.True(t, err == nil)
	dsbc.InsertBlock(dsblock)

	// 2. ErrUnknownAncestor
	dsblock.Header.BlockNumber = dsblock.NumberU64() + 100
	err = dsbc.ValidateBlock(dsblock)
	assert.True(t, err == ErrUnknownAncestor, err.Error())

	// 3. ErrKnownBlock
	dsblock2 := buildDSblock(dsbc)
	err = dsbc.ValidateBlock(dsblock2)
	assert.True(t, err == ErrKnownBlock, err.Error())

	// 4. ErrTimestamp
	dsblock.Header.Timestamp = nil
	err = dsbc.ValidateBlock(dsblock)
	assert.True(t, err == ErrTimestamp)

	// 5. Other GetXXX func
	genesis := dsbc.GetBlockByNumber(0)
	assert.True(t, genesis != nil && genesis.NumberU64() == dsbc.Genesis().NumberU64())

	notExist := dsbc.GetBlockByNumber(100)
	assert.True(t, notExist == nil)

	dsBlk := dsbc.getDSBlock(genesis)
	fmt.Printf("1.0: %+v", dsBlk)
	assert.True(t, dsBlk != nil)

	errBlk := dsbc.getDSBlock(nil)
	fmt.Printf("1.1: %+v", errBlk)
	assert.True(t, errBlk == nil)
}

func TestDSBlockChain_InsertChain(t *testing.T) {
	dbDir := "chaindata"
	os.RemoveAll(dbDir)
	defer os.RemoveAll(dbDir)
	// Case0. Initialize.
	db, err := database.NewRDBDatabase(dbDir, 16, 1024)
	assert.Equal(t, nil, err)

	// Case0.0 Genesis not inited yet. Fail to create DSBlockchain.
	badBC, err := NewDSBlockChain(db, nil, nil)
	assert.True(t, badBC == nil && err == ErrNoGenesis)

	// Case0.1 Genesis inited. Success to create DSBlockchain.
	config, hash, err := dsblockchain.SetupDefaultDSGenesisBlock(db)
	assert.True(t, config != nil && err == nil && hash != common.Hash{}, hash)
	fmt.Println(hexutil.Encode(hash.Bytes()))
	dsbc, err := NewDSBlockChain(db, nil, nil)
	assert.True(t, dsbc != nil && err == nil, "new dsblockchain end")
	assert.True(t, dsbc.Genesis() != nil)

	// Case1. Valid Block.
	dsblock := buildDSblock(dsbc)
	dsblock.Header.PreviousBlockHash = hash.Bytes()
	blocks := []protos.IBlock{dsblock}
	idx, err := dsbc.InsertChain(blocks)
	if err != nil {
		fmt.Println(err.Error())
	}
	assert.True(t, idx == 0 && err == nil)

	dsblock2 := buildDSblock(dsbc)
	dsblock2.GetHeader().PreviousBlockHash = dsblock.Hash().Bytes()
	dsblock2.GetHeader().BlockNumber = dsblock.NumberU64() + 1
	dsblock3 := dsblock2

	// Case2. dsblock3 is an invalid block.
	badBlocks := []protos.IBlock{dsblock2, dsblock3}
	idx, err = dsbc.InsertChain(badBlocks)
	assert.True(t, idx == 0 && err != nil, err.Error())

	// case2.1 Insert A single valid block.
	if err = dsbc.InsertBlock(dsblock2); err != nil {
		fmt.Println(err.Error())
	}
	maxNO := dsbc.CurrentBlock().NumberU64()
	assert.True(t, maxNO == 2, fmt.Sprintf("%d", maxNO))

	// Case3. Future block
	futureBlock := buildDSblock(dsbc)
	futureBlock.GetHeader().BlockNumber = 100
	fBlocks := []protos.IBlock{futureBlock}
	idx, err = dsbc.InsertChain(fBlocks)
	if err != nil {
		fmt.Printf("Err: %s, Idx: %d", err.Error(), idx)
	}
	assert.True(t, idx == 0 && err != nil)

	// Case4. Wait for a ProcFutureBlock
	time.Sleep(6 * time.Second)

	// Case5. Quit
	dsbc.Quit()
	time.Sleep(6 * time.Second)

}

func buildBLSSigner() (*protos.BLSSigner, error) {
	priv := &multibls.PriKey{}
	msgSigner := protos.BLSSigner{}
	miner, err := vrfminer.NewVrfMiner("1234567890")
	privKey, _ := miner.GetKeyPairs()

	priv.SetHexString(privKey)
	msgSigner.SetKey(priv)

	return &msgSigner, err
}

func buildDSblock(bc *DSBlockChain) *protos.DSBlock {
	dsPeer := protos.PeerEndpoint{
		Id:      &protos.PeerID{Name: "Test"},
		Address: "192.168.168.12",
		Port:    20000,
		Pubkey:  []byte("0x0000000000000000000000000000000000000000000000000000000000000000"),
		Roleid:  "ds",
	}

	minerPeer := protos.PeerEndpoint{
		Id:      &protos.PeerID{Name: "Test2"},
		Address: "192.168.168.12",
		Port:    20001,
		Pubkey:  []byte("0x0000000000000000000000000000000000000000000000000000000000000001"),
		Roleid:  "miner",
	}

	dsHeader := protos.DSBlockHeader{
		Version:           100,
		PreviousBlockHash: bc.Genesis().Hash().Bytes(),
		WinnerPubKey:      hexutil.MustDecode("0x0000000000000000000000000000000000000000000000000000000000000000"),
		WinnerNonce:       0,
		BlockNumber:       1,
		PowDifficulty:     0,
		Timestamp:         protos.CreateUtcTimestamp(),
		NewLeader:         &dsPeer,
		Miner:             &dsPeer,
	}

	dsBody := protos.DSBlockBody{
		ShardingNodes: []*protos.PeerEndpoint{&minerPeer},
		Committee:     []*protos.PeerEndpoint{&dsPeer},
	}

	dsblock := protos.DSBlock{Header: &dsHeader, Body: &dsBody}
	return &dsblock
}

func TestHex2Byte(t *testing.T) {
	s := "0x636f696e626173653a3139322e3136382e3136382e35323a3135303131"
	b := hexutil.MustDecode(s)
	assert.True(t, b != nil)
	fmt.Printf("s: %s, b: %+v, b2h: %s", s, b, hexutil.Encode(b))

}
