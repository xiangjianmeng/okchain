// Copyright The go-okchain Authors 2018,  All rights reserved.

package protos

import (
	"container/heap"
	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/util"
)

//Transactions ...
type Transactions []*Transaction

//Transaction
func NewTransaction(nonce uint64, pubKey []byte, to common.Address, amount uint64, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, pubKey, &to, amount, gasLimit, gasPrice, data)
}

func NewContractCreation(nonce uint64, pubKey []byte, amount uint64, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return newTransaction(nonce, pubKey, nil, amount, gasLimit, gasPrice, data)
}

func newTransaction(nonce uint64, pubKey []byte, to *common.Address, amount uint64, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	t := &Transaction{}
	t.Version = uint32(1)
	t.Nonce = nonce
	t.SenderPubKey = common.CopyBytes(pubKey)
	if to == nil {
		t.ToAddr = nil
	} else {
		t.ToAddr = to[:]
	}

	t.Amount = amount
	t.GasLimit = gasLimit
	if gasPrice != nil {
		t.GasPrice = gasPrice.Uint64()
	}
	t.Data = common.CopyBytes(data)
	t.Timestamp = CreateUtcTimestamp()
	return t
}

// type Transaction struct {
// 	Version              uint32     `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
// 	Nonce                uint64     `protobuf:"varint,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
// 	SenderPubKey         []byte     `protobuf:"bytes,3,opt,name=senderPubKey,proto3" json:"senderPubKey,omitempty"`
// 	ToAddr               []byte     `protobuf:"bytes,4,opt,name=toAddr,proto3" json:"toAddr,omitempty"`
// 	Amount               uint64     `protobuf:"varint,5,opt,name=amount,proto3" json:"amount,omitempty"`
// 	Signature            []byte     `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
// 	GasPrice             uint64     `protobuf:"varint,7,opt,name=gasPrice,proto3" json:"gasPrice,omitempty"`
// 	GasLimit             uint64     `protobuf:"varint,8,opt,name=gasLimit,proto3" json:"gasLimit,omitempty"`
// 	Code                 []byte     `protobuf:"bytes,9,opt,name=code,proto3" json:"code,omitempty"`
// 	Data                 []byte     `protobuf:"bytes,10,opt,name=data,proto3" json:"data,omitempty"`
// 	Timestamp            *Timestamp `protobuf:"bytes,15,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
// 	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
// 	XXX_unrecognized     []byte     `json:"-"`
// 	XXX_sizecache        int32      `json:"-"`
// }
func (t *Transaction) Hash() (h common.Hash) {
	//cant't has signature
	return util.Hash([]interface{}{t.Version, t.Nonce, t.SenderPubKey, t.ToAddr, t.Amount, t.GasPrice, t.GasLimit, t.Code, t.Data, t.Timestamp})
}
func (t *Transaction) Gas() uint64      { return t.GasLimit }
func (tx *Transaction) Value() *big.Int { return new(big.Int).SetUint64(tx.Amount) }

// func (t *Transaction) GasPrice() *big.Int {
// 	return new(big.Int).SetUint64(t.GasPrice)
// }

// AsMsg returns the transaction as a core.Msg.
//
// AsMsg requires a signer to derive the sender.
//
// XXX Rename Msg to something less arbitrary?
func (tx *Transaction) AsMsg() (*Msg, error) {
	msg := &Msg{
		nonce: tx.Nonce,
		//gasLimit: tx.GasLimit,
		//gasPrice: new(big.Int).SetUint64(tx.GasPrice),
		gasLimit: 10000000,
		gasPrice: new(big.Int).SetUint64(1),
		// from:     common.BytesToAddress(tx.SenderPubKey),
		//to:       &(common.BytesToAddress(tx.ToAddr)),
		amount:     new(big.Int).SetUint64(tx.Amount),
		data:       tx.Data,
		checkNonce: true,
	}
	if tx.ToAddr != nil {
		receipt := common.BytesToAddress(tx.ToAddr)
		msg.to = &receipt
	}
	signer := &P256Signer{}
	pub := signer.BytesToPubKey(tx.SenderPubKey)
	msg.from = signer.PubKeyToAddress(pub)

	return msg, nil
}

// Msg is a fully derived transaction and implements core.Msg
//
// NOTE: In a future PR this will be removed.
type Msg struct {
	to         *common.Address
	from       common.Address
	nonce      uint64
	amount     *big.Int
	gasLimit   uint64
	gasPrice   *big.Int
	data       []byte
	checkNonce bool
}

