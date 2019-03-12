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
	"math/big"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/core/vm"
	"github.com/ok-chain/okchain/crypto"
	"github.com/ok-chain/okchain/protos"
	"sort"
)

// yield half rewards per 1576800 Tx blocks
const DAMPEDBLOCKS = 1576800

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *config.ChainConfig // Chain configuration options
	bc     ChainContext        // Canonical block chain
	peer   StateProcessorPeer
}

type ShardReward struct {
	ShardId uint32
	GasFee  *big.Int
}

type ShardRewardList []ShardReward
func (p ShardRewardList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p ShardRewardList) Len() int           { return len(p) }
func (p ShardRewardList) Less(i, j int) bool { return p[i].GasFee.Cmp(p[j].GasFee) == -1 }

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *config.ChainConfig, bc ChainContext) *StateProcessor {
	return &StateProcessor{
		config: config,
		bc:     bc,
	}
}

func sortByGasFee(gasFeeUsedShards map[uint32]*big.Int) ShardRewardList {
	shardRewardList := make(ShardRewardList, len(gasFeeUsedShards))
	i := 0
	for k, v := range gasFeeUsedShards {
		shardRewardList[i] = ShardReward{k, v}
		i++
	}
	sort.Sort(shardRewardList)
	return shardRewardList
}


