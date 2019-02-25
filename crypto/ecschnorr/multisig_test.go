// Copyright The go-okchain Authors 2018,  All rights reserved.

package ecschnorr

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"
)

const N = 2

func TestMultiSig(t *testing.T) {
	////////////////////////////////////////////////////////
	fmt.Println("testMultiSig")

	//compute hash of msg
	h := sha256.New()
	h.Write([]byte("hello world"))
	digest := h.Sum(nil)

	curve := elliptic.P256()
	params := curve.Params()

	sks := make([]ecdsa.PrivateKey, 0, N)
	pks := make([]ecdsa.PublicKey, 0, N)
	keys := make([]ecdsa.PrivateKey, 0, N)
	pubs := make([]ecdsa.PublicKey, 0, N)
	responses := make([]big.Int, 0, N)

	//generate private key
	//Commitment Generation
	fmt.Println("Commitment Generation")

	for i := 0; i < N; i++ {
		//fmt.Println(i)

		pri, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			fmt.Printf("ecdsa key generate failed for [%v][%s]", curve, err)
			t.Fatal(err)
		}
		sks = append(sks, *pri)
		pks = append(pks, pri.PublicKey)

		pri2, err := ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			fmt.Printf("ecdsa key generate failed for [%v][%s]", curve, err)
			t.Fatal(err)
		}
		keys = append(keys, *pri2)
		pubs = append(pubs, pri2.PublicKey)
		pri2, _ = ecdsa.GenerateKey(curve, rand.Reader)
	}

	fmt.Println("sks ", sks)
	fmt.Println("pks ", pks)
	fmt.Println("keys ", keys)
	fmt.Println("pubs ", pubs)

	for i, v := range sks {
		fmt.Println(i, v)
	}
	fmt.Println("sk ", sks[0].D)
	fmt.Println("pk ", pks[0])

	//Chanllenge Generation
	fmt.Println("Chanllenge Generation")
	Q := AggregateCommits(pubs)
	fmt.Println("Q ", Q)
	fmt.Println("keys ", sks)

	pk := AggregatePubkeys(pks)
	fmt.Println("pk ", pk)

	//digest, err = csp.Hash(msg, &okcsp.Keccak256opts{})
	r, _ := GenChanllenge(Q, pk, digest, sha256.New())
	fmt.Println("r ", r)
	//fmt.Printf("r %v", r)
	fmt.Println("N ", params.N)

	//Response Generation
	fmt.Println("Response Generation")

	for i := 0; i < N; i++ {
		//fmt.Println(i)
		//compute s = k-dr
		dr := mulMod(sks[i].D, r, params.N)
		ts := subMod(keys[i].D, dr, params.N)
		fmt.Println("ts", ts)
		//fmt.Println("ts", *ts)
		//responses[i] = *ts
		responses = append(responses, *ts)
	}

	//Response Aggregation
	fmt.Println("Response Aggregation")
	fmt.Println("responses ", responses)
	//fmt.Println("responses ", &responses[1])
	s := AggregateResponses(responses, params.N)
	fmt.Println("s ", s)

	//Signature Verification
	valid, err := VerifyResponse(r, s, digest, pks)
	fmt.Println("Q ", Q)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Fatal("invalid signature")
	}
}
