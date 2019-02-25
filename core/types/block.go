/**
 * Copyright 2015 OKCoin Inc.
 * Block.go - block struct
 * 2018/7/18. niwei create
 * 2018/7/18. niwei 增加对DOPS的支持
 */

package types

import (
	"fmt"
	"io"
	"math/big"
	"sort"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
)

var (
	EmptyRootHash = DeriveSha(Transactions{})
)

type Header struct {
	ParentHash common.Hash    `json:"parentHash"       gencodec:"required"`
	Validator  common.Address `json:"validator"        gencodec:"required"` //@xq DPOS
	Coinbase   common.Address `json:"miner"            gencodec:"required"`
	Root       common.Hash    `json:"stateRoot"        gencodec:"required"`
	TxHash     common.Hash    `json:"transactionsRoot" gencodec:"required"`
	// ReceiptHash common.Hash       `json:"receiptsRoot"     gencodec:"required"`
	DposContext *DposContextProto `json:"dposContext"      gencodec:"required"` //@wei.ni DPOS
	Number      *big.Int          `json:"number"           gencodec:"required"`
	Time        *big.Int          `json:"timestamp"        gencodec:"required"`
	Extra       []byte            `json:"extraData"        gencodec:"required"`
	Reward      *big.Int          `json:"reward"           gencodec:"required"` //@wei.ni 奖励
}

func (h *Header) Hash() common.Hash {
	return rlpHash(h)
}

func (h *Header) Size() common.StorageSize {
	return common.StorageSize(unsafe.Sizeof(*h)) + common.StorageSize(len(h.Extra)+(h.Number.BitLen()+h.Time.BitLen())/8)
}

func (h *Header) PrintHerder() {
	fmt.Printf("============ Header %s ============\n", h.Number.String())
	fmt.Printf("PrevHash:    	%s\n", h.ParentHash.String())
	fmt.Printf("validator:      %s\n", h.Validator.String())
	fmt.Printf("hash:        	%s\n", h.Hash().String())
	fmt.Printf("\n\n")
}

// Body is a simple (mutable, non-safe) data container for storing and moving
// a block's data contents (transactions) together.
type Body struct {
	Transactions []*Transaction
}

type Block struct {
	header       *Header
	transactions Transactions

	size atomic.Value
	hash atomic.Value

	DposContext *DposContext //@wei.ni DPOS
	signs       []byte

	ReceivedAt   time.Time
	ReceivedFrom interface{}
}

type extblock struct {
	Header *Header
	Txs    Transactions
	Signs  []byte
}

func NewBlock(header *Header, txs Transactions) *Block {
	b := &Block{header: CopyHeader(header), transactions: txs}

	if len(txs) == 0 {
		b.header.TxHash = EmptyRootHash
	} else {
		b.header.TxHash = DeriveSha(Transactions(txs))
		b.transactions = make(Transactions, len(txs))
		copy(b.transactions, txs)
	}

	// b.header.ReceiptHash = EmptyRootHash

	return b
}

func NewBlockWithHeader(header *Header) *Block {
	return &Block{header: CopyHeader(header)}
}

// CopyHeader creates a deep copy of a block header to prevent side effects from
// modifying a header variable.
func CopyHeader(h *Header) *Header {
	cpy := *h
	if cpy.Time = new(big.Int); h.Time != nil {
		cpy.Time.Set(h.Time)
	}

	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}

	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}

	// add dposContextProto to header
	cpy.DposContext = &DposContextProto{}
	if h.DposContext != nil {
		cpy.DposContext = h.DposContext
	}
	return &cpy
}

// WithSeal returns a new block with the data from b but the header replaced with
// the sealed one.
func (b *Block) WithSeal(header *Header) *Block {
	cpy := *header

	return &Block{
		header:       &cpy,
		transactions: b.transactions,

		// add dposcontext
		DposContext: b.DposContext,
	}
}

// WithBody returns a new block with the given transaction and uncle contents.
func (b *Block) WithBody(transactions []*Transaction) *Block {
	block := &Block{
		header:       CopyHeader(b.header),
		transactions: make([]*Transaction, len(transactions)),
	}
	copy(block.transactions, transactions)

	return block
}

func (b *Block) Hash() common.Hash {
	if hash := b.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}

	v := b.header.Hash()
	b.hash.Store(v)
	return v
}

func (b *Block) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, extblock{
		Header: b.header,
		Txs:    b.transactions,
		Signs:  b.signs,
	})
}

func (b *Block) DecodeRLP(s *rlp.Stream) error {
	var eb extblock
	_, size, _ := s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}

	b.header, b.transactions, b.signs = eb.Header, eb.Txs, eb.Signs
	b.size.Store(common.StorageSize(rlp.ListSize(size)))
	return nil
}

func (b *Block) Number() *big.Int          { return new(big.Int).Set(b.header.Number) }
func (b *Block) Time() *big.Int            { return new(big.Int).Set(b.header.Time) }
func (b *Block) Validator() common.Address { return b.header.Validator }
func (b *Block) NumberU64() uint64         { return b.header.Number.Uint64() }
func (b *Block) Coinbase() common.Address  { return b.header.Coinbase }
func (b *Block) Root() common.Hash         { return b.header.Root }
func (b *Block) ParentHash() common.Hash   { return b.header.ParentHash }
func (b *Block) TxHash() common.Hash       { return b.header.TxHash }

// func (b *Block) ReceiptHash() common.Hash  { return b.header.ReceiptHash }
func (b *Block) Extra() []byte { return common.CopyBytes(b.header.Extra) }

func (b *Block) Reward() *big.Int { return new(big.Int).Set(b.header.Reward) }

func (b *Block) Header() *Header { return CopyHeader(b.header) }

// Body returns the non-header content of the block.
func (b *Block) Body() *Body           { return &Body{b.transactions} }
func (b *Block) DposCtx() *DposContext { return b.DposContext }
func (b *Block) Transactions() Transactions {
	return b.transactions
}

func (b *Block) PrintBlock() {
	fmt.Printf("============ Block %s ============\n", b.Header().Number.String())
	fmt.Printf("PrevHash:    	%s\n", b.Header().ParentHash.String())
	fmt.Printf("validator:      %s\n", b.Header().Validator.String())
	fmt.Printf("hash:        	%s\n", b.Header().Hash().String())
	fmt.Printf("tx size:        %d\n", b.Transactions().Len())

	for _, tx := range b.Transactions() {
		tx.PrintTransaction()
	}

	fmt.Printf("\n\n")
}

func (b *Block) Transaction(hash common.Hash) *Transaction {
	for _, transaction := range b.transactions {
		if transaction.Hash() == hash {
			return transaction
		}
	}
	return nil
}

func (b *Block) Size() common.StorageSize {
	if size := b.size.Load(); size != nil {
		return size.(common.StorageSize)
	}

	c := writeCounter(0)
	rlp.Encode(&c, b)
	b.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

type writeCounter common.StorageSize

func (c *writeCounter) Write(b []byte) (int, error) {
	*c += writeCounter(len(b))
	return len(b), nil
}

type Blocks []*Block

type BlockBy func(b1, b2 *Block) bool

func (self BlockBy) Sort(blocks Blocks) {
	bs := blockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type blockSorter struct {
	blocks Blocks
	by     func(b1, b2 *Block) bool
}

func (self blockSorter) Len() int { return len(self.blocks) }
func (self blockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self blockSorter) Less(i, j int) bool { return self.by(self.blocks[i], self.blocks[j]) }

func Number(b1, b2 *Block) bool { return b1.header.Number.Cmp(b2.header.Number) < 0 }
