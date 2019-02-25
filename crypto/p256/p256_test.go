// Copyright The go-okchain Authors 2018,  All rights reserved.

package p256

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ok-chain/okchain/protos"

	"github.com/ok-chain/okchain/common"
)

var (
	prv      *ecdsa.PrivateKey
	prvBytes []byte
	pub      *ecdsa.PublicKey
	pubBytes []byte
	signer   *P256Signer
)

func init() {
	prv, _ = P256GenKey() //私钥生成
	pub = &prv.PublicKey

	signer = &P256Signer{}
	prvBytes = signer.PrvKeyToBytes(prv) //私钥序列化
	pubBytes = signer.PubKeyToBytes(pub)

	fmt.Printf("PrivaKey: %s\n PublicKey: %s\n", common.Bytes2Hex(prvBytes), common.Bytes2Hex(pubBytes))
}

func TestCreateAddress(t *testing.T) {
	//地址生成算法
	address := signer.PubKeyToAddress(pub) //公钥生成地址
	fmt.Printf("address: %s\n", common.Bytes2Hex(address[:]))
}

func TestSignTx(t *testing.T) {
	//**************交易签名*************************************
	tx := protos.NewTransaction(uint64(0), pubBytes, common.Address{}, uint64(1), uint64(10000), big.NewInt(1), []byte{})
	//交易签名算法1，签名后直接赋值到tx.Signature
	signedTx, err := SignTx(tx, signer, prvBytes) //交易签名算法
	if err != nil {
		t.Fatal(err)
	}
	tx = signedTx
	//交易签名算法2，SignTx2返回交易签名
	// sig, err := SignTx2(tx, signer, prvBytes)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// tx.Signature = sig
	fmt.Printf("tx's sig[%d]: %+v\n", len(tx.Signature), common.Bytes2Hex(tx.Signature))

	//验证签名
	h := tx.Hash() //交易序列化
	valid, err := signer.VerifyHash(h[:], tx.Signature, pubBytes)
	if err != nil || !valid {
		t.Fatal(valid, err)
	}
}
