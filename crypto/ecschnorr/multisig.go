// Copyright The go-okchain Authors 2018,  All rights reserved.

package ecschnorr

import (
	"crypto/ecdsa"
	"crypto/sha256"
	asn1 "encoding/asn1"
	"hash"
	"math/big"
)

type multiSignature struct {
	R, S *big.Int
}

//aggregatePubkeys
func AggregatePubkeys(pubs []ecdsa.PublicKey) *ecdsa.PublicKey {
	p := pubs[0]
	for i := 1; i < len(pubs); i++ {
		p.X, p.Y = p.Add(pubs[i].X, pubs[i].Y, p.X, p.Y)
	}
	return &p
}

//aggregateCommits
func AggregateCommits(commits []ecdsa.PublicKey) *ecdsa.PublicKey {
	return AggregatePubkeys(commits)
}

//Generate Challenge
func GenChanllenge(commitPoint *ecdsa.PublicKey, pk *ecdsa.PublicKey, digest []byte, h hash.Hash) (challenge *big.Int, err error) {
	//random k
	params := commitPoint.Params()
	//bigOne := new(big.Int).SetUint64(1)
	//k, err := rand.Int(rand.Reader, params.N)
	//if err != nil {
	//	return nil, fmt.Errorf("Get rand k failed [%v]", err)
	//}

	//compute r=H(R, pk, m)
	N := (params.BitSize + 7) / 8
	//x, y := commitPoint.Curve.ScalarBaseMult(k.Bytes())
	//buf := point2oct(prv.Curve.ScalarBaseMult(k.Bytes()), params.BitSize/8)
	//buf := point2oct(x, y, N)
	buf := point2oct(commitPoint.X, commitPoint.Y, N)
	if _, err = h.Write(buf); err != nil {
		return
	}

	buf = point2oct(pk.X, pk.Y, N)
	if _, err = h.Write(buf); err != nil {
		return
	}
	//hash the message
	if _, err = h.Write(digest); err != nil {
		return
	}
	e := h.Sum(nil)
	//to big.Int, nmod
	r := new(big.Int).SetBytes(e)
	r = r.Mod(r, params.N)
	h.Reset()

	//compute s = k-dr
	//dr := mulMod(prv.D, r, params.N)
	//s := subMod(k, dr, params.N)

	return r, nil
}

//aggregateResponses
func AggregateResponses(responses []big.Int, N *big.Int) *big.Int {
	r := new(big.Int).Set(&responses[0])
	for i := 1; i < len(responses); i++ {
		r.Add(&responses[i], r)
		//big.Int.Add(&responses[0], &responses[1])
	}
	r.Mod(r, N)
	return r
}

type ecdsaSignature struct {
	R, S *big.Int
}

//aggregateResponses
func VerifyResponse(challenge *big.Int, response *big.Int, digest []byte, pks []ecdsa.PublicKey) (valid bool, err error) {
	sig, _ := asn1.Marshal(ecdsaSignature{challenge, response})
	pk := AggregatePubkeys(pks)
	return verifyECShnorr((*ECSchnorrPubKey)(pk), sig, digest, sha256.New())
}
