// Copyright The go-okchain Authors 2018,  All rights reserved.

// Copyright 2014 The go-ethereum Authors
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

package blockchain

import (
	"bytes"
	"fmt"
	"math/big"
	"runtime"
	"time"
)

import (
	"sort"

	"github.com/ok-chain/okchain/common"
	params "github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/protos"
)

type Validator interface {

	// ValidateHeader, concurrently ValidateHeader, ValidateSeal
	VerifyHeader(header *protos.TxBlockHeader) error

	// Validate block signature.
	VerifySeal(blk *protos.TxBlock) error

	// ValidateBody validates the given block's content.
	ValidateBody(block *protos.TxBlock) error

	// ValidateState validates the given statedb and optionally the receipts and gas used.
	ValidateState(block, parent *protos.TxBlock, state *state.StateDB, receipts types.Receipts, usedGas uint64) error
}

type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	txBC   ITxBlockchain       // Canonical TX block chain
	dsBC   IBasicBlockChain    // Canonical DS block chain
}

func NewBlockValidator(config *params.ChainConfig, txBC ITxBlockchain, dsBC IBasicBlockChain) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		txBC:   txBC,
		dsBC:   dsBC,
	}
	return validator
}

// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine. Verifying the seal may be done optionally here, or explicitly
// via the VerifySeal method.
func (v *BlockValidator) VerifyHeader(header *protos.TxBlockHeader) (err error) {

	// ErrKnownBlock
	log.Infof("header hash: %s, previous hash: %s, blk no: %d",
		common.Bytes2Hex(header.Hash().Bytes()), common.Bytes2Hex(header.GetPreviousBlockHash()), header.GetBlockNumber())
	blk := v.txBC.GetBlock(header.Hash(), header.BlockNumber)
	log.Infof("%+v", blk)
	if blk != nil {
		return ErrKnownBlock
	}

	// ErrUnknownAncestor
	phash := common.Hash{}
	phash.SetBytes(header.PreviousBlockHash)
	pBlk := v.txBC.GetBlock(phash, header.BlockNumber-1)
	if pBlk == nil {
		return ErrUnknownAncestor
	}

	parentHeader := pBlk.(*protos.TxBlock).GetHeader()
	err = v.verifyHeader(header, parentHeader)
	return err
}

func (v *BlockValidator) verifyHeader(header, parent *protos.TxBlockHeader) error {

	// Verify timestamp
	if (header.Timestamp == nil) || (header.Timestamp.GetNanosecond() == 0 || header.Timestamp.GetSecond() == 0) {
		return ErrTimestamp
	}

	t := big.Int{}
	t.SetInt64(protos.Timestamp2Time(header.Timestamp).Unix())
	if t.Cmp(big.NewInt(time.Now().Add(AllowedFutureBlkTime).Unix())) > 0 {
		return ErrFutureBlock
	}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// Verify block number.
	if (header.BlockNumber - parent.BlockNumber) > 1 {
		return ErrFutureBlock
	}

	// Verify DS block exists.
	if v.dsBC != nil {
		if dsBlock := v.dsBC.GetBlock(common.BytesToHash(header.GetDSBlockHash()), header.DSBlockNum); dsBlock == nil {
			return ErrDSBlockNotExist
		}
	}

	return nil
}

// Flt. 20181204. Concurrent verify header worker.
func (v *BlockValidator) verifyHeaderWorker(headers []*protos.TxBlockHeader, seals []bool, index int) error {

	blk := v.txBC.GetBlock(headers[index].Hash(), headers[index].BlockNumber)
	if blk != nil {
		return ErrKnownBlock
	}

	var parent *protos.TxBlockHeader
	if index == 0 {
		pblk := v.txBC.GetBlock(headers[0].ParentHash(), headers[0].BlockNumber-1)
		if pblk == nil {
			return ErrUnknownAncestor
		}
		parent = pblk.(*protos.TxBlock).GetHeader()
	} else if headers[index-1].Hash() == headers[index].ParentHash() {
		parent = headers[index-1]
	}

	return v.verifyHeader(headers[index], parent)
}

// Flt. 20181017. Not fully test becoz batch insert is not supported yet.
// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (v *BlockValidator) VerifyHeaders(headers []*protos.TxBlockHeader, seals []bool) (chan<- struct{}, <-chan error) {

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = v.verifyHeaderWorker(headers, seals, index)
				log.Infof("Finish to verifyHeader %+v, %+v, %d, %+v", headers[index], seals[index], index, errors[index])
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

