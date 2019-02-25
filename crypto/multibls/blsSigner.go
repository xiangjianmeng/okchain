// Copyright The go-okchain Authors 2018,  All rights reserved.

package multibls

/*
import "errors"
import (
	"github.com/ok-chain/okchain/crypto"
)

type BLSSigner struct {
}


func (b *BLSSigner) GenKeyParis() (sk crypto.PriKey, pk crypto.PubKey, err error) {
	return GenKeyPairs()
}

func (b *BLSSigner) Sign(k crypto.PriKey, m []byte) (signature []byte, err error) {
	return k.SignMessage(m)
}

func (b *BLSSigner) Verify(pubKey crypto.PubKey, msg, sig []byte) (valid bool, err error) {
	return pubKey.VerifyMessage(msg, sig)
}

func (b *BLSSigner) BytesToPriKey(buf []byte) (crypto.PriKey, error) {
	sec := new(PriKey)
	sec.SetLittleEndian(buf)
	return sec, nil
}

func (b *BLSSigner) BytesToPubKey(buf []byte) (crypto.PubKey, error) {
	pub := new(PubKey)
	pub.Deserialize(buf)
	return pub, nil
}

func (b *BLSSigner) AddPubKey(key1 crypto.PubKey, key2 crypto.PubKey) (crypto.PubKey, error) {
	k1, ok := key1.(*PubKey)
	if !ok {
		return nil, errors.New("key is not type *ECDSAPubKey")
	}

	k2, ok := key2.(*PubKey)
	if !ok {
		return nil, errors.New("key is not type *ECDSAPubKey")
	}

	k := *k1
	k.Add(k2.PublicKey)
	return &k, nil
}

//func (b *BLSSigner) AddSignature(sig1 crypto.Signature, sig2 crypto.Signature) (crypto.Signature, error) {
//	s1, ok := sig1.(*Sign)
//	if !ok {
//		return nil, errors.New("key is not type *ECDSAPubKey")
//	}
//
//	s2, ok := sig2.(*Sign)
//	if !ok {
//		return nil, errors.New("key is not type *ECDSAPubKey")
//	}
//
//	s := *s1
//	s.Add(s2)
//	return &s, nil
//}

func (b *BLSSigner) AddSignature(sig1 []byte, sig2 []byte) ([]byte, error) {
	s1 := new(Sig)
	s1.Deserialize(sig1)

	s2 := new(Sig)
	s2.Deserialize(sig2)

	s1.Add(&s2.Sign)
	return s1.Serialize(), nil
}
*/
