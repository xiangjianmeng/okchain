// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"log"
	"math/big"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/core/vm"
	"github.com/ok-chain/okchain/crypto"
	"github.com/ok-chain/okchain/protos"
)

const DSREWARD = 1000000

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *config.ChainConfig // Chain configuration options
	bc     ChainContext        // Canonical block chain
	peer   StateProcessorPeer
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *config.ChainConfig, bc ChainContext) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase).
//
// Process returns the receipts and logs accumulated during the process. If any of the
// transactions failed to execute it will return an error.
func (p *StateProcessor) Process(block *protos.TxBlock, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header
		allLogs  []*types.Log
		//gp       = new(GasPool).AddGas(block.Header.GasLimit)
		gp = new(GasPool).AddGas(10000000000000000)
	)
	// Iterate over and process the individual transactions
	for i, tx := range block.Body.Transactions {
		statedb.Prepare(tx.Hash(), block.Hash(), block.Header.BlockNumber, i)
		coreLogger.Debugf("tx data: %+v\n", tx)
		receipt, gas, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}
		// set sharding leader reward according to tx gas used
		statedb.AddBalance(common.BytesToAddress(block.Header.ShardingLeadCoinBase[p.peer.GetShardId(tx, block.Header.DSBlockNum)]), new(big.Int).SetUint64(gas*tx.GasPrice))
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	//p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.DposContext)

	// set reward for DS leader and DS back up
	for i, ds := range block.Header.DSCoinBase {
		if i == 0 {
			statedb.AddBalance(common.BytesToAddress(ds), new(big.Int).SetUint64(DSREWARD))
		} else {
			statedb.AddBalance(common.BytesToAddress(ds), new(big.Int).SetUint64(DSREWARD/2))
		}
	}
	return receipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) SetPeerServer(server StateProcessorPeer) {
	p.peer = server
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *config.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *protos.TxBlockHeader, tx *protos.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
	log.SetFlags(log.Lshortfile)
	log.Printf("ApplyTransaction\n")
	msg, err := tx.AsMsg()
	if err != nil {
		return nil, 0, err
	}
	// Create a new context to be used in the EVM environment
	context := NewEVMContext(msg, header, bc, author)

	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	vmenv := vm.NewEVM(context, statedb, config, cfg)

	// Apply the transaction to the current state (included in the env)
	_, gas, failed, err := ApplyMessage(vmenv, msg, gp)
	if err != nil {
		return nil, 0, err
	}
	// Update the state with pending changes
	var root []byte
	statedb.Finalise(true)
	*usedGas += gas
	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing wether the root touch-delete accounts.
	receipt := types.NewReceipt(root, failed, *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = gas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(vmenv.Context.Origin, tx.Nonce)
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())

	// TODO: add Bloom in future. FLT. 2018/11/05
	//receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	return receipt, gas, err
}

type StateProcessorPeer interface {
	GetShardId(tx *protos.Transaction, dsNum uint64) uint32
}
