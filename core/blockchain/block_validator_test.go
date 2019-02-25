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
	"os"
	"testing"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/core/blockchain/dsblockchain"
	"github.com/ok-chain/okchain/core/blockchain/txblockchain"
	okcdb "github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/core/vm"
	"github.com/ok-chain/okchain/protos"
	"github.com/stretchr/testify/assert"
)

const (
	fromAddr      = "0xbcf72fc47d4aeec195dba5e0e26cea2b75416ec4" //pubKey if have verify procedure.
	toAddr        = "0x84eaab7ecc07c333123a9a51976d596496d9e2a0" //always an address.
	txValue       = uint64(100)
	testEnvPrefix = "/Users/lingting.fu"
)

func getDefaultTxBlockChain(enableCache bool, pathIdx string) ITxBlockchain {

	txDbPath := testEnvPrefix + "/Desktop/1/txblk_rdb" + pathIdx
	dsDbPath := testEnvPrefix + "/Desktop/1/dsblk_rdb" + pathIdx
	os.RemoveAll(txDbPath)
	os.RemoveAll(dsDbPath)

	txChainDB, err := okcdb.NewRDBDatabase(txDbPath, 16, 1024)
	if err != nil {
		log.Errorf("Open TxBlockDB Error: <%s>", err.Error())
	}

	chainConfig, _, err := txblockchain.SetupTxBlock(txChainDB, testEnvPrefix+"/go/src/github.com/ok-chain/okchain/dev/genesis.json")
	if err != nil {
		panic(err)
	}

	cacheConfig := GetDefaultCacheConfig()
	if enableCache {
		cacheConfig.Disabled = false
	} else {
		cacheConfig.Disabled = true
	}

	txBlockChain, err := NewTxBlockChain(chainConfig, txChainDB, cacheConfig)
	dsChainDB, err := okcdb.NewRDBDatabase(dsDbPath, 16, 1024)
	dsblockchain.SetupDefaultDSGenesisBlock(dsChainDB)
	dsBlockChain, err := NewDSBlockChain(dsChainDB, nil, nil)

	txValidator := NewBlockValidator(chainConfig, txBlockChain, dsBlockChain)
	txBlockChain.SetValidator(txValidator)

	return txBlockChain
}

func getDefaultTxBlock(txBlockChain ITxBlockchain, transferAmount uint64, withTrx bool) *protos.TxBlock {

	dsBlockChain := txBlockChain.Validator().(*BlockValidator).dsBC
	currentBlock := txBlockChain.CurrentBlock()
	preHash := currentBlock.Hash()
	tmpStateDB, _ := txBlockChain.State()

	transactions := []*protos.Transaction{
		{
			SenderPubKey: common.HexToAddress(fromAddr).Bytes(),
			ToAddr:       common.HexToAddress(toAddr).Bytes(),
			Amount:       transferAmount,
			Timestamp:    protos.CreateUtcTimestamp(),
			GasLimit:     10000000,
			GasPrice:     1,
			Nonce:        tmpStateDB.GetNonce(common.HexToAddress(fromAddr)),
		},
	}

	txRoot := types.DeriveSha(protos.Transactions(transactions))
	crrTs := protos.CreateUtcTimestamp()

	minerPeer := protos.PeerEndpoint{
		Id:      &protos.PeerID{Name: "Test2"},
		Address: "192.168.168.12",
		Port:    20001,
		Pubkey:  []byte("0x0000000000000000000000000000000000000000000000000000000000000001"),
		Roleid:  "miner",
	}

	txHeader := &protos.TxBlockHeader{
		BlockNumber:       currentBlock.NumberU64() + 1,
		PreviousBlockHash: preHash.Bytes(),
		GasLimit:          4 * (1 << 30),
		Version:           0,
		Timestamp:         crrTs,
		DSBlockNum:        0,
		DSBlockHash:       dsBlockChain.Genesis().Hash().Bytes(),
		Miner:             &minerPeer,
	}

	txBody := protos.TxBlockBody{
		NumberOfMicroBlock: 1,
	}

	if withTrx {
		txHeader.TxRoot = txRoot.Bytes()
		txHeader.TxNum = uint64(len(transactions))
		txBody.Transactions = transactions
	}

	block := &protos.TxBlock{Header: txHeader, Body: &txBody}
	log.Infof("%+v", block)
	_, _, gasUsed, _ := txBlockChain.Processor().Process(block, tmpStateDB, vm.Config{})
	stateRoot := tmpStateDB.IntermediateRoot(true)
	txHeader.StateRoot = stateRoot.Bytes()
	txHeader.GasUsed = gasUsed
	return block
}

