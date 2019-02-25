package dummyminer

import (
	"math/big"

	pb "github.com/ok-chain/okchain/protos"
)

type DummyVerifier struct {
	publicKey string
}

func NewDummyVerifier(publicKey string) (*DummyVerifier, error) {
	miner := &DummyVerifier{}
	miner.publicKey = publicKey

	return miner, nil
}

func (vrf *DummyVerifier) Verify(seed []byte, blockNum uint64, difficulty *big.Int,
	result *pb.MiningResult) (bool, error) {
	return true, nil
}
