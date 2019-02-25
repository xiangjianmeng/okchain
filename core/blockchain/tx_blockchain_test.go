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

package blockchain

import (
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"
)
import (
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/hexutil"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/protos"
)

import (
	"github.com/ok-chain/okchain/core/blockchain/txblockchain"
	"github.com/stretchr/testify/assert"
)

//func TestBlockChainAllInOne(t *testing.T) {
//
//	memdb, _ := okcdb.NewMemDatabase()
//
//	path := "/Users/lingting.fu/go/src/github.com/ok-chain/okchain/core/txblockchain/data/genesis.json"
//	config, hash, err := SetupTxBlock(memdb, path)
//	if err != nil {
//		log.Error(err)
//	}
//
//	log.Error("SetupTxBlock: ", hash)
//
//	bc, err := NewBlockChain(config, memdb, GetDefaultCacheConfig())
//	if err != nil {
//		log.Error(err)
//	}
//
//	state, err := bc.State()
//	if err != nil {
//		log.Error(err)
//	}
//
//	b := state.GetBalance(common.HexToAddress("0x9b175d69a0A3236A1e6992d03FA0be0891D8D023"))
//	log.Error("GetBalance: ", b.String())
//
//	fixedDate := time.Date(2018, time.September, 20, 0, 0, 0, 0, time.UTC)
//	secs := uint64(fixedDate.Second())
//	nanos := uint64(fixedDate.Nanosecond())
//
//	header := &protos.TxBlockHeader{
//		Version:     1,
//		BlockNumber: 1,
//		PreviousBlockHash:  hash.Bytes(),
//		Timestamp:   &protos.Timestamp{Second: secs, Nanosecond: nanos},
//		StateRoot:   state.IntermediateRoot(false).Bytes(),
//	}
//
//	body := &protos.TxBlockBody{
//		NumberOfMicroBlock: 0,
//	}
//
//	newblock := &protos.TxBlock{
//		Header: header,
//		Body:   body,
//	}
//
//	err = bc.InsertChain(newblock)
//	if err != nil {
//		log.Error("InsertChain Fail: ", err)
//	}
//
//	//nb := bc.GetBlockByNumber(1)
//	nb := bc.GetBlock(newblock.Hash(), 1)
//	if nb == nil {
//		log.Error("GetBlockByNumber Fail: ", err)
//	}
//
//	log.Error("block Hash: ", nb.Hash())
//
//	header1 := &protos.TxBlockHeader{
//		Version:     1,
//		BlockNumber: 2,
//		PreviousBlockHash:  nb.Hash().Bytes(),
//		Timestamp:   &protos.Timestamp{Second: secs, Nanosecond: nanos},
//	}
//
//	body1 := &protos.TxBlockBody{
//		NumberOfMicroBlock: 0,
//	}
//
//	newblock1 := &protos.TxBlock{
//		Header: header1,
//		Body:   body1,
//	}
//
//	err = bc.InsertChain(newblock1)
//	if err != nil {
//		log.Error("inster 1")
//	}
//
//	currentBlock := bc.CurrentBlock()
//	log.Error(currentBlock.Hash(), currentBlock.Header.BlockNumber)
//	log.Error(newblock1.Hash(), newblock1.Header.BlockNumber)
//
//}

