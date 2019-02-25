// Copyright The go-okchain Authors 2018,  All rights reserved.

package protos

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"bytes"
	"reflect"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/crypto/multibls"
	"github.com/ok-chain/okchain/util"
)

type IBlock interface {
	Hash() (h common.Hash)
	MHash() (h common.Hash)
	NumberU64() uint64
	ParentHash() common.Hash
	ToReadable() (s string, err error)
	GetVoteTickets() int
}

func BlockNumberCompare(b1, b2 IBlock) bool {
	return b1.NumberU64() < b2.NumberU64()
}

type IBlocks []IBlock
type IBlockBy func(b1, b2 IBlock) bool

func (self IBlockBy) Sort(blocks IBlocks) {
	bs := IBlockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type IBlockSorter struct {
	blocks IBlocks
	by     func(b1, b2 IBlock) bool
}

func (self IBlockSorter) Len() int {
	return len(self.blocks)
}
func (self IBlockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}
func (self IBlockSorter) Less(i, j int) bool {
	return self.by(self.blocks[i], self.blocks[j])
}

type TxBlocks []*TxBlock

func (b *TxBlock) Hash() (h common.Hash) {
	return b.Header.Hash()
}

func (b *TxBlock) MHash() (h common.Hash) {
	m := []interface{}{b.Header.Hash().Bytes(), b.Body.Hash().Bytes()}
	return util.Hash(m)
}

func (b *TxBlock) NumberU64() uint64 {
	return b.Header.BlockNumber
}

func (b *TxBlock) ParentHash() common.Hash {
	return common.BytesToHash(b.Header.PreviousBlockHash)
}

func (b *TxBlock) Root() common.Hash {
	return common.BytesToHash(b.Header.StateRoot)
}

func (b *TxBlock) Transactions() Transactions {
	return b.Body.Transactions
}

func (b *TxBlock) GetVoteTickets() int {
	if b.GetHeader().GetBoolMap() == nil {
		return 0
	} else {
		couter := 0
		for i := 0; i < len(b.GetHeader().GetBoolMap()); i++ {
			if b.GetHeader().GetBoolMap()[i] {
				couter++
			}
		}
		return couter
	}
}

func (b *TxBlockHeader) Hash() (h common.Hash) {
	m := []interface{}{b.Version, b.PreviousBlockHash, b.BlockNumber, b.DSCoinBase, b.TxRoot,
		b.GasLimit, b.DSBlockNum, b.DSBlockHash, b.TxNum, b.Miner.Pubkey}

	for i := 0; i < len(b.DSCoinBase); i++ {
		m = append(m, b.DSCoinBase[i])
	}

	for i := 0; i < len(b.ShardingLeadCoinBase); i++ {
		m = append(m, b.ShardingLeadCoinBase[i])
	}
	return util.Hash(m)
}

func (b *TxBlockBody) Hash() (h common.Hash) {
	m := []interface{}{b.CurrentBlockHash, b.NumberOfMicroBlock}
	for i := 0; i < len(b.MicroBlockHashes); i++ {
		m = append(m, b.MicroBlockHashes[i])
	}
	for i := 0; i < len(b.Transactions); i++ {
		m = append(m, b.Transactions[i].Signature)
	}
	return util.Hash(m)
}

func (h *TxBlockHeader) ParentHash() common.Hash {
	return common.BytesToHash(h.PreviousBlockHash)
}

func (h *TxBlockHeader) Root() common.Hash {
	return common.BytesToHash(h.StateRoot)
}

func (b *TxBlockHeader) ToReadable() (s string, err error) {
	type readableHeader struct {
		TxBlockHeader
		HexParentHash  string   `json:"hexParentHash,omitempty"`
		HexCoinBase    []string `json:"hexCoinBase,omitempty"`
		HexStateRoot   string   `json:"hexStateRoot,omitempty"`
		HexTxRoot      string   `json:"hexTxRoot,omitempty"`
		HexDSBlockHash string   `json:"hexDSBlockHash,omitempty"`
	}

	coinbases := make([]string, 0, len(b.DSCoinBase))
	for i := 0; i < len(b.DSCoinBase); i++ {
		coinbases = append(coinbases, common.Bytes2Hex(b.DSCoinBase[i]))
	}

	rHeader := readableHeader{*b, common.Bytes2Hex(b.PreviousBlockHash), coinbases,
		common.Bytes2Hex(b.StateRoot), common.Bytes2Hex(b.TxRoot), common.Bytes2Hex(b.DSBlockHash)}

	headerInJson, err := json.Marshal(rHeader)
	return string(headerInJson), err
}

func (b *TxBlockBody) ToReadable() (s string, err error) {
	type readableBody struct {
		TxBlockBody
		HexMicroBlockHashs []string `json:"hexMicroBlockHashs,omitempty"`
	}

	mbHashes := make([]string, 0, len(b.MicroBlockHashes))
	for i := 0; i < len(b.MicroBlockHashes); i++ {
		mbHashes = append(mbHashes, common.Bytes2Hex(b.MicroBlockHashes[i]))
	}

	rBody := readableBody{*b, mbHashes}
	bodyInJson, err := json.Marshal(rBody)
	return string(bodyInJson), err
}

func (b *TxBlock) ToReadable() (s string, err error) {
	headerInJson, err := b.Header.ToReadable()
	bodyInJson, err := b.Body.ToReadable()
	s = fmt.Sprintf("{\"txheader\" : %s, \"txbody\" : %s}", string(headerInJson), string(bodyInJson))
	return s, nil
}

func (t *DSBlock) Hash() (h common.Hash) {
	return t.Header.Hash()
}

func (t *DSBlockHeader) Hash() (h common.Hash) {
	m := []interface{}{t.Version, t.Timestamp, t.PreviousBlockHash, t.WinnerPubKey, t.WinnerNonce, t.BlockNumber, t.PowDifficulty, t.NewLeader.Id.Name, t.Miner.Pubkey}
	return util.Hash(m)
}

func (t *DSBlock) MHash() (h common.Hash) {
	m := []interface{}{t.Header.Hash().Bytes(), t.Body.Hash().Bytes()}
	return util.Hash(m)
}

func (t *DSBlock) ToReadable() (s string, err error) {
	s = fmt.Sprintf("DSBlock: %+v", t)
	return s, err
}

func (t *DSBlockBody) Hash() (h common.Hash) {
	m := []interface{}{t.CurrentBlockHash}
	for i := 0; i < len(t.ShardingNodes); i++ {
		m = append(m, t.ShardingNodes[i].Pubkey)
	}
	return util.Hash(m)
}

func (b *DSBlock) GetVoteTickets() int {
	if b.GetHeader().GetBoolMap() == nil {
		return 0
	} else {
		couter := 0
		for i := 0; i < len(b.GetHeader().GetBoolMap()); i++ {
			if b.GetHeader().GetBoolMap()[i] {
				couter++
			}
		}
		return couter
	}
}

func (b *DSBlock) GetCoinbaseByPubkey(pubkey []byte) (r []byte, ok bool) {
	for _, p := range b.GetBody().GetCommittee() {
		equal := bytes.Compare(p.Pubkey, pubkey)
		if equal == 0 {
			return p.Coinbase, true
		}
	}

	for _, p := range b.GetBody().GetShardingNodes() {
		equal := bytes.Compare(p.Pubkey, pubkey)
		if equal == 0 {
			return p.Coinbase, true
		}
	}

	return nil, false
}

func BlsMultiPubkeyVerify(boolMap []bool, sortPeer []*PeerEndpoint, composedKey []byte) error {
	checkPubkey := &multibls.PubKey{}
	for i := 0; i < len(sortPeer); i++ {
		if boolMap[i] {
			pub := &multibls.PubKey{}
			err := pub.Deserialize(sortPeer[i].Pubkey)
			if err != nil {
				return err
			}
			checkPubkey.Add(&pub.PublicKey)
		}
	}
	composedPub := &multibls.PubKey{}
	err := composedPub.Deserialize(composedKey)
	if err != nil {
		return err
	}
	if checkPubkey.IsEqual(&composedPub.PublicKey) {
		return nil
	} else {
		return fmt.Errorf("Composed pubkey not equal, Calculated: %s, ToChecked: %s",
			checkPubkey.GetHexString(), composedPub.GetHexString())
	}
}

func (b *DSBlock) NumberU64() uint64 {
	return b.Header.BlockNumber
}
func (b *DSBlock) ParentHash() common.Hash {
	return common.BytesToHash(b.Header.PreviousBlockHash)
}

type tmpDSBody struct {
	ShardingNodes    []*PeerEndpoint
	Committee        []*PeerEndpoint
	CurrentBlockHash []byte
}

func (b *DSBlockBody) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, tmpDSBody{
		ShardingNodes:    b.ShardingNodes,
		Committee:        b.Committee,
		CurrentBlockHash: b.CurrentBlockHash,
	})
}

