package powminer

import (
	"fmt"
	"math/big"
	"testing"
)

func TestMining(t *testing.T) {

	//m, _ := NewPowMiner("e30fca8105313163079a3232e29a84de546bae4a211b278c0fca47306fdc14ab")
	m, _ := NewPowMiner("")
	seed := []byte("blockhash")
	blockNum := uint64(99)
	difficulty := big.NewInt(1666866)

	sk, _ := m.GetKeyPairs()

	fmt.Printf("sk: %s\n", sk)

	res, err := m.Mine(seed, blockNum, difficulty)

	if err != nil {
		t.Fatalf("Mining error: %s", err)
	}

	v, _ := NewPowVerifier(m.publicKey)
	verifyResult, err := v.Verify(seed, blockNum, difficulty, res)

	if err != nil {
		t.Fatalf("verify error: %s", err)
	}

	if !verifyResult {
		t.Fatalf("verify failed")
	}
}
