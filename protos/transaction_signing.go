// Copyright The go-okchain Authors 2018,  All rights reserved.

package protos

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/math"
	"github.com/ok-chain/okchain/crypto"
	"github.com/ok-chain/okchain/crypto/multibls"

	//"github.com/ok-chain/okchain/crypto/bls"
	logging "github.com/ok-chain/okchain/log"
)

var logger = logging.MustGetLogger("protos")

func MakeSigner(config string) CSigner {
	var signer CSigner
	switch config {
	//check config to define specified signer
	case "BLS":
		signer = BLSSigner{}
	case "S256":
		signer = S256Signer{}
	default:
		signer = P256Signer{}
	}
	return signer
}

// func Sender(signer CSigner, tx *Transaction) (common.Address, error) {
// 	return signer.Sender(tx)
// }

// SignTx signs the transaction using the given signer and private key
func SignTx(t *Transaction, s CSigner, prvBytes []byte) (*Transaction, error) {
	if signer, ok := s.(P256Signer); ok {
		prv, err := signer.BytesToPrvKey(prvBytes)
		if err != nil {
			return nil, err
		}
		t.SenderPubKey = signer.PubKeyToBytes(&prv.PublicKey)
	}
	h := t.Hash()
	sig, err := s.SignHash(h[:], prvBytes)
	if err != nil {
		return nil, err
	}
	t.Signature = sig
	return t, nil
	// return s.Sign(t)
}

func P256Sign(digest []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, prv, digest)
	if err != nil {
		return nil, err
	}
	var sig [64]byte
	rb, sb := r.Bytes(), s.Bytes()
	// fmt.Println(len(rb), len(sb))
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	return sig[:], nil
}

// Signer encapsulates transaction signature handling. Note that this interface is not a
// stable API and may change at any time to accommodate new protocol rules.
type CSigner interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *Transaction) (common.Address, error)
	// // Hash returns the hash to be signed.
	// Hash(tx *Transaction) common.Hash
	// //Verify returns true if the signature against tx is valid
	// Verify(tx *Transaction) (valid bool, err error)
	// //Sign returns signature of transaction
	// Sign(tx *Transaction) (t *Transaction, err error)
	//SignHash returns the signature of hash
	SignHash(msg []byte, prvBytes []byte) (sig []byte, err error)
	//VerifyHash returns true if the sig is valid
	VerifyHash(msg, sig, pubKey []byte) (valid bool, err error)
}

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

func (ec P256Signer) BytePubKeyToAddress(pub []byte) common.Address {
	return common.BytesToAddress(crypto.Keccak256(pub[1:])[12:])
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
func (ec P256Signer) LoadECDSA(file string) (*ecdsa.PrivateKey, error) {
	buf := make([]byte, 64)
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	if _, err := io.ReadFull(fd, buf); err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}
	return ec.BytesToPrvKey(key)
}

func (ec P256Signer) BytesToPrvKey(d []byte) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = elliptic.P256()
	//if 8*len(d) != priv.Params().BitSize {
	//	return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	//}
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

func (ec P256Signer) Sender(tx *Transaction) (common.Address, error) {
	if tx == nil {
		return common.Address{}, errors.New("Err Tx, must not be nil")
	}
	if tx.SenderPubKey == nil || len(tx.SenderPubKey) != 65 {
		return common.Address{}, fmt.Errorf("Invalid SenderPubKey length, Want 65, Get %d", len(tx.SenderPubKey))
	}
	// return common.BytesToAddress(Keccak256(pubBytes[1:])[12:])
	return common.BytesToAddress(crypto.Keccak256(tx.SenderPubKey[1:])[12:]), nil
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
	logger.Debugf("signHash, msg=%s, sig=%s\n", hex.EncodeToString(msg), hex.EncodeToString(sig))
	return sig, nil
}
func (ec P256Signer) VerifyHash(digest, sig, pubKey []byte) (valid bool, err error) {
	logger.Debugf("digest=%s, sig=%s, pubkey=%s", hex.EncodeToString(digest), hex.EncodeToString(sig), hex.EncodeToString(pubKey))
	if digest == nil || sig == nil || len(sig) != 64 || pubKey == nil {
		return false, fmt.Errorf("Err input len, digest[%d]=%s, sig[%d]=%s, pubkey[%d]=%s", len(digest), hex.EncodeToString(digest), len(sig), hex.EncodeToString(sig), len(pubKey), hex.EncodeToString(pubKey))
	}
	pub := ec.BytesToPubKey(pubKey)
	r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
	return ecdsa.Verify(pub, digest, r, s), nil
}

type BLSSigner struct {
	Prv *multibls.PriKey
}

func (b *BLSSigner) SetKey(prv *multibls.PriKey) {
	b.Prv = prv
}

func (b BLSSigner) Sender(tx *Transaction) (common.Address, error) {
	return common.Address{}, nil
}

func (b BLSSigner) SignHash(msg []byte, prvBytes []byte) (sig []byte, err error) {
	s := b.Prv.BlSSign(msg)
	sig = s.Serialize()
	return sig, nil
}

func (b BLSSigner) VerifyHash(msg, sig, pubKey []byte) (valid bool, err error) {
	//bls.BLSInit()
	s := &multibls.Sig{}
	err = s.Deserialize(sig)
	if err != nil {
		return false, err
	}

	pub := &multibls.PubKey{}
	err = pub.Deserialize(pubKey)
	if err != nil {
		return false, err
	}
	valid = s.BLSVerify(pub, msg)
	if valid {
		return true, nil
	} else {
		return false, errors.New("bls verify failed")
	}
}

//@hxy, 2018/11/2
type S256Signer struct {
}

func (ec S256Signer) PubKeyToAddress(pub *ecdsa.PublicKey) common.Address {
	return crypto.PubkeyToAddress(*pub)
}
func (ec S256Signer) PubKeyToBytes(pub *ecdsa.PublicKey) []byte {
	return crypto.FromECDSAPub(pub)
}
func (ec S256Signer) BytesToPubKey(pub []byte) *ecdsa.PublicKey {
	return crypto.ToECDSAPub(pub)
}
func (ec S256Signer) PrvKeyToBytes(prv *ecdsa.PrivateKey) []byte {
	return crypto.FromECDSA(prv)
}
func (ec S256Signer) BytesToPrvKey(d []byte) (*ecdsa.PrivateKey, error) {
	return crypto.ToECDSA(d)
}

func (ec S256Signer) Sender(tx *Transaction) (common.Address, error) {
	return common.BytesToAddress(tx.SenderPubKey), nil
}

func (ec S256Signer) SignHash(msg []byte, prvBytes []byte) ([]byte, error) {
	prv, err := ec.BytesToPrvKey(prvBytes)
	if err != nil {
		return nil, errors.New("Not a valid S256 prvBytes")
	}
	sig, err := crypto.Sign(msg, prv)
	if err != nil {
		return nil, err
	}
	return sig, nil
}
func (ec S256Signer) VerifyHash(digest, sig, pubKey []byte) (valid bool, err error) {
	if len(sig) != 65 {
		return false, fmt.Errorf("wrong size for signature: got %d, want 65", len(sig))
	}
	//if Ecrecover the pub, tx is valid against signature
	_, err = crypto.Ecrecover(digest, sig)
	if err != nil {
		return false, errors.New("Invalid signature")
	}
	return true, nil
}

func (b *BLSSigner) PubKeyBytes2String(pubKey []byte) (string, error) {
	pub := multibls.PubKey{}
	err := pub.Deserialize(pubKey)
	if err != nil {
		return "", err
	}
	return pub.GetHexString(), nil
}
