/**
 * Copyright 2015 OKCoin Inc.
 * utils.go -
 * 2018/7/13. niwei create
 */
package types

import (
	"bytes"
	//"math/big"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/core/trie"
	"github.com/ok-chain/okchain/crypto/sha3"
)

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

func RlpHash(data interface{}) common.Hash {
	return rlpHash(data)
}

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) common.Hash {
	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return trie.Hash()
}
