// Copyright The go-okchain Authors 2018,  All rights reserved.

package protos

import (
	"fmt"
	"testing"
)

var tx = &Transaction{
	SenderPubKey: []byte("alice"),
	ToAddr:       []byte("bob"),
	Amount:       uint64(100),
}

func TestTxHash(t *testing.T) {
	b := &Transaction{SenderPubKey: []byte("hxy")}
	// b := tx
	hash := b.Hash()
	for i := 0; i < 1000; i++ {
		if hash != b.Hash() {
			t.Fatal("Same Tx's must be the same.")
		}
	}
	fmt.Println(b.Hash(), tx.Hash())
	if b.Hash() == tx.Hash() {
		t.Fatal("Different Tx's hash must differ")
	}
}

func TestSignAndVerify(t *testing.T) {
	// var (
	// 	curve  = elliptic.P256()
	// 	prv, _ = ecdsa.GenerateKey(curve, rand.Reader)
	// 	//public must be serialize to bytes using elliptic.Marshal
	// 	v  = new(big.Int).SetUint64(1)
	// 	tx = newTransaction(0, nil, &common.Address{}, 1, 100, v, []byte{})
	// )
	// s := P256Signer{prv}
	// tx, err := s.Sign(tx)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// valid, err := s.Verify(tx)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if !valid {
	// 	t.Fatal("Failed to verify transaction signature. !")
	// }

}
func TestSignTx(t *testing.T) {
	// var (
	// 	curve  = elliptic.P256()
	// 	prv, _ = ecdsa.GenerateKey(curve, rand.Reader)
	// 	//public must be serialize to bytes using elliptic.Marshal
	// 	v  = new(big.Int).SetUint64(1)
	// 	tx = newTransaction(0, nil, &common.Address{}, 1, 100, v, []byte{})
	// )
	// s := P256Signer{prv}
	// // tx, err := s.Sign(tx)
	// tx, err := SignTx(tx, s)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// valid, err := s.Verify(tx)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if !valid {
	// 	t.Fatal("Failed to verify transaction signature. !")
	// }

}

func TestSign2VerifyHash(t *testing.T) {
	// var (
	// 	curve  = elliptic.P256()
	// 	prv, _ = ecdsa.GenerateKey(curve, rand.Reader)
	// )
	// digest := []byte("hello hxy")
	// s := P256Signer{prv}
	// pubBytes := s.PubKeyToBytes(&prv.PublicKey)
	// sig, err := s.SignHash(digest)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// valid, err := s.VerifyHash(digest, sig, pubBytes)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if !valid {
	// 	t.Fatal("Failed to verify transaction signature. !")
	// }

}
