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
	"errors"
)

var (
	// ErrKnownBlock is returned when a block to import is already known locally.
	ErrKnownBlock = errors.New("block already known")

	// ErrBlacklistedHash is returned if a block to import is on the blacklist.
	ErrBlacklistedHash = errors.New("blacklisted hash")

	// ErrNonceTooHigh is returned if the nonce of a transaction is higher than the
	// next one expected based on the local chain.
	ErrNonceTooHigh = errors.New("nonce too high")

	ErrNoGenesis = errors.New("Genesis not found in chain")

	ErrDuplicatedBlock = errors.New("block inserted before")

	// ErrUnknownAncestor is returned when validating a block requires an ancestor
	// that is unknown.
	ErrUnknownAncestor = errors.New("unknown ancestor")

	// ErrFutureBlock is returned when a block's timestamp is in the future according
	// to the current node.
	ErrFutureBlock = errors.New("block in the future")

	// ErrInvalidNumber is returned if a block's number doesn't equal it's parent's
	// plus one.
	ErrInvalidNumber = errors.New("invalid block number")

	ErrTimestamp = errors.New("timestamp incorrect")

	ErrPrunedAncestor = errors.New("pruned ancestor")

	ErrConcurrentCreated = errors.New("instance is created by other goroutine")

	ErrTransations = errors.New("transaction incorrect")

	ErrDSBlockNotExist = errors.New("ds block not exists")

	ErrTxBlockNotExist = errors.New("tx block not exists")

	ErrBlockSignerNotFound = errors.New("block signer not found")

	ErrMismatchMultiPubkey = errors.New("multi pubkey mismatch")

	ErrMismatchMultiSignature = errors.New("multi signature mismatch")

	ErrNotImplemented = errors.New("not implemented yet")

	ErrConvertionFail = errors.New("Fail to convert")

	ErrBitmapTolerence = errors.New("No sufficient tickets in bitmap or boolmap")

	ErrCheckCoinBase = errors.New("check coinbase failed")
)