// VerifySeal checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (v *BlockValidator) VerifySeal(block *protos.TxBlock) error {

	header := block.GetHeader()

	// check multi pubkey & signature.
	if header.BlockNumber > 1 && header.DSBlockNum >= 1 {
		if v.dsBC == nil || v.dsBC.(*DSBlockChain).GetBlsSigner() == nil {
			return ErrBlockSignerNotFound
		}

		if baseDSBlk := v.dsBC.GetBlockByNumber(header.DSBlockNum); baseDSBlk == nil {
			return ErrDSBlockNotExist
		} else {

			baseBlk := baseDSBlk.(*protos.DSBlock)
			blsSigner := v.dsBC.(*DSBlockChain).GetBlsSigner()

			txLogger.Debugf("ParentDSBlock: %+v, Before Sort Committee: %+v, BoolMap: %+v, BaseBlockNO: %d",
				baseBlk, baseBlk.Body.GetCommittee(), baseBlk.GetHeader().GetBoolMap(), baseBlk.NumberU64())
			composedPubkeys := header.GetMultiPubKey()
			sortedCommittee := []*protos.PeerEndpoint{}
			for i := 0; i < len(baseBlk.Body.GetCommittee()); i++ {
				sortedCommittee = append(sortedCommittee, baseBlk.Body.GetCommittee()[i])
			}
			sort.Sort(protos.ByPubkey{PeerEndpointList: sortedCommittee})
			if err := protos.BlsMultiPubkeyVerify(header.GetBoolMap(), sortedCommittee, composedPubkeys); err != nil {
				for i := 0; i < len(sortedCommittee); i++ {
					txLogger.Debugf("sortedCommittee[%d]: %s", i, common.Bytes2Hex(sortedCommittee[i].Pubkey))
				}
				txLogger.Debugf("BlsMultiPubkeyVerify. Error: %s, BoolMap: %+v, sortedCommittee: %+v, header: %+v",
					err.Error(), header.GetBoolMap(), sortedCommittee, header)
				return ErrMismatchMultiPubkey
			}

			txLogger.Debugf("check multi pubkey & signature. header: %+v", header)
			coinbase := block.Header.DSCoinBase
			block.Header.DSCoinBase = [][]byte{block.Header.DSCoinBase[0]}
			ret, err := blsSigner.VerifyHash(block.MHash().Bytes(), header.GetSignature(), header.GetMultiPubKey())
			if err != nil || !ret {
				txLogger.Debugf("verifyHeader. header: %+v, currentDSBlockNO: %d", header, header.DSBlockNum)
				return ErrMismatchMultiSignature
			}
			block.Header.DSCoinBase = coinbase

			if err := block.CheckCoinBase2(sortedCommittee, baseBlk); err != nil {
				txLogger.Debug(err.Error())
				return ErrCheckCoinBase
			}

			// Verify Bitmap in Header
			if block.GetVoteTickets() < GetToleranceSize(baseBlk.GetBody().GetCommittee()) {
				return ErrBitmapTolerence
			}
		}
	}

	return nil
}

// FLT. 20181010. Uncle's not designed in OKChain.
// VerifyUncles verifies that the given block's uncles conform to the consensus
//func (v *BlockValidator) ValidateUncles(block *protos.TxBlock) error {
//	return nil
//}

func (v *BlockValidator) ValidateBody(block *protos.TxBlock) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}

	if !v.HasBlock(block.ParentHash(), block.NumberU64()-1) {
		return ErrUnknownAncestor
	}

	if !v.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		return ErrPrunedAncestor
	}

	// Check transactions
	header := block.GetHeader()
	trxs := protos.Transactions{}
	trxs = block.GetBody().Transactions

	// FLT. 20181020. Adapt protobuf's bug of removing default value item to reduce communitcation data.
	trxValid := (header.TxNum == uint64(0) && (trxs == nil || len(trxs) == 0))
	if !trxValid && (trxs == nil && header.TxNum > 0) || header.TxNum != uint64(trxs.Len()) {
		return ErrTransations
	}

	expectedTxRoot := types.DeriveSha(trxs)
	if bytes.Compare(expectedTxRoot.Bytes(), header.GetTxRoot()) != 0 {
		return nil
		// FLT. 20181019. Quit
		// return ErrTransations
	}

	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block, parent *protos.TxBlock, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {

	header := block.GetHeader()

	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(true); bytes.Compare(header.GetStateRoot(), root.Bytes()) != 0 {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root(), root)
	}
	return nil
}

// HasBlock checks if a block is fully present in the database or not.
func (bc *BlockValidator) HasBlock(hash common.Hash, number uint64) bool {
	blk := bc.txBC.GetBlock(hash, number)
	return blk != nil
}

// HasState checks if state trie is fully present in the database or not.
func (bc *BlockValidator) HasState(hash common.Hash) bool {
	_, err := bc.txBC.StateDB().OpenTrie(hash)
	return err == nil
}

// HasBlockAndState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (bc *BlockValidator) HasBlockAndState(hash common.Hash, number uint64) bool {
	// Check first that the block itself is known
	block := bc.txBC.GetBlock(hash, number)
	return block != nil && bc.HasState(block.(*protos.TxBlock).Root())
}
