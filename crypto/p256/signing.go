// Copyright The go-okchain Authors 2018,  All rights reserved.

package p256

import (
	types "github.com/ok-chain/okchain/protos"
)

// SignTx signs the transaction using the given signer and private key
func SignTx(t *types.Transaction, signer *P256Signer, prvBytes []byte) (*types.Transaction, error) {
	prv, err := signer.BytesToPrvKey(prvBytes)
	if err != nil {
		return nil, err
	}
	t.SenderPubKey = signer.PubKeyToBytes(&prv.PublicKey)

	h := t.Hash()
	sig, err := signer.SignHash(h[:], prvBytes)
	if err != nil {
		return nil, err
	}
	t.Signature = sig
	return t, nil
}

// SignTx signs the transaction using the given signer and private key
func SignTx2(t *types.Transaction, signer *P256Signer, prvBytes []byte) (sig []byte, error error) {
	prv, err := signer.BytesToPrvKey(prvBytes)
	if err != nil {
		return nil, err
	}
	t.SenderPubKey = signer.PubKeyToBytes(&prv.PublicKey)

	h := t.Hash()
	return signer.SignHash(h[:], prvBytes)
}
