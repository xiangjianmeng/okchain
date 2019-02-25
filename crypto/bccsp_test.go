// Copyright The go-okchain Authors 2018,  All rights reserved.

package crypto

import (
	"testing"
)

// bls signature test
func TestExample(t *testing.T) {

	signer := &ECDSASigner{}
	pri, pub, _ := signer.GenKeyParis()
	t.Log("pri is :", pri)
	t.Log("pri bytes is :", pri.Bytes())
	t.Log("pub is :", pub)
	t.Log("pub bytes is :", pub.Bytes())

	m := []byte("Hello ECDSA")
	t.Log("Sign Message :", m)
	sig, err := signer.Sign(pri, m)
	if err != nil {
		t.Error(err)
	}

	t.Log("sig is :", sig)
	t.Log("sig len is : ", len(sig))
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
