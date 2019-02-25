// Copyright The go-okchain Authors 2018,  All rights reserved.

package ecschnorr

import (
	"math/big"
)

func zeroSlice(buf []byte) {
	if len(buf) == 0 {
		return
	}
	for i := 0; i < len(buf); i++ {
		buf[i] = byte(0)
	}
}

func point2oct(x, y *big.Int, N int) []byte {
	var buf = make([]byte, N*2)
	xBytes, yBytes := x.Bytes(), y.Bytes()
	copy(buf[N-len(xBytes):], xBytes)
	copy(buf[2*N-len(yBytes):], yBytes)
	return buf
}
func oct2point(buf []byte) (x, y *big.Int) {
	x = new(big.Int).SetBytes(buf[:len(buf)/2])
	y = new(big.Int).SetBytes(buf[len(buf)/2:])
	return
}

//returns (x*y)mod m
func mulMod(x, y, m *big.Int) (z *big.Int) {
	z = new(big.Int).Mul(x, y)
	return z.Mod(z, m)
}

//returns (x-y)mod m
func subMod(x, y, m *big.Int) (z *big.Int) {
	z = new(big.Int).Sub(x, y)
	return z.Mod(z, m)
}