func testInsertChain(txChainDB okcdb.Database, withTrx bool) {

	// 2018/09/19. FLT. New a finalblock Chain.
	txBlockChain := getDefaultTxBlockChain(false, testEnvPrefix+"/go/src/github.com/ok-chain/okchain/core/txblockchain/data/genesis.json")

	var block *protos.TxBlock
	var txHeader *protos.TxBlockHeader

	transactions := []*protos.Transaction{
		{
			SenderPubKey: common.HexToAddress(fromAddr).Bytes(),
			ToAddr:       common.HexToAddress(toAddr).Bytes(),
			Amount:       txValue,
			Timestamp:    protos.CreateUtcTimestamp(),
		},
	}
	minerPeer := protos.PeerEndpoint{
		Id:      &protos.PeerID{Name: "Test2"},
		Address: "192.168.168.12",
		Port:    20001,
		Pubkey:  []byte("0x0000000000000000000000000000000000000000000000000000000000000001"),
		Roleid:  "miner",
	}

	txRoot := types.DeriveSha(protos.Transactions(transactions))

	for i := 0; i < 10; i++ {

		crrBlock := txBlockChain.CurrentBlock()

		crrTs := protos.CreateUtcTimestamp()
		txHeader = &protos.TxBlockHeader{
			BlockNumber:       crrBlock.NumberU64() + 1,
			PreviousBlockHash: crrBlock.Hash().Bytes(),
			Version:           0,
			Timestamp:         crrTs,
			Miner:             &minerPeer,
		}

		txBody := protos.TxBlockBody{
			NumberOfMicroBlock: 1,
		}

		if withTrx {
			txHeader.TxRoot = txRoot.Bytes()
			txBody.Transactions = transactions
		}

		block = &protos.TxBlock{Header: txHeader, Body: &txBody}

		err := txBlockChain.InsertBlock(block)
		if err != nil {
			panic(err)
		}

	}

	storedBlock := txBlockChain.GetBlock(block.Hash(), txHeader.BlockNumber)
	fmt.Printf("StoredBlock: BlockNO: %d hexHash: %s %+v \n", storedBlock.NumberU64(), hexutil.Encode(storedBlock.Hash().Bytes()), storedBlock)
	fmt.Printf("SummitBlock: BlockNO: %d hexHash: %s %s \n", block.Header.BlockNumber, hexutil.Encode(block.Hash().Bytes()), block.String())
	//assert.True(t, storedBlock != nil, storedBlock.String())
	//assert.True(t, storedBlock.GetHeader().BlockNumber == txHeader.BlockNumber && txHeader.BlockNumber >= 2, storedBlock.String())
	//assert.True(t, storedBlock.Hash() == txHeader.Hash(), storedBlock.String())
}

func BenchmarkBlockChain_InsertChain_WithRocksDB(b *testing.B) {

	dbPath := testEnvPrefix + "/Desktop/txblk_rocksdb/"
	os.RemoveAll(dbPath)

	txChainDB, err := okcdb.NewRDBDatabase(dbPath, 16, 1024)
	if err != nil {
		log.Errorf("Open TxBlockDB Error: <%s>", err.Error())
	}
	defer func() {
		txChainDB.Close()
	}()

	testInsertChain(txChainDB, true)

}

func BenchmarkBlockChain_InsertChain_WithLevelDB(b *testing.B) {

	dbPath := testEnvPrefix + "/Desktop/txblk_leveldb/"
	os.RemoveAll(dbPath)

	txChainDB, err := okcdb.NewLDBDatabase(dbPath, 16, 1024)
	if err != nil {
		log.Errorf("Open TxBlockDB Error: <%s>", err.Error())
	}
	defer txChainDB.Close()

	testInsertChain(txChainDB, true)

}

