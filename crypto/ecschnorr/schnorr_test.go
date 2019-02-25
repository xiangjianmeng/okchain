// Copyright The go-okchain Authors 2018,  All rights reserved.

package ecschnorr

import (
	"testing"

	"runtime"

	"github.com/ok-chain/okchain/crypto"
)

var (
	// kh     = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	// pri, _ = crypto.HexToECDSA(kh)

	msg = crypto.Keccak256([]byte("xueyang.han@okcoin.com"))
)

func Test_Schnorr(t *testing.T) {
	schnorr := NewECSchnorrSigner()
	pri, pub, err := schnorr.GenKeyParis()
	if err != nil {
		t.Error(err)
	}

	t.Logf("private key:%v", pri.Bytes())
	t.Logf("public key:%v", pub.Bytes())

	msg := []byte("@xueyang.han")
	sig, err := schnorr.Sign(pri, msg)
	if err != nil {
		t.Error(err)
	}

	valid, err := schnorr.Verify(pub, msg, sig)
	if err != nil {
		t.Error(err)
	}
	if !valid {
		t.Error(ErrSigCheck)
	} else {
		t.Log("verify signature success")
	}
}

var (
	numCPUs = 4
)

func BenchmarkSign(b *testing.B) {
	b.ReportAllocs()
	runtime.GOMAXPROCS(1)
	schnorr := NewECSchnorrSigner()
	pri, _, _ := schnorr.GenKeyParis()
	for i := 0; i < b.N; i++ {
		schnorr.Sign(pri, msg)
	}
}

func BenchmarkVerify(b *testing.B) {
	b.ReportAllocs()
	runtime.GOMAXPROCS(1)
	schnorr := NewECSchnorrSigner()
	pri, pub, _ := schnorr.GenKeyParis()
	sig, _ := schnorr.Sign(pri, msg)
	for i := 0; i < b.N; i++ {
		schnorr.Verify(pub, msg, sig)
	}
}

func BenchmarkS(b *testing.B) {
	b.StopTimer()
	b.ReportAllocs()
	runtime.GOMAXPROCS(numCPUs)

	schnorr := NewECSchnorrSigner()

	pri, _, _ := schnorr.GenKeyParis()

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			schnorr.Sign(pri, msg)
		}
	})
}

func BenchmarkV(b *testing.B) {
	b.StopTimer()
	b.ReportAllocs()
	runtime.GOMAXPROCS(numCPUs)

	schnorr := NewECSchnorrSigner()

	pri, pub, _ := schnorr.GenKeyParis()
	sig, _ := schnorr.Sign(pri, msg)

	b.StartTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			schnorr.Verify(pub, msg, sig)
		}
	})
}
