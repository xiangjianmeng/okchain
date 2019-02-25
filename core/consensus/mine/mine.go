package mine

import (
	"math/big"

	"github.com/ok-chain/okchain/core/consensus/mine/dummy"
	"github.com/ok-chain/okchain/core/consensus/mine/pow"
	"github.com/ok-chain/okchain/core/consensus/mine/vrf"
	pb "github.com/ok-chain/okchain/protos"
)

type Miner interface {
	Mine(seed []byte, blockNum uint64, difficulty *big.Int) (*pb.MiningResult, error)
	GetKeyPairs() (string, string)
}

type Verifier interface {
	Verify(seed []byte, blockNum uint64, difficulty *big.Int, result *pb.MiningResult) (bool, error)
}

func NewMiner(minertype string, secretKey string) (Miner, error) {

	if minertype == "pow" {

		return powminer.NewPowMiner(secretKey)

	} else if minertype == "vrf" {

		return vrfminer.NewVrfMiner(secretKey)

	}

	return dummyminer.NewDummyMiner(secretKey)
}

func NewVerifier(minertype string, key string) (Verifier, error) {

	if minertype == "pow" {

		return powminer.NewPowVerifier(key)

	} else if minertype == "vrf" {

		return vrfminer.NewVrfVerifier(key)

	}

	return dummyminer.NewDummyVerifier(key)
}
