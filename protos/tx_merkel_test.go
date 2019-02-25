// Copyright The go-okchain Authors 2019,  All rights reserved.

package protos

import (
	"testing"
)

func buildTransactions(n int) Transactions {
	var txs Transactions
	for i := 0; i < n; i++ {
		txs = append(txs, &Transaction{Nonce: uint64(i)})
	}
	return txs
}

func TestComputeMerkelRoot0(t *testing.T) {
	txs := buildTransactions(0)
	root := ComputeMerkelRoot(txs)
	if !VerifyMerkelRoot(txs, root) {
		t.Fatal("verify merkel fail")
	}
	// fmt.Printf("root.len=%d, root=%+v", len(root), root)
}

func TestComputeMerkelRootN(t *testing.T) {
	for i := 1; i <= 1000; i++ {
		// fmt.Println("tx's len=", i)
		txs := buildTransactions(i)
		root := ComputeMerkelRoot(txs)
		if !VerifyMerkelRoot(txs, root) {
			t.Fatal("verify merkel fail")
		}
		// fmt.Printf("root.len=%d, root=%+v", len(root), root)
	}
}
