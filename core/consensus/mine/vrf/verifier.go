package vrfminer

import (
	"github.com/ok-chain/okchain/crypto/vrf/p256"
	pb "github.com/ok-chain/okchain/protos"

	"math/big"
)

type VrfVerifier struct {
	publicKey string
}

func NewVrfVerifier(publicKey string) (*VrfVerifier, error) {
	miner := &VrfVerifier{}
	miner.publicKey = publicKey

	return miner, nil
}

func (vrf *VrfVerifier) Verify(seed []byte, blockNum uint64, difficulty *big.Int,
	result *pb.MiningResult) (bool, error) {

	var value [32]byte
	for i := range value {
		value[i] = result.Vrfvalue[i]
	}
	return p256.Verify(vrf.publicKey, seed, result.Proof, value)
}
