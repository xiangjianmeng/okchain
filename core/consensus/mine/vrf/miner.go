package vrfminer

import (
	"encoding/hex"
	"math/big"

	"github.com/ok-chain/okchain/crypto/vrf"
	"github.com/ok-chain/okchain/crypto/vrf/p256"
	pb "github.com/ok-chain/okchain/protos"
)

type VrfMiner struct {
	privateKey       vrf.PrivateKey
	publicKey        string
	privateKeyString string
}

func NewVrfMiner(secretKey string) (*VrfMiner, error) {
	miner := &VrfMiner{}

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

func (miner *VrfMiner) GetKeyPairs() (string, string) {
	return miner.privateKeyString, miner.publicKey
}

func (miner *VrfMiner) Mine(seed []byte, blockNum uint64, difficulty *big.Int) (*pb.MiningResult, error) {

	value, proof := miner.privateKey.Evaluate(seed)
	result := &pb.MiningResult{}
	result.Proof = proof

	result.Vrfvalue = make([]byte, 32)
	for i := range value {
		result.Vrfvalue[i] = value[i]
	}

	return result, nil
}
