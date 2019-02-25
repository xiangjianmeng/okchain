// Copyright The go-okchain Authors 2018,  All rights reserved.

package protos

import (
	"fmt"
	"testing"
)

func TestTxBlockHash(t *testing.T) {
	b := TxBlock{Header: &TxBlockHeader{Version: 1, PreviousBlockHash: []byte("0000"), BlockNumber: uint64(1)}}
	hash := b.Hash()
	fmt.Println(hash)
	for i := 0; i < 1000; i++ {
		if hash != b.Hash() {
			t.Fatal("Hash must be the same always.")
		}
	}
	fmt.Println(b.Header.Hash())
	for i := 0; i < 1000; i++ {
		if hash != b.Header.Hash() {
			t.Fatal("Hash must be the same always.")
		}
	}
}

func TestDsBlockHash(t *testing.T) {
	b := DSBlock{Header: &DSBlockHeader{}}
	hash := b.Hash()
	for i := 0; i < 1000; i++ {
		if hash != b.Hash() {
			t.Fatal("Hash must be the same always.")
		}
	}
	for i := 0; i < 1000; i++ {
		if hash != b.Header.Hash() {
			t.Fatal("Hash must be the same always.")
		}
	}
}

func TestTxBlockToJsonString(t *testing.T) {
	// h := TxBlockHeader{}
	// b := TxBlockBody{}
	// blk := TxBlock{}
	// blk.Header = &h
	// blk.Body = &b

	// s, err := blk.ToReadable()
	// fmt.Println(s)
	// assert.True(t, err == nil)

	// h.BlockNumber = 1
	// h.PreviousBlockHash, err = hexutil.Decode("0x1000000000000000000000000000000000000000200000000000000000000000")
	// // h.ConsensusID = 100
	// h.Coinbase, err = hexutil.Decode("0x1000000000000000000000000000000000000000200000000000000000000001")

	// b.Signature, err = hexutil.Decode("0x1000000000000000000000000000000000000000200000000000000000050001")
	// b.ConsensusMetadata, err = hexutil.Decode("0x1000000000000000000000000000000000000000200000000000000030050001")

	// mbHash0, err := hexutil.Decode("0x1000000000000000000000000000000000000000200000000000000000050001")
	// mbHash1, err := hexutil.Decode("0x1000000000000000000000000000000000000000200000000000000000050005")

	// b.MicroBlockHashes = append(b.MicroBlockHashes, mbHash0)
	// b.MicroBlockHashes = append(b.MicroBlockHashes, mbHash1)

	// s, err = blk.ToReadable()
	// fmt.Println(s)
	// assert.True(t, err == nil)

}
