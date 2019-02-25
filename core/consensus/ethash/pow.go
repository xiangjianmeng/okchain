package ethash

import (
	"bytes"
	"math/big"
	"runtime"
)

import (
	"github.com/ok-chain/okchain/crypto/sha3"
)

var maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

func Mine2(blockNum uint64, difficulty *big.Int, seed []byte) (nonce uint64, mixHash, result []byte) {
	//head := &types.Header{Number: big.NewInt(blockNum), Difficulty: big.NewInt(100)}
	ethash := NewTester()
	defer ethash.Close()

	sha := sha3.NewKeccak256()
	sha.Write(seed)
	hash := sha.Sum(nil)

	dataset := ethash.dataset(blockNum)
	//difficulty := big.NewInt(100)
	target := new(big.Int).Div(maxUint256, difficulty)
	nonce = uint64(1)
	for {
		mixHash, result = hashimotoFull(dataset.dataset, hash, nonce)
		if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
			return nonce, mixHash, result
		}
		nonce++
	}
	//return nonce, mixHash, result
}

func Check2(blockNum uint64, mixhash []byte, nonce uint64, difficulty *big.Int, seed []byte) bool {
	ethash := NewTester()
	defer ethash.Close()
	cache := ethash.cache(blockNum)

	sha := sha3.NewKeccak256()
	sha.Write(seed)
	hash := sha.Sum(nil)

	size := uint64(32 * 1024)
	digest, res := hashimotoLight(uint64(size), cache.cache, hash, nonce)

	runtime.KeepAlive(cache)

	if !bytes.Equal(mixhash, digest) {
		return false
	}
	target := new(big.Int).Div(maxUint256, difficulty)
	if new(big.Int).SetBytes(res).Cmp(target) > 0 {
		return false
	}

	return true
}

func Mine(blockNum uint64, difficulty *big.Int, rand1, rand2, ipAddr, pubKey string) (nonce uint64, mixHash, result []byte) {
	//head := &types.Header{Number: big.NewInt(blockNum), Difficulty: big.NewInt(100)}
	ethash := NewTester()
	defer ethash.Close()

	strs := rand1 + rand2 + ipAddr + pubKey
	// unchecksummed := hex.EncodeToString(a[:])
	sha := sha3.NewKeccak256()
	sha.Write([]byte(strs))
	hash := sha.Sum(nil)

	dataset := ethash.dataset(blockNum)
	//difficulty := big.NewInt(100)
	target := new(big.Int).Div(maxUint256, difficulty)
	nonce = uint64(1)
	for {
		mixHash, result = hashimotoFull(dataset.dataset, hash, nonce)
		if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
			return nonce, mixHash, result
		}
		nonce++
	}
	//return nonce, mixHash, result
}

func Check(blockNum uint64, mixhash []byte, nonce uint64, difficulty *big.Int, rand1, rand2, ipAddr, pubKey string) bool {
	ethash := NewTester()
	defer ethash.Close()
	cache := ethash.cache(blockNum)

	size := uint64(32 * 1024)

	strs := rand1 + rand2 + ipAddr + pubKey
	sha := sha3.NewKeccak256()
	sha.Write([]byte(strs))
	hash := sha.Sum(nil)

	digest, res := hashimotoLight(uint64(size), cache.cache, hash, nonce)

	runtime.KeepAlive(cache)

	if !bytes.Equal(mixhash, digest) {
		return false
	}
	target := new(big.Int).Div(maxUint256, difficulty)
	if new(big.Int).SetBytes(res).Cmp(target) > 0 {
		return false
	}

	return true
}