func testBlockChain_Consensus_SimpleTrx_InsertChain(t *testing.T, txChain ITxBlockchain) {
	// 1. Origin Balance is (bcf72fc47d4aeec195dba5e0e26cea2b75416ec4, 10000000000000000000) (84eaab7ecc07c333123a9a51976d596496d9e2a0, 1).

	crrStateDB, _ := txChain.State()
	newBalnace := crrStateDB.GetBalance(common.HexToAddress(fromAddr))
	expectBalance := big.NewInt(1000000000000000)
	assert.True(t, newBalnace.Cmp(expectBalance) == 0)
	fmt.Printf("New: %+v, Expected: %+v\n", newBalnace, expectBalance)

	txBlock := getDefaultTxBlock(txChain, txValue, true)
	newBlock := txBlock
	nilBlock := txChain.GetBlockByNumber(newBlock.NumberU64())
	assert.True(t, nilBlock == nil, nilBlock)

	err := txChain.InsertBlock(newBlock)
	assert.True(t, err == nil)

	notnilBlock := txChain.GetBlockByNumber(newBlock.NumberU64())
	assert.True(t, notnilBlock != nil, notnilBlock)

	crrStateDB, _ = txChain.State()
	crrtFromNonce := crrStateDB.GetNonce(common.HexToAddress(fromAddr))
	crrtToNonce := crrStateDB.GetNonce(common.HexToAddress(toAddr))
	fmt.Printf("NewFromNonce: %d, NewToNonce: %d\n", crrtFromNonce, crrtToNonce)
	assert.True(t, crrtFromNonce > 0)

	// 2. After Balance is (0x9b175d69a0A3236A1e6992d03FA0be0891D8D023, 99999899900) (0xD8104E3E6dE811DD0cc07d32cCcE2f4f4B38403a, 101).
	crrStateDB, _ = txChain.State()
	newBalnace = crrStateDB.GetBalance(common.HexToAddress(fromAddr))
	expectBalance = big.NewInt(99999899900)
	assert.True(t, newBalnace.Cmp(expectBalance) == 0)
	fmt.Printf("New: %+v, Expected: %+v\n", newBalnace, expectBalance)

	newBalnace = crrStateDB.GetBalance(common.HexToAddress(toAddr))
	expectBalance = big.NewInt(101)
	assert.True(t, newBalnace.Cmp(expectBalance) == 0)
	fmt.Printf("New: %+v, Expected: %+v\n", newBalnace, expectBalance)
	assert.True(t, txChain.CurrentBlock().NumberU64() == notnilBlock.NumberU64(), "Current Block Incorrect.")

	nextBlock := getDefaultTxBlock(txChain, txValue, true)
	if err = txChain.InsertBlock(nextBlock); err != nil {
		fmt.Println(err)
	}

	crrStateDB, _ = txChain.State()
	newBalnace = crrStateDB.GetBalance(common.HexToAddress(toAddr))
	msg, err := txChain.CurrentBlock().ToReadable()
	expectBalance = big.NewInt(201)
	fmt.Printf("New: %+v, Expected: %+v, %s\n", newBalnace, expectBalance, msg)
	assert.True(t, newBalnace.Cmp(expectBalance) == 0)

	// 3. Call TxBlockChain.Stop()
	txChain.Quit()
	assert.True(t, txChain.GetProcInterrupt() == true)
	txChain.Validator().(*BlockValidator).dsBC.GetDatabase().Close()

	txChain.GetDatabase().Close()

	select {
	case <-time.After(10 * time.Second):
		return
	}

}

func sleep() {

	select {
	case <-time.After(12 * time.Second):
		return
	}
}

func testBlockChain_Consensus_DisorderBlock_InsertChain(t *testing.T, txChain ITxBlockchain) {
	// 1. Create Valid Block
	goodBlk1 := getDefaultTxBlock(txChain, txValue, true)
	err1 := txChain.InsertBlock(goodBlk1)
	goodBlk2 := getDefaultTxBlock(txChain, txValue, true)
	err2 := txChain.InsertBlock(goodBlk2)
	assert.True(t, err2 == nil && err1 == nil)
	goodStateDB, _ := txChain.State()
	oldBalance := goodStateDB.GetBalance(common.HexToAddress(toAddr))
	txChain.GetDatabase().Close()
	txChain.Quit()
	sleep()
	fmt.Println("\n\n\n=====================================================================================================================")

	// 2. Insert disorder block into the other txBlockChain.
	newTxBC := getDefaultTxBlockChain(false, "2")
	err1 = newTxBC.InsertBlock(goodBlk2)
	err2 = newTxBC.InsertBlock(goodBlk1)
	assert.True(t, err1 != nil && err2 == nil)
	sleep()

	// 3. Verify balance.
	crrStateDB, _ := newTxBC.State()
	newBalnace := crrStateDB.GetBalance(common.HexToAddress(toAddr))
	assert.True(t, newBalnace.Cmp(oldBalance) == 0)
	fmt.Println(oldBalance, newBalnace)

	sleep()
	newTxBC.GetDatabase().Close()
}

func TestBlockChain_Consensus_SimpleTrx_InsertChain(t *testing.T) {
	txChain := getDefaultTxBlockChain(false, "1")
	testBlockChain_Consensus_SimpleTrx_InsertChain(t, txChain)

	txCacheChain := getDefaultTxBlockChain(true, "1")
	testBlockChain_Consensus_SimpleTrx_InsertChain(t, txCacheChain)
}