func getDefaultSmartContractTxBlock(txBlockChain ITxBlockchain, transferAmount uint64, contractAddress []byte, funcAddress []byte) *protos.TxBlock {

	dsBlockChain := txBlockChain.Validator().(*BlockValidator).dsBC
	currentBlock := txBlockChain.CurrentBlock()
	preHash := currentBlock.Hash()
	tmpStateDB, _ := txBlockChain.State()

	B := toAddr
	byteCode := common.Hex2Bytes("60806040526000805460a060020a60e060020a0319600160a060020a031990911633171690556103d7806100346000396000f3006080604052600436106100615763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416630ef99c21811461006657806320b13d16146100a5578063274ffe62146100ba578063938307b0146100cf575b600080fd5b34801561007257600080fd5b5061008867ffffffffffffffff600435166100f1565b6040805167ffffffffffffffff9092168252519081900360200190f35b3480156100b157600080fd5b50610088610259565b3480156100c657600080fd5b50610088610281565b3480156100db57600080fd5b5061008867ffffffffffffffff6004351661029e565b3360009081526001602052604081205467ffffffffffffffff908116908316811015610164576040805133815267ffffffffffffffff8516602082015281517f63793a50a2c20026405f2ee45fe11926d77642d4ca386b407da82e400c06a233929181900390910190a160009150610253565b336000818152600160209081526040808320805467ffffffffffffffff19811667ffffffffffffffff9182168a900382161790915583547bffffffffffffffff0000000000000000000000000000000000000000198116740100000000000000000000000000000000000000009182900483168a900383169091021790935580519384529186168383015260609083018190526014908301527f7375636365737320746f207769746864726177200000000000000000000000006080830152517f146a58beba342ed4a52e97ea831586a15f0e30e620ce18a946d54115ef4486d59160a0908290030190a18291505b50919050565b60005474010000000000000000000000000000000000000000900467ffffffffffffffff1690565b3360009081526001602052604090205467ffffffffffffffff1690565b336000818152600160209081526040808320805467ffffffffffffffff8082168801811667ffffffffffffffff1990921691909117909155835474010000000000000000000000000000000000000000808204831688018316027bffffffffffffffff000000000000000000000000000000000000000019909116178455815194855285168482015260609184018290526013918401919091527f7375636365737320746f206465706f736974200000000000000000000000000060808401525190917f146a58beba342ed4a52e97ea831586a15f0e30e620ce18a946d54115ef4486d5919081900360a00190a150503360009081526001602052604090205467ffffffffffffffff16905600a165627a7a72305820d346d028e4952ab3ebc7c0d2a02d35b892937fc848af72f15b137b0d28d420470029")

	transactions := []*protos.Transaction{
		{
			SenderPubKey: common.HexToAddress(B).Bytes(),
			ToAddr:       nil,
			Amount:       transferAmount,
			Timestamp:    protos.CreateUtcTimestamp(),
			GasLimit:     4 * (1 << 20),
			GasPrice:     1,
			Nonce:        tmpStateDB.GetNonce(common.HexToAddress(B)),
			Code:         nil,
			Data:         byteCode,
		},
	}

	if contractAddress != nil {
		transactions[0].ToAddr = contractAddress
	}

	if funcAddress != nil {
		transactions[0].Data = funcAddress
	}

	txRoot := types.DeriveSha(protos.Transactions(transactions))

	crrTs := protos.CreateUtcTimestamp()
	txHeader := &protos.TxBlockHeader{
		BlockNumber:       currentBlock.NumberU64() + 1,
		PreviousBlockHash: preHash.Bytes(),
		GasLimit:          4 * (1 << 30),
		Version:           0,
		Timestamp:         crrTs,
		TxNum:             uint64(len(transactions)),
		TxRoot:            txRoot.Bytes(),
		DSBlockNum:        0,
		DSBlockHash:       dsBlockChain.Genesis().Hash().Bytes(),
	}

	txBody := protos.TxBlockBody{
		NumberOfMicroBlock: 1,
		Transactions:       transactions,
	}
	block := &protos.TxBlock{Header: txHeader, Body: &txBody}
	_, _, gasUsed, _ := txBlockChain.Processor().Process(block, tmpStateDB, vm.Config{})
	stateRoot := tmpStateDB.IntermediateRoot(true)
	txHeader.StateRoot = stateRoot.Bytes()
	txHeader.GasUsed = gasUsed

	return block
}

func TestBlockValidator_VerifyHeader_BadCase(t *testing.T) {
	bc := getDefaultTxBlockChain(false, "1")

	// ErrUnknownAncestor
	badBlk := getDefaultTxBlock(bc, txValue, true)
	header := badBlk.Header
	header.PreviousBlockHash = common.Hex2Bytes("0x00000021")
	log.Infof("%+v", header)
	err := bc.Validator().VerifyHeader(header)
	assert.True(t, err != nil && err == ErrUnknownAncestor)

	// ErrTimestamp
	badBlk = getDefaultTxBlock(bc, txValue, true)
	header = badBlk.Header
	header.Timestamp = nil
	err = bc.Validator().VerifyHeader(header)
	assert.True(t, err != nil && err == ErrTimestamp)

	// ErrFutureBlock
	badBlk = getDefaultTxBlock(bc, txValue, true)
	header = badBlk.Header
	header.BlockNumber = header.BlockNumber + 100
	err = bc.Validator().VerifyHeader(header)
	isFuture, _ := bc.IsFutureBlockErr(err)
	assert.True(t, err != nil && isFuture)

	//// ErrInvalidNumber
	//badBlk = getDefaultTxBlock(bc, txValue)
	//header = badBlk.Header
	//header.BlockNumber = header.BlockNumber - 1
	//err = bc.Validator().VerifyHeader(header)
	//log.Infof(err.Error())
	//assert.True(t, err != nil && err == ErrInvalidNumber)

	// ErrDSBlockNotExist
	badBlk = getDefaultTxBlock(bc, txValue, true)
	header = badBlk.Header
	header.DSBlockHash = common.Hex2Bytes("0x000000")
	err = bc.Validator().VerifyHeader(header)
	assert.True(t, err != nil && err == ErrDSBlockNotExist)

	badBlk = getDefaultTxBlock(bc, txValue, true)
	header = badBlk.Header
	header.GasUsed = header.GasLimit + 1
	err = bc.Validator().VerifyHeader(header)
	assert.True(t, err != nil)
}
