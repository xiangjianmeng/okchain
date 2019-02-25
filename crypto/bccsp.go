// Copyright The go-okchain Authors 2018,  All rights reserved.

package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/ok-chain/okchain/crypto/sha3"
	"github.com/pkg/errors"
)

// PriKey represents a cryptographic private key
type PriKey interface {
	// Bytes converts this key to its byte representation,
	// if this operation is allowed.
	Bytes() []byte

	// PublicKey returns the corresponding public key part of an asymmetric public/private key pair.
	// This method returns an error in symmetric key schemes.
	PubKey() (PubKey, error)

	SignMessage(message []byte) (signature []byte, err error)

	//BytesToPrivateKey([]byte) (PriKey, error)
}

// PubKey represents a cryptographic public key
type PubKey interface {
	// Bytes converts this key to its byte representation,
	// if this operation is allowed.
	Bytes() []byte

	VerifyMessage(m, sig []byte) (valid bool, err error)

	//BytesToPublicKey([]byte) (PubKey, error)
}

//type Signature interface {
//	Bytes() ([]byte, error)
//}

// BCCSP is the blockchain cryptographic service provider that offers
// the implementation of cryptographic standards and algorithms.
type Signer interface {

	// KeyGen generates a key using opts.
	GenKeyPairs() (sk PriKey, pk PubKey, err error)

	// Sign signs digest using key k.
	// The opts argument should be appropriate for the algorithm used.
	//
	// Note that when a signature of a hash of a larger message is needed,
	// the caller is responsible for hashing the larger message and passing
	// the hash (as digest).
	//SignHash(k PriKey, digest []byte) (signature []byte, err error)

	// Verify verifies signature against key k and digest
	// The opts argument should be appropriate for the algorithm used.
	//VerifyHash(k PubKey, signature, digest []byte) (valid bool, err error)

	Sign(k PriKey, m []byte) (signature []byte, err error)

	Verify(k PubKey, signature, m []byte) (valid bool, err error)

	BytesToPriKey([]byte) (PriKey, error)

	BytesToPubKey([]byte) (PubKey, error)
}

type ECDSASigner struct {
}

func (b *ECDSASigner) GenKeyParis() (sk PriKey, pk PubKey, err error) {
	curve := S256()
	priv, _ := ecdsa.GenerateKey(curve, rand.Reader)
	pub := &ECDSAPubKey{priv.Curve, priv.X, priv.Y}

	return (*ECDSAPriKey)(priv), pub, nil
}

func (b *ECDSASigner) Sign(k PriKey, m []byte) (signature []byte, err error) {
	return k.SignMessage(m)
}

func (b *ECDSASigner) Verify(pubKey PubKey, msg, sig []byte) (valid bool, err error) {
	return pubKey.VerifyMessage(msg, sig)
}

func (b *ECDSASigner) BytesToPriKey(buf []byte) (PriKey, error) {
	pri, err := ToECDSA(buf)
	return (*ECDSAPriKey)(pri), err
}

func (b *ECDSASigner) BytesToPubKey(buf []byte) (PubKey, error) {
	pub := ToECDSAPub(buf)
	return (*ECDSAPubKey)(pub), nil
}

func (b *ECDSASigner) AddPubKey(key1 PubKey, key2 PubKey) (PubKey, error) {
	k1, ok := key1.(*ECDSAPubKey)
	if !ok {
		return nil, errors.New("key is not type *ECDSAPubKey")
	}

	k2, ok := key2.(*ECDSAPubKey)
	if !ok {
		return nil, errors.New("key is not type *ECDSAPubKey")
	}
	res, err := AddECDSAPub((*ecdsa.PublicKey)(k1), (*ecdsa.PublicKey)(k2))
	return (*ECDSAPubKey)(res), err
}

type ECDSAPriKey ecdsa.PrivateKey
type ECDSAPubKey ecdsa.PublicKey

func (priv *ECDSAPriKey) Bytes() []byte {
	return FromECDSA((*ecdsa.PrivateKey)(priv))
}

func (priv *ECDSAPriKey) PubKey() (PubKey, error) {
	return (*ECDSAPubKey)(&priv.PublicKey), nil
}

func (priv *ECDSAPriKey) SignMessage(message []byte) (signature []byte, err error) {
	hw := sha3.NewKeccak256()
	hash := hw.Sum(message)
	//ecdsa.Sign(rand.Reader, (*ecdsa.PrivateKey)(priv), hash)
	sig, err := Sign(hash[:32], (*ecdsa.PrivateKey)(priv))
	return sig[:64], err
}

func (pub *ECDSAPubKey) Bytes() []byte {
	return FromECDSAPub((*ecdsa.PublicKey)(pub))
}

func (pub *ECDSAPubKey) VerifyMessage(m, sig []byte) (valid bool, err error) {
	pubBytes := pub.Bytes()
	hw := sha3.NewKeccak256()
	hash := hw.Sum(m)
	fmt.Println("pubBytes is ", pubBytes)
	return VerifySignature(pubBytes, hash[:32], sig), nil
}
