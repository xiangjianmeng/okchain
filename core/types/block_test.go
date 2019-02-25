/**
 * Copyright 2015 OKCoin Inc.
 * block_test.go - block 测试
 * 2018/7/18. niwei create
 */

package types

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/core/database"
)

func genblcok() *Block {
	coinbase := common.BytesToAddress([]byte("515669b681730d83458f9a1d87dfe17bda62f9b1"))
	header := Header{}
	header.ParentHash = common.BytesToHash([]byte{})
	header.Coinbase = coinbase
	header.TxHash = EmptyRootHash
	header.Number = new(big.Int).SetInt64(0)
	header.Time = new(big.Int).SetInt64(time.Now().Unix())
	header.Reward = new(big.Int).SetInt64(100)

	return NewBlockWithHeader(&header)
}

// from bcValidBlockTest.json, "SimpleTx"
func TestBlockEncoding(t *testing.T) {
	db, _ := database.NewMemDatabase()
	dposCtx, _ := NewDposContext(db)
	inputBlock := Block{
		header: &Header{
			Validator:   common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"),
			Coinbase:    common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"),
			Root:        common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"),
			Time:        big.NewInt(1426516743),
			DposContext: dposCtx.ToProto(),
		},
	}
	tx1 := NewTransaction(Binary, 0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(10), nil)
	//tx1, _ = tx1.WithSignature(HomesteadSigner{}, common.Hex2Bytes("9bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094f8a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b100"))
	inputBlock.transactions = []*Transaction{tx1}
	inputHash := inputBlock.Hash()
	blockEnc, _ := rlp.EncodeToBytes(extblock{
		Header: inputBlock.header,
		Txs:    inputBlock.transactions,
	})
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Validator", block.Validator(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("Coinbase", block.Coinbase(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("Root", block.Root(), common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"))
	check("Time", block.Time(), big.NewInt(1426516743))
	check("Size", block.Size(), common.StorageSize(len(blockEnc)))
	check("Hash", block.Hash(), inputHash)
	check("len(Transactions)", len(block.Transactions()), 1)
	check("Transactions[0].Hash", block.Transactions()[0].Hash(), tx1.Hash())
	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}