func NewMsg(from common.Address, to *common.Address, nonce uint64, amount *big.Int, gas uint64, gasPrice *big.Int, data []byte, checkNonce bool) *Msg {
	return &Msg{
		from:       from,
		to:         to,
		nonce:      nonce,
		amount:     amount,
		gasLimit:   gas,
		gasPrice:   gasPrice,
		data:       data,
		checkNonce: checkNonce,
	}
}

func (m Msg) From() common.Address { return m.from }
func (m Msg) To() *common.Address  { return m.to }
func (m Msg) GasPrice() *big.Int   { return m.gasPrice }
func (m Msg) Value() *big.Int      { return m.amount }
func (m Msg) Gas() uint64          { return m.gasLimit }
func (m Msg) Nonce() uint64        { return m.nonce }
func (m Msg) Data() []byte         { return m.data }
func (m Msg) CheckNonce() bool     { return m.checkNonce }

func (tx *Transaction) CheckNonce() bool { return true }

// To returns the recipient address of the transaction.
// It returns nil if the transaction is a contract creation.
func (tx *Transaction) To() *common.Address {
	if tx.ToAddr == nil {
		return nil
	}
	to := common.BytesToAddress(tx.ToAddr)
	return &to
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *Transaction) Size() common.StorageSize {
	// if size := tx.size.Load(); size != nil {
	// 	return size.(common.StorageSize)
	// }
	// c := writeCounter(0)
	// rlp.Encode(&c, &tx.data)
	// tx.size.Store(common.StorageSize(c))
	// return common.StorageSize(c)
	enc, _ := proto.Marshal(tx)
	return common.StorageSize(len(enc))
}

// Cost returns amount + gasprice * gaslimit.
func (tx *Transaction) Cost() *big.Int {
	total := new(big.Int).Mul(new(big.Int).SetUint64(tx.GasPrice), new(big.Int).SetUint64(tx.GasLimit))
	total.Add(total, new(big.Int).SetUint64(tx.Amount))
	return total
}

func (s Transactions) Len() int { return len(s) }

func (s Transactions) GetRlp(i int) []byte {
	// GetRlp implements Rlpable and returns the i'th element of s in rlp.
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// Swap swaps the i'th and the j'th element in s.
func (s Transactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// TxDifference returns a new set which is the difference between a and b.
func TxDifference(a, b Transactions) Transactions {
	keep := make(Transactions, 0, len(a))

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
func (s TxByNonce) Less(i, j int) bool { return s[i].Nonce < s[j].Nonce }
func (s TxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type TxByPrice Transactions

func (s TxByPrice) Len() int { return len(s) }
func (s TxByPrice) Less(i, j int) bool {
	return new(big.Int).SetUint64(s[i].GasPrice).Cmp(new(big.Int).SetUint64(s[j].GasPrice)) > 0
}
func (s TxByPrice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *TxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*Transaction))
}

func (s *TxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

// TransactionsByPriceAndNonce represents a set of transactions that can return
// transactions in a profit-maximizing sorted order, while supporting removing
// entire batches of transactions for non-executable accounts.
type TransactionsByPriceAndNonce struct {
	txs   map[common.Address]Transactions // Per account nonce-sorted list of transactions
	heads TxByPrice                       // Next transaction for each unique account (price heap)
}

// NewTransactionsByPriceAndNonce creates a transaction set that can retrieve
// price sorted transactions in a nonce-honouring way.
//
// Note, the input map is reowned so the caller should not interact any more with
// if after providing it to the constructor.
func NewTransactionsByPriceAndNonce(txs map[common.Address]Transactions) *TransactionsByPriceAndNonce {
	signer := P256Signer{}
	// Initialize a price based heap with the head transactions
	heads := make(TxByPrice, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := signer.Sender(accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &TransactionsByPriceAndNonce{
		txs:   txs,
		heads: heads,
	}
}

// Peek returns the next transaction by price.
func (t *TransactionsByPriceAndNonce) Peek() *Transaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *TransactionsByPriceAndNonce) Shift() {
	signer := MakeSigner("")
	acc, _ := signer.Sender(t.heads[0])
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
func (t *TransactionsByPriceAndNonce) Pop() {
	heap.Pop(&t.heads)
}
