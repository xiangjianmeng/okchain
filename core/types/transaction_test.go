/**
 * Copyright 2015 OKCoin Inc.
 * transaction.go - testing
 * 2018/7/15. niwei create
 */

package types

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/crypto"
)

//BytesToAddress //515669b681730d83458f9a1d87dfe17bda62f9b1
func newTx() *Transaction {
	to := common.BytesToAddress([]byte("515669b681730d83458f9a1d87dfe17bda62f9b1"))
	amount := big.NewInt(1)
	return NewTransaction(Binary, 0, to, amount, []byte{})
}

func TestTransactionHash(t *testing.T) {
	tx := newTx()
	h := tx.Hash()

	t.Logf("hash : %x", h.Str())

	//var buf bytes.Buffer
	b := []byte{}
	buf := bytes.NewBuffer(b)

	err := tx.EncodeRLP(buf)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("tx encoderlp: %x", buf.Bytes())

	s := rlp.NewStream(buf, 0)

	tx1 := Transaction{}

	err = tx1.DecodeRLP(s)
	if err != nil {
		t.Fatal(err)
	}

	from := common.BytesToAddress([]byte("515669b681730d83458f9a1d87dfe17bda62f9b2"))

	t.Logf("from addr: %s", from)

}

func TestSingTx(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	tx, err := SignTx(NewTransaction(Binary, 0, addr, new(big.Int), nil), key)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("tx size: %f", tx.Size())

	addr1, err := Sender(tx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("addr : %b", addr)
	t.Logf("addr1: %b", addr1)
}
