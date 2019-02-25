package sortition

import (
	"fmt"
	"math/big"
	"testing"

	"encoding/hex"

	"github.com/ok-chain/okchain/crypto/vrf/p256"
)

func TestBinomialKW(t *testing.T) {
	var r *big.Float
	r = binomialKW(0, 2, 0.5)
	fmt.Println(r)

	r = binomialKW(1, 2, 0.5)
	fmt.Println(r)

	r = binomialKW(2, 2, 0.5)
	fmt.Println(r)
}

func TestBinomialKW1(t *testing.T) {
	var r *big.Float
	r = binomialKW(100, 200, 0.5)
	fmt.Println(r)

	r = binomialKW(19900, 20000, 0.5)
	fmt.Println(r)

	r = binomialKW(199900, 200000, 0.5)
	fmt.Println(r)
}

func TestSortition(t *testing.T) {
	tau := big.NewInt(5000)
	w := big.NewInt(1000)
	W := big.NewInt(10000)

	sk, _ := p256.GenerateKey()
	seed := make([]byte, 20)
	value, proof, j := Sortition(sk, seed, tau, w, W)
	fmt.Println("value: ", value)
	fmt.Println("proof: ", proof)
	fmt.Println("j: ", j)
}

func TestSortitionMinMax(t *testing.T) {
	tau := big.NewInt(5000)
	w := big.NewInt(1000)
	W := big.NewInt(10000)
	var i int
	min := new(big.Int).Set(w)
	max := big.NewInt(0)

	for i = 0; i < 10000; i++ {
		sk, _ := p256.GenerateKey()
		seed := make([]byte, 20)
		_, _, j := Sortition(sk, seed, tau, w, W)
		//fmt.Println("j: ", j)
		if 1 == j.Cmp(max) {
			max.Set(j)
		}
		if -1 == j.Cmp(min) {
			min.Set(j)
		}
	}
	fmt.Println("min: ", min)
	fmt.Println("max: ", max)

}

func TestVerifySort(t *testing.T) {
	tau := big.NewInt(5000)
	w := big.NewInt(1000)
	W := big.NewInt(10000)

	sk, pk := p256.GenerateKey()
	seed := make([]byte, 20)
	value, proof, j := Sortition(sk, seed, tau, w, W)
	refJ := VerifySort(pk, value, proof, seed, tau, w, W)
	fmt.Println("j is ", j)
	if j.Cmp(refJ) != 0 {
		fmt.Println("VerfiySort failed!")
	}
}

func TestVerifySort20000(t *testing.T) {
	tau := big.NewInt(10000 - 1)
	w := big.NewInt(10000)
	W := big.NewInt(10000)

	sk, pk := p256.GenerateKey()
	seed := make([]byte, 20)
	value, proof, j := Sortition(sk, seed, tau, w, W)
	refJ := VerifySort(pk, value, proof, seed, tau, w, W)
	fmt.Println("j is ", j)
	if j.Cmp(refJ) != 0 {
		t.Errorf("VerfiySort failed!")
	}
}

//计算一下概率累积曲线，测试运行时间和精度
func TestExample(t *testing.T) {
	var r *big.Float

	sum := big.NewFloat(0)
	var i int64
	for i = 0; i < 20000; i++ {
		r = binomialKW(i, 20000, 0.5)
		sum.Add(sum, r)
		if i%1000 == 0 {
			fmt.Println("i is ", i)
		}
	}
	fmt.Println(sum)
	//sum is 0.9999999999999997, 偏差还行，就是运行的慢了点
}

func TestVRF1(t *testing.T) {

	privateKey, _, publicKeyStr := p256.GenerateKey2()

	publicKeytmp := p256.String2Pubkey(publicKeyStr)
	i := 0
	for {
		fmt.Printf("=============\n")

		blockHash := []byte("blockHash")
		value1, proof := privateKey.Evaluate(blockHash)

		for i, s := range value1 {
			fmt.Printf("%v: %v\n", i, s)

		}

		fmt.Printf("value1: %x\n", value1)

		value1str := fmt.Sprintf("%x", value1)

		v := make([]byte, 32)
		v, _ = hex.DecodeString(value1str)

		fmt.Printf("value1: %x, value1tmp: %x\n", value1, v)

		fmt.Printf("proof: %x\n", proof)

		value2, err := publicKeytmp.ProofToHash(blockHash, proof)

		fmt.Printf("value2: %x\n", value1)
		if err != nil {
			t.Errorf("ProofToHash failed: %s", err)
		}

		if value1 != value2 {
			t.Errorf("ProofToHash failed!")
		}
		i++
		if i >= 3 {
			return
		}
	}
}

func TestSortitionCoinAge(t *testing.T) {
	seed := "sdfgsfdhsrthetrhtyhbethetgbfgbdf"
	for j := 0; j < 20; j++ {
		for i := 0; i < 35; i++ {
			seed = fmt.Sprintf("(%s) (%d)", seed, j+i)
			calc(int64(i), seed)
		}
		fmt.Printf("\n")
	}
}

func calc(age int64, seedStr string) {
	coinAge := big.NewInt(age)

	seed := []byte(seedStr)
	//fmt.Printf("====================\n")
	sk, _ := p256.GenerateKey()

	for i := 0; i < 1; i++ {

		value, proof, j := SortitionCoinAge(sk, seed, coinAge)
		_ = value
		_ = proof
		//_, _, j := SortitionCoinAge(sk, seed, coinAge)
		//fmt.Printf("w: %d, j: %d\n", age, j)
		fmt.Printf("%d,", j)
		//fmt.Printf("value: %x\n", value)
		//fmt.Printf("proof: %x\n", proof)
	}

	//fmt.Printf("\n")

}

func TestVerifySortCoinAge(t *testing.T) {
	coinAge := big.NewInt(10)

	sk, pk := p256.GenerateKey()
	seed := make([]byte, 20)
	value, proof, j := SortitionCoinAge(sk, seed, coinAge)
	b := VerifySortCoinAge(pk, value, proof, seed, coinAge, j)
	if b {
		fmt.Println("VerifySortCoinAge success!")
	} else {
		fmt.Println("VerfiySortCoinAge failed!")
	}
}
