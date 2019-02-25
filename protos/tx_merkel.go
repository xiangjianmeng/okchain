// Copyright The go-okchain Authors 2019,  All rights reserved.

package protos

import (
	"reflect"

	"github.com/ok-chain/okchain/crypto/merkel"
)

// var emptyRoot = []byte("00000000000000000000000000000000")

//compute TxRoot
func ComputeMerkelRoot(txs Transactions) []byte {
	var txHashes [][]byte

	for _, tx := range txs {
		txHashes = append(txHashes, tx.Hash().Bytes())
	}
	mTree := merkel.NewMerkleTree(txHashes)

	return mTree.RootNode.Data
}

func VerifyMerkelRoot(txs Transactions, root []byte) bool {
	return reflect.DeepEqual(ComputeMerkelRoot(txs), root)
}
