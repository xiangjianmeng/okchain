package config

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ok-chain/okchain/accounts/keystore"
	"github.com/ok-chain/okchain/common"
	types "github.com/ok-chain/okchain/protos"
)

func TestMakeAccountManager(t *testing.T) {
	//load configuration
	config, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	//make accounts manager
	amr, _, err := MakeAccountManager(config)
	if err != nil {
		t.Fatal(err)
	}
	if amr == nil {
		t.Fatal("MakeAccountManager failed")
	}
	//Get KeyStore
	ks := amr.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	//Create a new Account
	acc, _ := ks.NewAccount("okchain")
	fmt.Println("acc:", acc.Address)

	//keyJson, _ := ks.Export(acc, "okchain", "abc") //私钥导出
	ks.Unlock(acc, "okchain") //账户解锁
	//sig,  := ks.SignTx(acc, byte("tx content")) //交易内容签名
	//fmt.Println("sig:", sig)
	// crypto.Verify()
}

func TestKeyStore(t *testing.T) {
	ks, err := NewKeyStore("mykeystore")
	if err != nil {
		t.Error(err)
	}

	acc, err := ks.NewAccount("okchain")
	if err != nil {
		t.Error(err)
	}
	err = ks.Unlock(acc, "okchain")
	if err != nil {
		t.Error(err)
	}
	tx := types.NewTransaction(0, nil, common.Address{}, 1, 100, big.NewInt(100), []byte{})
	stx, err := ks.SignTx(acc, tx)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(stx)
}
