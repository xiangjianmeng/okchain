package core

import (
	"math/big"
	"testing"

	"github.com/ok-chain/okchain/core/database"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/vm"
	"github.com/ok-chain/okchain/protos"
)

func DummyFinalBlock() *protos.TxBlock {
	var txs []*protos.Transaction
	for i := 0; i < 10; i++ {
		tx := &protos.Transaction{
			SenderPubKey: []byte("alice"),
			ToAddr:       []byte("bob"),
			Amount:       uint64(i),
			Nonce:        uint64(i),
		}
		txs = append(txs, tx)
	}
	return &protos.TxBlock{}
}

func TestProcess(t *testing.T) {
	var (
		block      = DummyFinalBlock()
		memdb, _   = database.NewMemDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(memdb))
	)
	statedb.AddBalance(common.BytesToAddress([]byte("alice")), new(big.Int).SetUint64(100))

	if _, _, _, err := NewStateProcessor(nil, nil).Process(block, statedb, vm.Config{}); err != nil {
		t.Fatal(err)
	}

	b1 := statedb.GetBalance(common.BytesToAddress([]byte("alice")))
	b2 := statedb.GetBalance(common.BytesToAddress([]byte("bob")))

	t.Logf("Balance: Alice=%s Bob=%s\n", b1.String(), b2.String())
}
