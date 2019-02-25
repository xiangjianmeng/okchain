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

package blspbft

import (
	"errors"
)

var (
	ErrVerifyBlock = errors.New("block verify failed")

	ErrComposeMessage = errors.New("compose message failed")

	ErrInvalidMessage = errors.New("invalid message")

	ErrIncorrectMessageSource = errors.New("incorrect message source")

	ErrMultiCastMessage = errors.New("multicast message failed")

	ErrUnmarshalMessage = errors.New("unmarshal message failed")

	ErrVerifyMessage = errors.New("bls message verify failed")

	ErrBLSPubKeyVerify = errors.New("bls multi pubkey verify failed")

	ErrCheckCoinBase = errors.New("check coinbase failed")

	ErrGetCoinBase = errors.New("get coinbase failed")

	ErrCalStateRoot = errors.New("calculate state root failed")
)
