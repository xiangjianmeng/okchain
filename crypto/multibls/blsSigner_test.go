// Copyright The go-okchain Authors 2018,  All rights reserved.

package multibls

/*
import (
	"fmt"
	"testing"
)

// bls signature test
func TestExample(t *testing.T) {

	signer := &BLSSigner{}
	pri, pub, _ := signer.GenKeyParis()

	m := []byte("Hello BLS")
	t.Log("Sign Message :", m)
	sig, _ := signer.Sign(pri, m)
	valid, err := signer.Verify(pub, m, sig)

	if err != nil {
		t.Error(err)
	}
	if valid {
		t.Log("verify sucsess")
	} else {
		t.Error("verify fail")
	}
}

// bls multi-sign
func TestSecSign(t *testing.T) {
	t.Log("testMultiSign")

	signer := &BLSSigner{}

	err := Init(CurveFp254BNb)
	if err != nil {
		fmt.Println("init fail")
		return
	}

	sec1, pub1, _ := signer.GenKeyParis()
	sec2, pub2, _ := signer.GenKeyParis()
	sec3, pub3, _ := signer.GenKeyParis()

	m := []byte("test test")
	sign1, _ := sec1.SignMessage(m)
	t.Log("sign1    :", sign1)

	sign2, _ := sec2.SignMessage(m)
	t.Log("sign2    :", sign2)

	sign3, _ := sec3.SignMessage(m)
	t.Log("sign3    :", sign3)

	sign, _ := signer.AddSignature(sign1, sign2)
	t.Log("sign1 add sign2:", sign)

	pub, _ := signer.AddPubKey(pub1, pub2)

	valid, _ := signer.Verify(pub, m, sign)

	if valid {
		fmt.Println("sign1 add sign2 success")
	}

	sign, _ = signer.AddSignature(sign, sign3)
	t.Log("sign add sign3:", sign)

	pub, _ = signer.AddPubKey(pub, pub3)

	valid, _ = signer.Verify(pub, m, sign)

	if valid {
		fmt.Println("sign add sign3 success")
	}
}

// bls multi-sign
func TestSignerSign(t *testing.T) {
	t.Log("testMultiSign")

	signer := &BLSSigner{}

	err := Init(CurveFp254BNb)
	if err != nil {
		fmt.Println("init fail")
		return
	}

	sec1, pub1, _ := signer.GenKeyParis()
	sec2, pub2, _ := signer.GenKeyParis()
	sec3, pub3, _ := signer.GenKeyParis()

	m := []byte("test test")
	sign1, _ := signer.Sign(sec1, m)
	t.Log("sign1    :", sign1)

	sign2, _ := signer.Sign(sec2, m)
	t.Log("sign2    :", sign2)

	sign3, _ := signer.Sign(sec3, m)
	t.Log("sign3    :", sign3)

	sign, _ := signer.AddSignature(sign1, sign2)
	t.Log("sign1 add sign2:", sign)

	pub, _ := signer.AddPubKey(pub1, pub2)

	valid, _ := signer.Verify(pub, m, sign)

	if valid {
		fmt.Println("sign1 add sign2 success")
	}

	sign, _ = signer.AddSignature(sign, sign3)
	t.Log("sign add sign3:", sign)

	pub, _ = signer.AddPubKey(pub, pub3)

	valid, _ = signer.Verify(pub, m, sign)

	if valid {
		fmt.Println("sign add sign3 success")
	}
}

// serialize
func TestBytes(t *testing.T) {
	t.Log("testMultiSign")

	signer := &BLSSigner{}

	err := Init(CurveFp254BNb)
	if err != nil {
		fmt.Println("init fail")
		return
	}

	sec1, pub1, _ := signer.GenKeyParis()
	sec2, pub2, _ := signer.GenKeyParis()
	sec3, pub3, _ := signer.GenKeyParis()

	t.Log("sec1    :", sec1.Bytes())
	t.Log("sec2    :", sec2.Bytes())
	t.Log("sec3    :", sec3.Bytes())
	t.Log("pub1    :", pub1.Bytes())
	t.Log("pub2    :", pub2.Bytes())
	t.Log("pub3    :", pub3.Bytes())
}
*/
