package powminer

import (
	"math/big"

	"github.com/ok-chain/okchain/core/consensus/ethash"
	pb "github.com/ok-chain/okchain/protos"
)

type PowVerifier struct {
	publicKey string
}

func NewPowVerifier(publicKey string) (*PowVerifier, error) {
	miner := &PowVerifier{}
	miner.publicKey = publicKey

	return miner, nil
}

func (vrf *PowVerifier) Verify(seed []byte, blockNum uint64, difficulty *big.Int,
	result *pb.MiningResult) (bool, error) {

	// verify pow calculate
	ret := ethash.Check2(blockNum, result.MixHash, result.Nonce, difficulty, seed)

	return ret, nil
}
