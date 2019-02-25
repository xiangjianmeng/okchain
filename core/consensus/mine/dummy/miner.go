package dummyminer

import (
	"encoding/hex"
	"math/big"

	"github.com/ok-chain/okchain/crypto/vrf"
	"github.com/ok-chain/okchain/crypto/vrf/p256"
	pb "github.com/ok-chain/okchain/protos"
)

type DummyMiner struct {
	privateKey       vrf.PrivateKey
	publicKey        string
	privateKeyString string
}

func NewDummyMiner(secretKey string) (*DummyMiner, error) {
	miner := &DummyMiner{}

	var pubKey, privKeyByte []byte
	var privKey vrf.PrivateKey
	var err error

	if len(secretKey) > 0 {
		privKey, pubKey, err = p256.RestoreKeyPair(secretKey)
		miner.privateKeyString = secretKey
	} else {
		privKey, privKeyByte, pubKey, err = p256.GenerateKeyPair()
		miner.privateKeyString = hex.EncodeToString(privKeyByte)
	}

	if err != nil {
		return nil, err
	}

	miner.privateKey = privKey
	miner.publicKey = hex.EncodeToString(pubKey)
	return miner, nil
}

func (miner *DummyMiner) GetKeyPairs() (string, string) {
	return miner.privateKeyString, miner.publicKey
}

func (miner *DummyMiner) Mine(seed []byte, blockNum uint64, difficulty *big.Int) (*pb.MiningResult, error) {

	mineResult := &pb.MiningResult{}

	mineResult.Nonce = 2018
	mineResult.MixHash = []byte("dummy PowResult")
	mineResult.PowResult = []byte("dummy PowResult")

	return mineResult, nil
}
