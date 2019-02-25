// Copyright The go-okchain Authors 2018,  All rights reserved.

/*
Copyright 2018 The Okchain Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package role

import (
	"errors"
)

var (
	ErrInvalidMessage = errors.New("invalid message")

	ErrMarshalMessage = errors.New("marshal message failed")

	ErrUnmarshalMessage = errors.New("unmarshal message failed")

	ErrVerifyMessage = errors.New("bls message verify failed")

	ErrComposeMessage = errors.New("compose message failed")

	ErrTargetState = errors.New("invalid target state")

	ErrDuplicatedMessage = errors.New("message received before")

	ErrDuplicatedBlock = errors.New("block inserted before")

	ErrInsertBlock = errors.New("insert block failed")

	ErrHandleInCurrentState = errors.New("cannot handle in current state")

	ErrInsufficientCommittee = errors.New("insufficient committee nodes")

	ErrEmptyBlock = errors.New("block is nil")

	ErrVerifyBlock = errors.New("block verify failed")

	ErrMine = errors.New("mine failed")

	ErrEmptyPoWList = errors.New("pow list is empty")

	ErrPubKeyConvert = errors.New("pubkey convert failed")

	ErrVerifyMine = errors.New("mine result verify failed")

	ErrBLSPubKeyVerify = errors.New("bls multi pubkey verify failed")

	ErrComposeBlock = errors.New("compose block failed")

	ErrVerifyTransaciton = errors.New("transaction verify failed")

	ErrMultiCastMessage = errors.New("multicast message failed")

	ErrSignMessage = errors.New("message sign failed")

	ErrGetCoinbase = errors.New("get coinbase failed")
)
