package sortition

import (
	"fmt"
	"math/big"

	"github.com/ok-chain/okchain/common/math"
	"github.com/ok-chain/okchain/crypto/vrf"
)

var (
	hashLen = int64(256)
)

func binomialKW(k, w int64, p float64) *big.Float {
	a := new(big.Int)
	a.Binomial(w, k)

	Fp := big.NewFloat(p)
	Fq := big.NewFloat(float64(1) - p)

	//b or c will be 0, if w is a big number
	//b := m.Pow(p, float64(k))
	//fmt.Println("b is ", b)
	//c := m.Pow(1-p, float64(w-k))
	//fmt.Println("c is ", c)
	// Fb := big.NewFloat(b)
	// Fc := big.NewFloat(c)
	//z.Mul(z, Fb)
	//fmt.Println("z*Fb is ", z)
	//z.Mul(z, Fc)
	//fmt.Println("z*Fc is ", z)

	z := new(big.Float)
	z.SetInt(a)
	var i int64
	for i = 0; i < k; i++ {
		z.Mul(z, Fp)
	}
	for i = 0; i < w-k; i++ {
		z.Mul(z, Fq)

	}

	return z
}

func cumulative(j int64, w int64, p float64) *big.Float {
	var i int64
	sum := big.NewFloat(0)
	for i = 0; i <= j; i++ {
		sum.Add(sum, binomialKW(i, w, p))
	}
	return sum
}

func getVotes(hash [32]byte, w *big.Int, p *big.Float) *big.Int {
	hashValue := new(big.Int).SetBytes(hash[:])
	tt := math.BigPow(2, hashLen)
	ttm1 := new(big.Int).Sub(tt, big.NewInt(1))
	maxValue := new(big.Int).Set(ttm1)

	var a, b, value big.Float
	a.SetInt(hashValue)
	b.SetInt(maxValue)
	value.Quo(&a, &b)
	//fmt.Println("value is ", &value)

	//Fp := big.NewFloat(p)
	Fp := p
	Fq := big.NewFloat(1)
	Fq.Sub(Fq, Fp)
	t := new(big.Float).Quo(Fp, Fq) // t is p/1-p

	curr := big.NewFloat(1)
	//i=0;i<w;i++
	for i := big.NewInt(0); i.Cmp(w) == -1; i.Add(i, big.NewInt(1)) {
		curr.Mul(curr, Fq)
	}
	//curr = cumulative(0, w, p)
	next := new(big.Float)
	item := new(big.Float).Copy(curr)

	//j=0;j<w && value >= curr; j++
	j := big.NewInt(0)
	t1 := new(big.Float).SetInt(w.Sub(w, j)) // t1 = w-j
	t2 := new(big.Float).SetInt(j.Add(j, big.NewInt(1)))
	for ; j.Cmp(w) == -1 && value.Cmp(curr) != -1; j.Add(j, big.NewInt(1)) {
		//	next := cumulative(j,w,p)
		item.Mul(item, t)
		item.Mul(item, t1)
		item.Quo(item, t2) // j+1 is needed
		t1.Sub(t1, big.NewFloat(1))
		t2.Add(t2, big.NewFloat(1))

		next.Add(curr, item)
		if value.Cmp(next) == -1 {
			return j
		}
		curr = next
	}
	if j.Cmp(w) == 0 {
		return j
	}
	return big.NewInt(0)
}

/* Sortition
* sk: 私钥
* seed: 种子
* tau: 期望选中的份额
* w: 个人的份额
* W: 总份额

* value: 产生的随机数
* proof: 随机数的证据
* j: 被选中的份额
 */
func Sortition(sk vrf.PrivateKey, seed []byte, tau, w, W *big.Int) (value [32]byte, proof []byte, j *big.Int) {
	value, proof = sk.Evaluate(seed)
	if W.Cmp(big.NewInt(0)) != 1 {
		fmt.Println("variable W is not positive")
		return value, proof, big.NewInt(0)
	}

	//p := float64(tau)/float64(W)
	p := new(big.Float).SetInt(tau)
	p.Quo(p, new(big.Float).SetInt(W))
	j = getVotes(value, w, p)
	return value, proof, j
}

/* VerifySort
* pk: 公钥
* value: 产生的随机数
* proof: 随机数的证据
* seed: 种子
* tau: 期望选中的份额
* w: 个人的份额
* W: 总份额

* j: 被选中的份额
 */
func VerifySort(pk vrf.PublicKey, value [32]byte, proof []byte, seed []byte, tau, w, W *big.Int) (j *big.Int) {
	flag, err := pk.Verify(seed, proof, value)
	if err != nil {
		fmt.Println("VerifySort Error")
	}
	if !flag {
		return big.NewInt(0)
	}
	//p := float64(tau)/float64(W)
	p := new(big.Float).SetInt(tau)
	p.Quo(p, new(big.Float).SetInt(W))
	j = getVotes(value, w, p)
	return j
}

/* SortitionCoinAge
* pk: 公钥
* value: 产生的随机数
* proof: 随机数的证据
* seed: 种子
* coinAge: 个人的币龄

* value: 产生的随机数
* proof: 随机数的证据
* j: 被选中的份额
 */
func SortitionCoinAge(sk vrf.PrivateKey, seed []byte, coinAge *big.Int) (value [32]byte, proof []byte, j *big.Int) {
	return Sortition(sk, seed, big.NewInt(1), coinAge, big.NewInt(2))
}

/* VerifySortCoinAge
* pk: 公钥
* value: 产生的随机数
* proof: 随机数的证据
* seed: 种子
* coinAge: 个人的币龄
* j: 被选中的份额

* b: 验证为真
 */
func VerifySortCoinAge(pk vrf.PublicKey, value [32]byte, proof []byte, seed []byte, coinAge *big.Int, j *big.Int) (b bool) {
	s := VerifySort(pk, value, proof, seed, big.NewInt(1), coinAge, big.NewInt(2))
	if j.Cmp(s) == 0 {
		return true
	} else {
		return false
	}
}