func TestBlockChain_Consensus_DisorderTrx_InsertChain(t *testing.T) {
	txChain := getDefaultTxBlockChain(false, "1")
	testBlockChain_Consensus_DisorderBlock_InsertChain(t, txChain)
}

func testBlockChain_Consensus_ContractTrx_InsertChain(t *testing.T, txChain ITxBlockchain) {

	// 1. Origin Account A:(0x9b175d69a0A3236A1e6992d03FA0be0891D8D023, 100000000000) B:(0xD8104E3E6dE811DD0cc07d32cCcE2f4f4B38403a, 1).
	//    A -- 50000000000 --> B
	block1 := getDefaultTxBlock(txChain, 50000000000, true)
	err := txChain.InsertBlock(block1)
	assert.True(t, err == nil)

	// 2. B Create a Smart Contract C(depositToken, withdrawToken, totalToken, notifyEvent)
	//    Check C is correct(Including Account, Storage)
	transferAmount := uint64(1 << 20)
	contractBlk := getDefaultSmartContractTxBlock(txChain, transferAmount, nil, nil)
	err = txChain.InsertBlock(contractBlk)
	assert.True(t, err == nil)

	trxs := contractBlk.Transactions()
	receipts := txChain.(*TxBlockChain).GetReceiptsByHash(contractBlk.Hash())
	assert.True(t, len(trxs) == len(receipts))

	contractAddress := receipts[0].ContractAddress
	txLogger.Info(contractAddress)

	stateDB2, _ := txChain.State()
	codeHash := stateDB2.GetCodeHash(contractAddress)
	assert.True(t, codeHash != common.Hash{})

	contractBlance := stateDB2.GetBalance(contractAddress)
	expectedBlance := big.NewInt(int64(transferAmount))
	assert.True(t, contractBlance.Cmp(expectedBlance) == 0)

	// 3. A call C.totalToken. TODO: Logs Should not be empty. Case error.
	//    Check receipts & event
	contractCallBlk := getDefaultSmartContractTxBlock(txChain, transferAmount, contractAddress.Bytes(), common.Hex2Bytes("20b13d16"))
	err = txChain.InsertBlock(contractCallBlk)
	callReceipt := txChain.(*TxBlockChain).GetReceiptsByHash(contractCallBlk.Hash())[0]
	assert.True(t, callReceipt.Logs != nil && len(callReceipt.Logs) > 0)

	// 4. A call C.depositToken, and get receipt.
	//    Check receipts & events

	// 5. A call C.withdrawToken, C.totalToken and get receipt.
	//    Check receipts & event

}

func TestBlockChain_Consensus_ContractTrx_InsertChain(t *testing.T) {
	txChain := getDefaultTxBlockChain(false, "1")
	testBlockChain_Consensus_ContractTrx_InsertChain(t, txChain)
}

func TestBlockChain_EmptyBlock(t *testing.T) {

	txChain := getDefaultTxBlockChain(false, "1")
	firstEmptyBlock := getDefaultTxBlock(txChain, 0, false)
	err := txChain.InsertBlock(firstEmptyBlock)
	assert.True(t, err == nil)
}

func TestFindCommonAncestor(t *testing.T) {
	bc := getDefaultTxBlockChain(false, "1")
	b1 := getDefaultTxBlock(bc, txValue, true)
	b2 := getDefaultTxBlock(bc, txValue, true)

	expectedAncestor := bc.Genesis()
	gotAncestor := txblockchain.FindCommonAncestor(bc.GetDatabase(), b1.GetHeader(), b2.GetHeader())

	assert.True(t, expectedAncestor != nil && gotAncestor != nil, "expectedAncestor == nil || gotAncestor == nil")
	assert.True(t, expectedAncestor.NumberU64() == gotAncestor.GetBlockNumber(), "Block Number Wrong")
	assert.True(t, expectedAncestor.Hash() == gotAncestor.Hash(), "Hash Wrong")

}
