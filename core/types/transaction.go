/**
 * Copyright 2015 OKCoin Inc.
 * transaction.go - transaction struct
 * 2018/7/18. niwei create
 */

package types

import (
	"container/heap"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync/atomic"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/crypto"
)

var (
	ErrInvalidSig  = errors.New("invalid transaction v, r, s values")
	ErrInvalidType = errors.New("invalid transaction type")
)

const (
	Binary TxType = iota
	LoginCandidate
	LogoutCandidate
	Delegate
	UnDelegate
)

/**
 * @wei.ni
 * TxType 用来支持DPOS共识
**/
type TxType uint8

type Transaction struct {
	data txdata

	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type txdata struct {
	Type    TxType          `json:"type"     gencodec:"required"`
	Nonce   uint64          `json:"nonce"    gencodec:"required"`
	To      *common.Address `json:"to"       gencodec:"required"`
	Amount  *big.Int        `json:"value"    gencodec:"required"`
	Payload []byte          `json:"input"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	// This is only used when marshaling to JSON.
	Hash *common.Hash `json:"hash" rlp:"-"`
}

func NewTransaction(txType TxType, nonce uint64, to common.Address, amount *big.Int, data []byte) *Transaction {
	// compatible with raw transaction with to address is empty
	if txType != Binary && (to == common.Address{}) {
		return newTransaction(txType, nonce, nil, amount, data)
	}
	return newTransaction(txType, nonce, &to, amount, data)
}

func newTransaction(txType TxType, nonce uint64, to *common.Address, amount *big.Int, data []byte) *Transaction {
	if len(data) > 0 {
		data = common.CopyBytes(data)
	}

	d := txdata{
		Type:    txType,
		Nonce:   nonce,
		To:      to,
		Payload: data,
		Amount:  new(big.Int),
		V:       new(big.Int),
		R:       new(big.Int),
		S:       new(big.Int),
	}

	if amount != nil {
		d.Amount.Set(amount)
	}

	return &Transaction{data: d}
}

func (t *Transaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &t.data)
}

func (t *Transaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&t.data)
	if err == nil {
		t.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

func (t *Transaction) Hash() common.Hash {
	if hash := t.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}

	v := rlpHash(t)
	t.hash.Store(v)
	return v
}

func (t *Transaction) Size() common.StorageSize {
	if size := t.size.Load(); size != nil {
		return size.(common.StorageSize)
	}

	c := writeCounter(0)
	rlp.Encode(&c, &t.data)
	t.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

func (t *Transaction) Data() []byte     { return common.CopyBytes(t.data.Payload) }
func (t *Transaction) Value() *big.Int  { return new(big.Int).Set(t.data.Amount) }
func (t *Transaction) Nonce() uint64    { return t.data.Nonce }
func (t *Transaction) CheckNonce() bool { return true }
func (t *Transaction) Type() TxType     { return t.data.Type }

func (t *Transaction) To() *common.Address {
	if t.data.To == nil {
		return nil
	} else {
		to := *t.data.To
		return &to
	}
}

// func (tx *Transaction) AsMessage() (Message, error) {
// 	msg := Message{
// 		nonce:      tx.data.AccountNonce,
// 		to:         tx.data.Recipient,
// 		amount:     tx.data.Amount,
// 		data:       tx.data.Payload,
// 		checkNonce: true,
// 	}

// 	var err error
// 	msg.from, err = Sender(s, tx)
// 	return msg, err
// }

/**
 * @wei.ni
 * setSign 设置对当前交易的签名数据
 * input : sig []byte 签名数据
 * return: 成功 nil
**/
func (t *Transaction) setSign(sig []byte) error {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}

	t.data.R = new(big.Int).SetBytes(sig[:32])
	t.data.S = new(big.Int).SetBytes(sig[32:64])
	t.data.V = new(big.Int).SetBytes([]byte{sig[64] + 27})

	return nil
}

func (t *Transaction) PrintTransaction() {
	from, _ := Sender(t)
	fmt.Printf("	============ Transaction %x ============\n", t.Type())
	fmt.Printf("	From:       %s\n", from.String())
	fmt.Printf("	To:    		%s\n", (*t.To()).String())
	fmt.Printf("	Nonce:      %x\n", t.Nonce())
	fmt.Printf("	Amount:     %s\n", t.Value().String())
	fmt.Printf("\n")
}

/**
 * @xueyang.han
 * GetSign, synatize v, s, r to bytes
**/
func (t *Transaction) GetSign() []byte {
	sig := make([]byte, 65)
	v, r, s := t.SignatureValiues()
	copy(sig[:32], r.Bytes())
	copy(sig[32:64], s.Bytes())
	copy(sig[64:], v.Bytes())

	return sig
}

/**
 * @wei.ni
 * SignatureValiues 返回交易的签名信息
 * input: 无
 * return value, V, R, S
**/
func (t *Transaction) SignatureValiues() (*big.Int, *big.Int, *big.Int) {
	return t.data.V, t.data.R, t.data.S
}

/**
 * @wei.ni
 * SigHash
 * input: 无
 * return: common.Hash
**/
func (t *Transaction) SigHash() common.Hash {
	return rlpHash([]interface{}{
		t.data.Nonce,
		t.data.Type,
		t.data.Payload,
		t.data.Amount,
	})
}

/**
 * @wei.ni
 * SignTx 签名交易
 * input:
 * 		tx: 	需要签名的交易
 * 		prv: 	签名账号的私钥
 * return:
 * 		成功 返回签名后的交易 tx, 无error
 * 		失败 返回 nil和错误信息
**/

func SignTx(tx *Transaction, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := tx.SigHash()
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}

	if err = tx.setSign(sig); err != nil {
		return nil, err
	}

	return tx, nil
}

/**
 * @wei.ni
 * Sender 获取当前交易的签名者的 Address
 * input:
 * 		t 交易
 * return:
 * 		成功 返回签名者的地址
 * 		失败 返回空地址和错误信息
**/
func Sender(t *Transaction) (common.Address, error) {
	if from := t.from.Load(); from != nil {
		return from.(common.Address), nil
	}
	sighash := t.SigHash()

	if t.data.V.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}

	V := byte(t.data.V.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, t.data.R, t.data.S) {
		return common.Address{}, ErrInvalidSig
	}

	r, s := t.data.R.Bytes(), t.data.S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V

	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}

	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	t.from.Store(addr)
	return addr, nil
}

// Transactions is a Transaction slice type for basic sorting.
type Transactions []*Transaction

// Len returns the length of s.
func (s Transactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s Transactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// TxDifference returns a new set t which is the difference between a to b.
func TxDifference(a, b Transactions) (keep Transactions) {
	keep = make(Transactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// TxByNonce implements the sort interface to allow sorting a list of transactions
// by their nonces. This is usually only useful for sorting transactions from a
// single account, otherwise a nonce comparison doesn't make much sense.
type TxByNonce Transactions

func (s TxByNonce) Len() int           { return len(s) }
func (s TxByNonce) Less(i, j int) bool { return s[i].data.Nonce < s[j].data.Nonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

func (s *TxByNonce) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}

func (s *TxByNonce) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

type TransactionsByNonce struct {
	txs   map[common.Address]Transactions // Per account nonce-sorted list of transactions
	heads TxByNonce                       // Next transaction for each unique account
}

// NewTransactionsByNonce creates a transaction set that can retrieve
// sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByNonce(txs map[common.Address]Transactions) *TransactionsByNonce {
	// Initialize a nonce based heap with the head transactions
	heads := make(TxByNonce, 0, len(txs))
	for _, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := Sender(accTxs[0])
		txs[acc] = accTxs[1:]
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByNonce{
		txs:   txs,
		heads: heads,
	}
}

// Peek returns the next transaction by nonce.
func (t *TransactionsByNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByNonce) Shift() {
	acc, _ := Sender(t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *TransactionsByNonce) Pop() {
	heap.Pop(&t.heads)
}

// AsMessage returns the transaction as a core.Message.
//
// AsMessage requires a signer to derive the sender.
//
// XXX Rename message to something less arbitrary?
func (tx *Transaction) AsMessage() (Message, error) {
	return Message{}, nil
}

// Message is a fully derived transaction and implements core.Message
//
// NOTE: In a future PR this will be removed.
type Message struct {
	from       common.Address
	to         *common.Address
	nonce      uint64
	amount     uint64
	gasLimit   uint64
	gasPrice   uint64
	data       []byte
	checkNonce bool
}

func NewMessage(from common.Address, to *common.Address, nonce uint64, amount uint64, gasLimit uint64, gasPrice uint64, data []byte, checkNonce bool) Message {
	return Message{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gasLimit,
		gasPrice:   gasPrice,
		data:       data,
		checkNonce: checkNonce,
	}
}

func (m Message) From() common.Address { return m.from }
func (m Message) To() *common.Address  { return m.to }
func (m Message) GasPrice() uint64     { return m.gasPrice }
func (m Message) Value() uint64        { return m.amount }
func (m Message) Gas() uint64          { return m.gasLimit }
func (m Message) Nonce() uint64        { return m.nonce }
func (m Message) Data() []byte         { return m.data }
func (m Message) CheckNonce() bool     { return m.checkNonce }
