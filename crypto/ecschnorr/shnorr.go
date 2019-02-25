// Copyright The go-okchain Authors 2018,  All rights reserved.

package ecschnorr

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"errors"
	"fmt"
	"hash"
	"math/big"

	"github.com/ok-chain/okchain/crypto"
	"github.com/ok-chain/okchain/crypto/sha3"
)

var (
	ErrSigLen   = errors.New("Signature Length Invalid, Must be 64")
	ErrSigCheck = errors.New("Not a valid ec-schnorr signature")
)

// combinedMult implements fast multiplication S1*g + S2*p (g - generator, p - arbitrary point)
type combinedMult interface {
	CombinedMult(bigX, bigY *big.Int, baseScalar, scalar []byte) (x, y *big.Int)
}

type ECSchnorrSigner struct {
}
type ECSchnorrPriKey ecdsa.PrivateKey
type ECSchnorrPubKey ecdsa.PublicKey

func NewECSchnorrSigner() *ECSchnorrSigner {
	return &ECSchnorrSigner{}
}

func (priv *ECSchnorrPriKey) Bytes() []byte {
	return crypto.FromECDSA((*ecdsa.PrivateKey)(priv))
}

func (priv *ECSchnorrPriKey) PubKey() (crypto.PubKey, error) {
	if priv == nil {
		return nil, nil
	}
	return (*ECSchnorrPubKey)(&priv.PublicKey), nil
}

func (priv *ECSchnorrPriKey) SignMessage(message []byte) (signature []byte, err error) {
	hw := sha3.NewKeccak256()
	hash := hw.Sum(message)
	ecdsa.Sign(rand.Reader, (*ecdsa.PrivateKey)(priv), hash)
	return signECShnorr(priv, message, sha256.New())
}

func (pub *ECSchnorrPubKey) Bytes() []byte {
	return crypto.FromECDSAPub((*ecdsa.PublicKey)(pub))
}

func (pub *ECSchnorrPubKey) VerifyMessage(m, sig []byte) (valid bool, err error) {
	return verifyECShnorr(pub, sig, m, sha256.New())
}

func (b *ECSchnorrSigner) GenKeyParis() (sk crypto.PriKey, pk crypto.PubKey, err error) {
	//curve := crypto.S256()
	//P256 is faster
	curve := elliptic.P256()
	priv, _ := ecdsa.GenerateKey(curve, rand.Reader)
	pub := &ECSchnorrPubKey{priv.Curve, priv.X, priv.Y}

	return (*ECSchnorrPriKey)(priv), pub, nil
}

func (b *ECSchnorrSigner) Sign(k crypto.PriKey, m []byte) (signature []byte, err error) {
	return k.SignMessage(m)
}

func (b *ECSchnorrSigner) Verify(pubKey crypto.PubKey, msg, sig []byte) (valid bool, err error) {
	return pubKey.VerifyMessage(msg, sig)
}

func (b *ECSchnorrSigner) BytesToPriKey(buf []byte) (crypto.PriKey, error) {
	pri, err := crypto.ToECDSA(buf)
	return (*ECSchnorrPriKey)(pri), err
}

func (b *ECSchnorrSigner) BytesToPubKey(buf []byte) (crypto.PubKey, error) {
	pub := crypto.ToECDSAPub(buf)
	return (*ECSchnorrPubKey)(pub), nil
}

func (b *ECSchnorrSigner) AddPubKey(key1 crypto.PubKey, key2 crypto.PubKey) (crypto.PubKey, error) {
	k1, ok := key1.(*ECSchnorrPubKey)
	if !ok {
		return nil, errors.New("key is not type *ECDSAPubKey")
	}

	k2, ok := key2.(*ECSchnorrPubKey)
	if !ok {
		return nil, errors.New("key is not type *ECDSAPubKey")
	}
	res, err := crypto.AddECDSAPub((*ecdsa.PublicKey)(k1), (*ecdsa.PublicKey)(k2))
	return (*ECSchnorrPubKey)(res), err
}

//signECShnorr
func signECShnorr(prv *ECSchnorrPriKey, msg []byte, h hash.Hash) (sig []byte, err error) {
	//random k
	params := prv.Params()
	//bigOne := new(big.Int).SetUint64(1)
	k, err := rand.Int(rand.Reader, params.N)
	if err != nil {
		return nil, fmt.Errorf("Get rand k failed [%v]", err)
	}
	//compute r=H(R, pk, m)
	N := (params.BitSize + 7) / 8
	x, y := prv.Curve.ScalarBaseMult(k.Bytes())
	//buf := point2oct(prv.Curve.ScalarBaseMult(k.Bytes()), params.BitSize/8)
	buf := point2oct(x, y, N)
	if _, err = h.Write(buf); err != nil {
		return
	}
	buf = point2oct(prv.PublicKey.X, prv.PublicKey.Y, N)
	if _, err = h.Write(buf); err != nil {
		return
	}
	//hash the message
	if _, err = h.Write(msg); err != nil {
		return
	}
	e := h.Sum(nil)
	//to big.Int, nmod
	r := new(big.Int).SetBytes(e)
	r = r.Mod(r, params.N)
	h.Reset()
	//compute s = k-dr
	dr := mulMod(prv.D, r, params.N)
	s := subMod(k, dr, params.N)
	return asn1.Marshal(ecdsaSignature{r, s})
}

//verifyECShnorr
func verifyECShnorr(pub *ECSchnorrPubKey, sig, msg []byte, h hash.Hash) (valid bool, err error) {
	var signature ecdsaSignature
	_, err = asn1.Unmarshal(sig, &signature)
	if err != nil {
		return false, err
	}
	r, s := signature.R, signature.S
	params := pub.Params()
	bigOne := new(big.Int).SetUint64(1)
	if s.Cmp(new(big.Int).Sub(params.N, bigOne)) > 0 || r.Cmp(new(big.Int).Sub(params.N, bigOne)) > 0 {
		return false, errors.New("r, s is not valid")
	}
	//compute Q = sG+r*pk
	var grx, grv *big.Int
	if opt, ok := pub.Curve.(combinedMult); ok {
		grx, grv = opt.CombinedMult(pub.X, pub.Y, s.Bytes(), r.Bytes())
	} else {
		gsx, gsy := pub.ScalarBaseMult(s.Bytes())
		gex, gey := pub.ScalarMult(pub.X, pub.Y, r.Bytes())
		grx, grv = pub.Add(gsx, gsy, gex, gey)
	}
	N := (params.BitSize + 7) / 8
	buf := point2oct(grx, grv, N)
	if _, err = h.Write(buf); err != nil {
		return
	}
	buf = point2oct(pub.X, pub.Y, N)
	if _, err = h.Write(buf); err != nil {
		return
	}
	//hash the message
	if _, err = h.Write(msg); err != nil {
		return
	}
	e := h.Sum(nil)
	rr := new(big.Int).SetBytes(e)
	rr = rr.Mod(rr, params.N)
	h.Reset()
	//fmt.Printf("r: %v\nrr: %v\n", r, rr)
	if r.Cmp(rr) != 0 {
		return false, errors.New("Invalid Signature")
	}
	return true, nil
}
