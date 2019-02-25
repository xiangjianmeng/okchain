// Copyright The go-okchain Authors 2018,  All rights reserved.

package p256

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

func P256Sign(digest []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	r, s, err := ecdsa.Sign(rand.Reader, prv, digest)
	if err != nil {
		return nil, err
	}
	var sig [64]byte
	rb, sb := r.Bytes(), s.Bytes()
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	return sig[:], nil
}

func P256GenKey() (prv *ecdsa.PrivateKey, err error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}
