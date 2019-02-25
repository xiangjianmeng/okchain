// Copyright The go-okchain Authors 2018,  All rights reserved.

package p256

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/math"
	"github.com/ok-chain/okchain/crypto"
	types "github.com/ok-chain/okchain/protos"
)

//@hxy, 2018/11/2
type P256Signer struct {
}

func (ec P256Signer) PubKeyToAddress(pub *ecdsa.PublicKey) common.Address {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return common.Address{}
	}
	pubBytes := elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	return common.BytesToAddress(crypto.Keccak256(pubBytes[1:])[12:])
}
func (ec P256Signer) PubKeyToBytes(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)
}
func (ec P256Signer) BytesToPubKey(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), pub)
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
}
func (ec P256Signer) PrvKeyToBytes(prv *ecdsa.PrivateKey) []byte {
	if prv == nil {
		return nil
	}
	return math.PaddedBigBytes(prv.D, prv.Params().BitSize/8)
}
func (ec P256Signer) BytesToPrvKey(d []byte) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = elliptic.P256()
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(priv.PublicKey.Curve.Params().N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

func (ec P256Signer) Sender(tx *types.Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, errors.New("Err Tx, must not be nil")
	}
	if tx.SenderPubKey == nil || len(tx.SenderPubKey) != 65 {
		return common.Address{}, fmt.Errorf("Invalid SenderPubKey length, Want 65, Get %d", len(tx.SenderPubKey))
	}
	// return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
	return common.BytesToAddress(crypto.Keccak256(tx.SenderPubKey[1:])[12:]), nil
}

func (ec P256Signer) Sign(t *types.Transaction, prv *ecdsa.PrivateKey) (*types.Transaction, error) {
	// sig, err := crypto.Sign(h[:], prv)
	t.SenderPubKey = elliptic.Marshal(prv.Curve, prv.X, prv.Y)
	h := t.Hash()
	sig, err := P256Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	t.Signature = sig
	// fmt.Printf("signtx, tx hash=%s, sig=%s\n", hex.EncodeToString(h[:]), hex.EncodeToString(sig))
	return t, nil
}

func (ec P256Signer) SignHash(msg []byte, prvBytes []byte) ([]byte, error) {
	// sig, err := crypto.Sign(h[:], prv)
	prv, err := ec.BytesToPrvKey(prvBytes)
	if err != nil {
		return nil, err
	}
	sig, err := P256Sign(msg, prv)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("signHash, msg=%s, sig=%s\n", hex.EncodeToString(msg), hex.EncodeToString(sig))
	return sig, nil
}
func (ec P256Signer) VerifyHash(digest, sig, pubKey []byte) (valid bool, err error) {
	// fmt.Printf("digest=%s, sig=%s, pubkey=%s", hex.EncodeToString(digest), hex.EncodeToString(sig), hex.EncodeToString(pubKey))
	if digest == nil || sig == nil || len(sig) != 64 || pubKey == nil {
		return false, fmt.Errorf("Err input len, digest[%d]=%s, sig[%d]=%s, pubkey[%d]=%s", len(digest), hex.EncodeToString(digest), len(sig), hex.EncodeToString(sig), len(pubKey), hex.EncodeToString(pubKey))
	}
	pub := ec.BytesToPubKey(pubKey)
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
	return ecdsa.Verify(pub, digest, r, s), nil
}