func (b *DSBlockBody) DecodeRLP(s *rlp.Stream) error {
	var eb tmpDSBody
	s.Kind()
	if err := s.Decode(&eb); err != nil {
		return err
	}
	b.ShardingNodes = eb.ShardingNodes
	b.Committee = eb.Committee
	b.CurrentBlockHash = eb.CurrentBlockHash
	return nil
}

type DsBlocks []*DSBlock
type DsBlockBy func(b1, b2 *DSBlock) bool

func (self DsBlockBy) Sort(blocks DsBlocks) {
	bs := dsblockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type dsblockSorter struct {
	blocks DsBlocks
	by     func(b1, b2 *DSBlock) bool
}

func (self dsblockSorter) Len() int {
	return len(self.blocks)
}

func (self dsblockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}

func (self dsblockSorter) Less(i, j int) bool {
	return self.by(self.blocks[i], self.blocks[j])
}

func DsNumber(b1, b2 *DSBlock) bool {
	return b1.Header.BlockNumber < b2.Header.BlockNumber
}

type TxsBlocks []*TxBlock
type TxBlockBy func(b1, b2 *TxBlock) bool

func (self TxBlockBy) Sort(blocks TxBlocks) {
	bs := txblockSorter{
		blocks: blocks,
		by:     self,
	}
	sort.Sort(bs)
}

type txblockSorter struct {
	blocks TxBlocks
	by     func(b1, b2 *TxBlock) bool
}

func (self txblockSorter) Len() int {
	return len(self.blocks)
}

func (self txblockSorter) Swap(i, j int) {
	self.blocks[i], self.blocks[j] = self.blocks[j], self.blocks[i]
}

func (self txblockSorter) Less(i, j int) bool {
	return self.by(self.blocks[i], self.blocks[j])
}

func TxNumber(b1, b2 *TxBlock) bool {
	return b1.Header.BlockNumber < b2.Header.BlockNumber
}

func (t *PoWSubmission) Hash() (h common.Hash) {
	m := []interface{}{t.Result.Nonce, t.Result.PowResult, t.Result.MixHash, t.Rand1, t.Rand2, t.BlockNum, t.Peer.Pubkey}
	return util.Hash(m)
}

func (t *MicroBlock) Hash() (h common.Hash) {
	m := []interface{}{t.Header.BlockNumber, t.Header.TxNum, t.Header.DSBlockNum, t.Header.DSBlockHash, t.Miner.Pubkey, t.ShardID, t.ShardingLeadCoinBase}
	for i := 0; i < len(t.Transactions); i++ {
		m = append(m, t.Transactions[i].Signature)
	}
	return util.Hash(m)
}

func (t *VCBlock) Hash() (h common.Hash) {
	m := []interface{}{t.Header.Stage, t.Header.ViewChangeDSNo, t.Header.ViewChangeTXNo, t.Header.NewLeader.Pubkey}
	return util.Hash(m)
}

//func (t *TxBlock) CheckCoinBase(committeeSort PeerEndpointList, dschain *blockchain.DSBlockChain) error {
//
//	tcoinbase := append([]string{}, common.Bytes2Hex(t.Header.DSCoinBase[0]))
//	for i := 0; i < len(t.Header.BoolMap); i++ {
//		if t.Header.BoolMap[i] {
//			cb, ret := dschain.GetPubkey2CoinbaseCache().GetCoinbaseByPubkey(committeeSort[i].Pubkey)
//			if !ret {
//				return errors.New("cannot get pubkey")
//			}
//			tcoinbase = append(tcoinbase, common.Bytes2Hex(cb))
//		}
//	}
//
//	mcoinbase := []string{}
//	for i := 0; i < len(t.Header.DSCoinBase); i++ {
//		mcoinbase = append(mcoinbase, common.Bytes2Hex(t.Header.DSCoinBase[i]))
//	}
//
//	sort.Strings(tcoinbase)
//	sort.Strings(mcoinbase)
//
//	if !reflect.DeepEqual(tcoinbase, mcoinbase) {
//		return errors.New("coinbases not match")
//	}
//
//	return nil
//}

type IPubkey2Coinbase interface {
	GetCoinbaseByPubkey(pubkey []byte) (r []byte, ok bool)
}

func (t *TxBlock) CheckCoinBase2(committeeSort PeerEndpointList, pubKeyToCoinBaseMap IPubkey2Coinbase) error {

	tcoinbase := append([]string{}, common.Bytes2Hex(t.Header.DSCoinBase[0]))
	for i := 0; i < len(t.Header.BoolMap); i++ {
		if t.Header.BoolMap[i] {
			pubkey := committeeSort[i].Pubkey
			cb, ok := pubKeyToCoinBaseMap.GetCoinbaseByPubkey(pubkey)
			if ok {
				tcoinbase = append(tcoinbase, common.Bytes2Hex(cb))
			} else {
				return fmt.Errorf("No %s is found in Cache", common.ToHex(pubkey))

			}
		}
	}

	mcoinbase := []string{}
	for i := 0; i < len(t.Header.DSCoinBase); i++ {
		mcoinbase = append(mcoinbase, common.Bytes2Hex(t.Header.DSCoinBase[i]))
	}

	sort.Strings(tcoinbase)
	sort.Strings(mcoinbase)

	if !reflect.DeepEqual(tcoinbase, mcoinbase) {
		return fmt.Errorf("coinbases not match, \ntcoinbase(%d): %+v, \nmcoinbase(%d): %+v",
			len(tcoinbase), tcoinbase, len(mcoinbase), mcoinbase)
	}

	return nil
}
