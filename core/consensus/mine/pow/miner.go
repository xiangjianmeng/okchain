package powminer

import (
	"math/big"

	"github.com/ok-chain/okchain/core/consensus/ethash"
	"github.com/ok-chain/okchain/crypto/multibls"
	pb "github.com/ok-chain/okchain/protos"
)

type PowMiner struct {
	privateKey       multibls.PriKey
	publicKey        string
	privateKeyString string
}

func NewPowMiner(secretKey string) (*PowMiner, error) {
	miner := &PowMiner{}

	if len(secretKey) > 0 {
		err := miner.privateKey.SetHexString(secretKey)
		if err != nil {
			return miner, err
		}
		miner.privateKeyString = secretKey
		miner.publicKey = miner.privateKey.GetPublicKey().GetHexString()
	} else {
		priv, pk, err := multibls.BLSInit()
		if err != nil {
			return miner, err
		}
		miner.publicKey = pk.GetHexString()
		miner.privateKeyString = priv.GetHexString()
		miner.privateKey = priv
	}

	return miner, nil
}

func (miner *PowMiner) GetKeyPairs() (string, string) {
	return miner.privateKeyString, miner.publicKey
}

func (miner *PowMiner) Mine(seed []byte, blockNum uint64, difficulty *big.Int) (*pb.MiningResult, error) {

	nonce, hash, result := ethash.Mine2(blockNum, difficulty, seed)
	mineResult := &pb.MiningResult{}

	mineResult.Nonce = nonce
	mineResult.MixHash = hash
	mineResult.PowResult = result

	return mineResult, nil
}