func assignRwards(block *protos.TxBlock, statedb *state.StateDB, gasFeeTotal *big.Int, gasFeeUsedShards map[uint32]*big.Int) {
	blockNum := block.Header.BlockNumber
	yieldReward := new(big.Int).SetUint64(50 * config.Okp >> (blockNum / DAMPEDBLOCKS))
	totalReward := new(big.Int).Add(gasFeeTotal, yieldReward)
	DSRewardTotal := new(big.Int)
	if len(gasFeeUsedShards) >= 1 {
		// sort shard gas fee in ascending order.
		shardsGasFeeList := sortByGasFee(gasFeeUsedShards)
		shardGasFeePercentile := new(big.Int).SetUint64(100)
		shardsBackupCnt := make(map[uint32]uint32)
		shardsCnt := len(gasFeeUsedShards)
		for i := 0; i < shardsCnt; i++ {
			// TODO:change to true backup count of shard
			shardsBackupCnt[shardsGasFeeList[i].ShardId] = 1
		}

		// DSMoreRate imploys that DS is assigned more 10 percent out totalReward than shard.
		DSMoreRate := new(big.Int).SetUint64(11)
		// compute shardAverageReward according to 1.1 * shardAverageReward + shardsCnt * shardAverageReward = totalReward.
		shardAverageReward := new(big.Int).Div(new(big.Int).Mul(totalReward, new(big.Int).SetUint64(10)),
			new(big.Int).Add(DSMoreRate, new(big.Int).Mul(new(big.Int).SetUint64(10), new(big.Int).SetUint64(uint64(shardsCnt)))))
		shardsRewardTotal := new(big.Int).Mul(new(big.Int).SetUint64(uint64(shardsCnt)), shardAverageReward)
		DSRewardTotal = DSRewardTotal.Sub(totalReward, shardsRewardTotal)
		// all shards reward except the last one.
		shardsRewardNoLast := new(big.Int)
		for i := 0; i < len(shardsGasFeeList)-1; i++ {
			shardGasFee := shardsGasFeeList[i]
			// calculate percent of shard in gasFeeTotal
			shardPercent := new(big.Int).Div(new(big.Int).Mul(shardGasFee.GasFee, shardGasFeePercentile), gasFeeTotal)
			shardReward := new(big.Int).Div(new(big.Int).Mul(shardsRewardTotal, shardPercent), shardGasFeePercentile)
			shardsRewardNoLast.Add(shardsRewardNoLast, shardReward)
			coreLogger.Debug("Sharding[", shardGasFee.ShardId, "]", shardReward)
			// TODO:assign shardReward to every node of shard in average.
			//shardNodeAverageReward := new(big.Int).Div(shardReward,
			//	new(big.Int).SetUint64(uint64(shardsBackupCnt[shardGasFee.ShardId]+1)))
			//for i := 0; i < int(shardsBackupCnt[shardGasFee.ShardId]); i++ {
			//	coreLogger.Debug("Sharding[", shardGasFee.ShardId, "]", "Backup : ", shardNodeAverageReward)
			//  statedb.AddBalance(common.BytesToAddress(block.Header.ShardingLeadCoinBase[shardGasFee.ShardId][i+1]), shardNodeAverageReward)
			//	shardReward.Sub(shardReward, shardNodeAverageReward)
			//}
			coreLogger.Debug("Sharding[", shardGasFee.ShardId, "]", "Leader : ", shardReward)
			statedb.AddBalance(common.BytesToAddress(block.Header.ShardingLeadCoinBase[shardGasFee.ShardId]), shardReward)
		}
		// assign last shard reward
		lastShardReward := new(big.Int).Sub(shardsRewardTotal, shardsRewardNoLast)
		coreLogger.Debug("Sharding[", shardsGasFeeList[len(shardsGasFeeList)-1].ShardId, "]", " : ", lastShardReward)
		//lastShardAverageReward := new(big.Int).Div(lastShardReward,
		//	new(big.Int).SetUint64(uint64(shardsBackupCnt[shardsGasFeeList[len(shardsGasFeeList)-1].ShardId]+1)))
		//for i := 0; i < int(shardsBackupCnt[shardsGasFeeList[len(shardsGasFeeList)-1].ShardId]); i++ {
		//	fmt.Println("Sharding[", shardsGasFeeList[len(shardsGasFeeList)-1].ShardId, "]", "Backup : ", lastShardAverageReward)
		//	lastShardReward.Sub(lastShardReward, lastShardAverageReward)
		//}
		coreLogger.Debug("Sharding[", shardsGasFeeList[len(shardsGasFeeList)-1].ShardId, "]", "Leader : ", lastShardReward)
		statedb.AddBalance(common.BytesToAddress(block.Header.ShardingLeadCoinBase[shardsGasFeeList[len(shardsGasFeeList)-1].ShardId]), lastShardReward)
	} else {
		DSRewardTotal.Set(totalReward)
	}
	// assign DS reward
	DsBackupCnt := new(big.Int).SetInt64(int64(len(block.Header.DSCoinBase) - 1))
	// DSLeaderMoreRate imploys that DSLeader is assigned more 10 percent out DSRewardTotal than shard.
	DSLeaderMoreRate := new(big.Int).SetUint64(11 )
	// compute shardAverageReward according to 1.1 * DSBackupAverageReward + DsBackupCnt * DSBackupAverageReward = totalReward.
	DSBackupAverageReward := new(big.Int).Div(new(big.Int).Mul(DSRewardTotal, new(big.Int).SetUint64(10 )),
		new(big.Int).Add(DSLeaderMoreRate, new(big.Int).Mul(new(big.Int).SetUint64(10 ), DsBackupCnt)))
	for i := 1; i < len(block.Header.DSCoinBase); i++ {
		statedb.AddBalance(common.BytesToAddress(block.Header.DSCoinBase[i]), DSBackupAverageReward)
		DSRewardTotal.Sub(DSRewardTotal, DSBackupAverageReward)
		coreLogger.Debug("DSBackup : ", DSBackupAverageReward)
	}
	statedb.AddBalance(common.BytesToAddress(block.Header.DSCoinBase[0]), DSRewardTotal)
	coreLogger.Debug("DSLeader : ", DSRewardTotal)
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
		gasFeeUsedShards = make(map[uint32]*big.Int)
		gasFeeTotal = new(big.Int)
	)
	// Iterate over and process the individual transactions
	for i, tx := range block.Body.Transactions {
		statedb.Prepare(tx.Hash(), block.Hash(), block.Header.BlockNumber, i)
		coreLogger.Debugf("tx data: %+v\n", tx)
		receipt, gas, err := ApplyTransaction(p.config, p.bc, nil, gp, statedb, header, tx, usedGas, cfg)
		if err != nil {
			return nil, nil, 0, err
		}

		// store used gas of tx according to shardId
		shardId := p.peer.GetShardId(tx, block.Header.DSBlockNum)
		txFee := new(big.Int).SetUint64(gas*tx.GasPrice)
		gasFeeTotal.Add(gasFeeTotal, txFee)
		if gasFeeUsedShards[shardId] == nil {
			gasFeeUsedShards[shardId] = new(big.Int)
		}
		gasFeeUsedShards[shardId].Add(gasFeeUsedShards[shardId], txFee)
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	// p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.DposContext)
	assignRwards(block, statedb, gasFeeTotal, gasFeeUsedShards)

	return receipts, allLogs, *usedGas, nil
}

func (p *StateProcessor) SetPeerServer(server StateProcessorPeer) {
	p.peer = server
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns an error if the transaction failed,
// indicating the block was invalid.
func ApplyTransaction(config *config.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *protos.TxBlockHeader, tx *protos.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, uint64, error) {
	coreLogger.Debug("ApplyTransaction\n")
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
