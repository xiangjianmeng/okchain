// Copyright The go-okchain Authors 2019,  All rights reserved.

package multibls

/*
#cgo CFLAGS:-DMCLBN_FP_UNIT_SIZE=6
#cgo LDFLAGS:-lbls384 -lgmpxx -lstdc++ -lgmp -lcrypto
#include <bls/bls.h>
*/
import "C"

import (
	"unsafe"

	. "github.com/bls"
	//"fmt"
	"github.com/pkg/errors"
	//. "github.com/okchain/blstest/BLS"
	//. "github.com/okchain/blstest/BLS"
)

type PriKey struct {
	SecretKey
}

type Sig struct {
	Sign
}

// PublicKey --
type PubKey struct {
	PublicKey
}

// getPointer --
func (pub *PubKey) getPointer() (p *C.blsPublicKey) {
	// #nosec

	return (*C.blsPublicKey)(unsafe.Pointer(&pub.PublicKey))
}

// getPointer --
func (sign *Sig) getPointer() (p *C.blsSignature) {
	// #nosec
	return (*C.blsSignature)(unsafe.Pointer(sign))
}

// getPointer --
func (sec *PriKey) getPointer() (p *C.blsSecretKey) {
	// #nosec
	return (*C.blsSecretKey)(unsafe.Pointer(sec))
}

func BLSInit() (sec PriKey, pub PubKey, err error) {

	err = Init(CurveFp254BNb)
	if err != nil {
		return sec, pub, err
	}
	//Init
	{
		var id ID
		err = id.SetLittleEndian([]byte{6, 5, 4, 3, 2, 1})
		if err != nil {
			return sec, pub, err
		}
		var id2 ID
		err = id2.SetHexString(id.GetHexString())
		if err != nil {
			return sec, pub, err
		}
		if !id.IsEqual(&id2) {
			return sec, pub, errors.New("hex string id not equal")
		}
		err = id2.SetDecString(id.GetDecString())
		if err != nil {
			return sec, pub, err
		}
		if !id.IsEqual(&id2) {
			return sec, pub, errors.New("dec string id not equal")
		}
	}

	{
		//sec = new(&SecretKey)
		err := sec.SetLittleEndian([]byte{1, 2, 3, 4, 5, 6})
		if err != nil {
			return sec, pub, err
		}
		//t.Log("sec=", sec.GetHexString())
	}
	sec.SecretKey.SetByCSPRNG()
	//fmt.Println("create secret key", sec.GetHexString())

	pub.PublicKey = *sec.SecretKey.GetPublicKey()
	//fmt.Println("create public key", pub)

	return sec, pub, nil
}

// Sign -- Constant Time version
func (sec *PriKey) BlSSign(buf []byte) (sign *Sig) {
	sign = new(Sig)
	//buf := []byte(m)3

	// #nosec bls_c.cpp
	C.blsSign(sign.getPointer(), sec.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf)))
	return sign
}

func (sign *Sig) BLSVerify(pub *PubKey, buf []byte) bool {
	// #nosec
	return C.blsVerify(sign.getPointer(), pub.getPointer(), unsafe.Pointer(&buf[0]), C.size_t(len(buf))) == 1
}

/*
func GenKeyPairs() (sec *PriKey, pub *PubKey, err error) {
	err = Init(CurveFp254BNb)
	sec = new(PriKey)

	if err != nil {
		return nil, nil, err
	}

	// Create key pair
	//fmt.Println("create secret key")

	sec.SecretKey.SetByCSPRNG()
	//fmt.Println("create secret key", sec.GetHexString())

	pub.PublicKey = sec.SecretKey.GetPublicKey()
	//fmt.Println("create public key", pub)

	return sec, pub, nil
}


func (pub *PubKey) VerifyMessage(msg, sig []byte) (bool, error) {
	BLSInit()
	sign := new(Sig)
	err := sign.Deserialize(sig)
	if err != nil {
		return false, err
	}
	// #nosec
	return C.blsVerify(sign.getPointer(), pub.getPointer(), unsafe.Pointer(&msg[0]), C.size_t(len(msg))) == 1, nil
}*/
