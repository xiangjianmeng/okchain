package ethash

import (
	"math/big"
	"testing"
)

func TestTestMode(t *testing.T) {
	blockNum := uint64(10)
	difficulty := big.NewInt(100)

	rand1 := "abc"
	rand2 := "afk"

	pubKey := ""
	address := "127.0.0.1"

	nonce, mixHash, _ := Mine(blockNum, difficulty, rand1, rand2, address, pubKey)
	isTrue := Check(blockNum, mixHash, nonce, difficulty, rand1, rand2, address, pubKey)
	if isTrue {
		t.Log("pow check passed")
	} else {
		t.Fatalf("pow check failed")
	}
}
