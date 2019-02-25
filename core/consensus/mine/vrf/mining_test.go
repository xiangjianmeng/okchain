package vrfminer

import (
	"testing"
)

func TestMining(t *testing.T) {

	//m, _ := NewVrfMiner("")
	//m, _ := NewVrfMiner("178c14253200fa22c85c87807ec913c94cf73e5c9c59088c512159d889e0988c")
	m, _ := NewVrfMiner("c4227c66032e55d30751bd03b82d43ba692ddda9b39f59d62ec25d0c7f6ccb25")

	seed := []byte("blockhash")
	blockNum := uint64(99)
	res, err := m.Mine(seed, blockNum, nil)

	if err != nil {
		t.Fatalf("Mining error: %s", err)
	}

	v, _ := NewVrfVerifier(m.publicKey)

	verifyResult, err := v.Verify(seed, blockNum, nil, res)

	if err != nil {
		t.Fatalf("verify error: %s", err)
	}

	if !verifyResult {
		t.Fatalf("verify failed")
	}
}
