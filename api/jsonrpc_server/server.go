// Copyright The go-okchain Authors 2018,  All rights reserved.

package jsonrpc_server

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"math"
	"math/big"
	"time"

	ps "github.com/ok-chain/okchain/core/server"

	"sync"

	"github.com/ok-chain/okchain/accounts"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/core"
	"github.com/ok-chain/okchain/core/blockchain/txblockchain"
	"github.com/ok-chain/okchain/core/state"
	"github.com/ok-chain/okchain/core/types"
	"github.com/ok-chain/okchain/core/vm"
	"github.com/ok-chain/okchain/crypto"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/rpc"
	"github.com/spf13/viper"
)

const (
	defaultGasPrice = 1
)

var rpcLogger = logging.MustGetLogger("rpc")

type ApiBackend struct {
	peer *ps.PeerServer
}

func newCoreApiServer(peer *ps.PeerServer) *ApiBackend {
	return &ApiBackend{peer}
}

func APIs(p *ps.PeerServer) []rpc.API {
	locker := new(AddrLocker)
	apis := []rpc.API{}
	apis = append(apis, rpc.API{
		Namespace: "okchain",
		Version:   "1.0",
		Service:   newCoreApiServer(p),
		Public:    true,
	}, rpc.API{
		Namespace: "account",
		Version:   "1.0",
		Service:   newAccountAPIServer(p, locker),
		Public:    true,
	})
	return apis
}

func (s *ApiBackend) GetAccount(ctx context.Context, Address string) (*protos.Account, error) {
	state, err := s.peer.TxBlockChain().State()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	address := common.BytesToAddress(common.FromHex(Address))
	//if state.GetBalance(address).Uint64() == 0 {
	//	return nil, errors.New("No such account of this address")
	//}
	return &protos.Account{
		Nonce:       state.GetNonce(address),
		Balance:     state.GetBalance(address).Uint64(),
		CodeHash:    state.GetCode(address),
		StakeWeight: state.GetWeight(address).Uint64(),
	}, nil
}

func (s *ApiBackend) GetDsBlock(ctx context.Context, Number uint64) (*RPCDsBlock, error) {
	num := Number
	block := s.peer.DsBlockChain().GetBlockByNumber(num).(*protos.DSBlock)
	return newRPCDsBlock(block), nil
}

func formatPeerEndpoints(peers []*protos.PeerEndpoint) (res []*RPCPeerEndpoint) {
	for _, node := range peers {
		p := &RPCPeerEndpoint{
			Id:      node.Id,
			Address: node.Address,
			Port:    node.Port,
			Pubkey:  node.Pubkey,
			Roleid:  node.Roleid,
		}
		res = append(res, p)
	}
	return res
}

func newRPCDsBlock(block *protos.DSBlock) *RPCDsBlock {
	header := &RPCDsBlockHeader{
		Version:           block.Header.Version,
		Timestamp:         block.Header.Timestamp,
		PreviousBlockHash: common.BytesToHash(block.Header.PreviousBlockHash),
		ShardingSum:       block.Header.ShardingSum,
		WinnerPubKey:      block.Header.WinnerPubKey,
		WinnerNonce:       block.Header.WinnerNonce,
		BlockNumber:       block.Header.BlockNumber,
		PowDifficulty:     block.Header.PowDifficulty,
		//NewLeader:         block.Header.NewLeader,
		BoolMap:     block.Header.BoolMap,
		Signature:   block.Header.Signature,
		MultiPubKey: block.Header.MultiPubKey,
	}
	in := []*protos.PeerEndpoint{block.Header.NewLeader}
	header.NewLeader = formatPeerEndpoints(in)[0]

	in = []*protos.PeerEndpoint{block.Header.Miner}
	header.Miner = formatPeerEndpoints(in)[0]
	body := &RPCDsBlockBody{
		//ShardingNodes:    block.Body.ShardingNodes,
		//Committee:        block.Body.Committee,
		CurrentBlockHash: common.BytesToHash(block.Body.CurrentBlockHash),
	}
	body.ShardingNodes = formatPeerEndpoints(block.Body.ShardingNodes)
	body.Committee = formatPeerEndpoints(block.Body.Committee)

	return &RPCDsBlock{
		Header: header,
		Body:   body,
	}
}

type RPCPeerEndpoint struct {
	Id      *protos.PeerID `protobuf:"bytes,1,opt,name=id" json:"id,omitempty"`
	Address string         `protobuf:"bytes,2,opt,name=address" json:"address,omitempty"`
	Port    uint32         `protobuf:"varint,3,opt,name=port" json:"port,omitempty"`
	Pubkey  []byte         `protobuf:"bytes,4,opt,name=pubkey" json:"pubkey,omitempty"`
	Roleid  string         `protobuf:"bytes,5,opt,name=roleid" json:"roleid,omitempty"`
}

type RPCDsBlockHeader struct {
	Version           uint32            `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Timestamp         *protos.Timestamp `protobuf:"bytes,2,opt,name=timestamp" json:"timestamp,omitempty"`
	PreviousBlockHash common.Hash       `protobuf:"bytes,3,opt,name=previousBlockHash,proto3" json:"previousBlockHash,omitempty"`
	ShardingSum       uint32            `protobuf:"varint,4,opt,name=shardingSum" json:"shardingSum,omitempty"`
	WinnerPubKey      []byte            `protobuf:"bytes,5,opt,name=winnerPubKey,proto3" json:"winnerPubKey,omitempty"`
	WinnerNonce       uint64            `protobuf:"varint,6,opt,name=winnerNonce" json:"winnerNonce,omitempty"`
	BlockNumber       uint64            `protobuf:"varint,7,opt,name=blockNumber" json:"blockNumber,omitempty"`
	PowDifficulty     uint64            `protobuf:"varint,8,opt,name=powDifficulty" json:"powDifficulty,omitempty"`
	NewLeader         *RPCPeerEndpoint  `protobuf:"bytes,9,opt,name=newLeader" json:"newLeader,omitempty"`
	Miner             *RPCPeerEndpoint  `protobuf:"bytes,10,opt,name=miner" json:"miner,omitempty"`
	BoolMap           []bool            `protobuf:"varint,10,rep,packed,name=boolMap" json:"boolMap,omitempty"`
	Signature         []byte            `protobuf:"bytes,11,opt,name=signature,proto3" json:"signature,omitempty"`
	MultiPubKey       []byte            `protobuf:"bytes,12,opt,name=multiPubKey,proto3" json:"multiPubKey,omitempty"`
}

type RPCDsBlockBody struct {
	ShardingNodes    []*RPCPeerEndpoint `protobuf:"bytes,1,rep,name=shardingNodes" json:"shardingNodes,omitempty"`
	Committee        []*RPCPeerEndpoint `protobuf:"bytes,2,rep,name=committee" json:"committee,omitempty"`
	CurrentBlockHash common.Hash        `protobuf:"bytes,3,opt,name=currentBlockHash,proto3" json:"currentBlockHash,omitempty"`
}

type RPCDsBlock struct {
	Header *RPCDsBlockHeader
	Body   *RPCDsBlockBody
}

func (s *ApiBackend) GetTxBlock(ctx context.Context, Number uint64) (*RPCTxBlock, error) {
	num := Number
	block := s.peer.TxBlockChain().GetBlockByNumber(num).(*protos.TxBlock)
	return newRPCTxBlock(block), nil
}

//func newRPCDsBlock(block *protos.DSBlock) *

func newRPCTxBlock(block *protos.TxBlock) *RPCTxBlock {
	var coinbases []common.Address
	for _, coinbase := range block.Header.DSCoinBase {
		coinbases = append(coinbases, common.BytesToAddress(coinbase))
	}

	var scoinbases []common.Address
	for _, coinbase := range block.Header.ShardingLeadCoinBase {
		scoinbases = append(scoinbases, common.BytesToAddress(coinbase))
	}

	header := &RPCTxBlockHeader{
		Version:              block.Header.Version,
		Timestamp:            block.Header.Timestamp,
		PreviousBlockHash:    common.BytesToHash(block.Header.PreviousBlockHash),
		BlockNumber:          block.Header.BlockNumber,
		DSCoinbase:           coinbases,
		ShardingLeadCoinBase: scoinbases,
		StateRoot:            common.BytesToHash(block.Header.StateRoot),
		TxRoot:               common.BytesToHash(block.Header.TxRoot),
		GasLimit:             block.Header.GasLimit,
		GasUsed:              block.Header.GasUsed,
		DSBlockNum:           block.Header.DSBlockNum,
		DSBlockHash:          common.BytesToHash(block.Header.DSBlockHash),
		TxNum:                block.Header.TxNum,
		BoolMap:              block.Header.BoolMap,
		Signature:            block.Header.Signature,
		MultiPubKey:          block.Header.MultiPubKey,
	}

	in := []*protos.PeerEndpoint{block.Header.Miner}
	header.Miner = formatPeerEndpoints(in)[0]

	body := &RPCTxBlockBody{
		NumberOfMicroBlock: block.Body.NumberOfMicroBlock,
		CurrentBlockHash:   common.BytesToHash(block.Body.CurrentBlockHash),
	}
	var hashs []common.Hash
	for _, hash := range block.Body.MicroBlockHashes {
		hashs = append(hashs, common.BytesToHash(hash))
	}
	body.MicroBlockHashes = hashs
	var rpctrans []*RPCTransaction
	for i, trans := range block.Body.Transactions {
		rpctrans = append(rpctrans, newRPCTransaction(trans, block.Hash(), header.BlockNumber, uint64(i)))
	}
	body.Transactions = rpctrans
	return &RPCTxBlock{
		Header: header,
		Body:   body,
	}
}

//func newRPCTxBlock(block *protos.TxBlock) *RPCTxBlock {
//	rpctxblock := RPCTxBlock{
//		Version:              block.Header.Version,
//		Timestamp:            block.Header.Timestamp,
//		PreviousBlockHash:    common.BytesToHash(block.Header.PreviousBlockHash),
//		BlockNumber:          block.Header.BlockNumber,
//		Coinbase:             block.Header.Coinbase,
//		StateRoot:            common.BytesToHash(block.Header.StateRoot),
//		TxRoot:               common.BytesToHash(block.Header.TxRoot),
//		GasLimit:             block.Header.GasLimit,
//		GasUsed:              block.Header.GasUsed,
//		DSBlockHash:          common.BytesToHash(block.Header.DSBlockHash),
//		DSBlockNum:           block.Header.DSBlockNum,
//		TxNum:                block.Header.TxNum,
//		NumberOfMicroBlock:   block.Body.NumberOfMicroBlock,
//		//MicroBlockHashes:     block.Body.MicroBlockHashes,
//		ShardingLeaderPubKey: block.Body.ShardingLeaderPubKey,
//		CurrentBlockHash:     common.BytesToHash(block.Body.CurrentBlockHash),
//		Signature:            block.Header.Signature,
//	}
//	var hashs []common.Hash
//	for _,hash := range block.Body.MicroBlockHashes{
//		hashs=append(hashs,common.BytesToHash(hash))
//	}
//	rpctxblock.MicroBlockHashes=hashs
//	var rpctrans []*RPCTransaction
//	for _, trans := range block.Body.Transactions {
//		rpctrans = append(rpctrans, newRPCTransaction(trans, common.Hash{}, 0, 0))
//	}
//	rpctxblock.Transactions = rpctrans
//	return &rpctxblock
//}

type RPCTxBlockHeader struct {
	Version              uint32            `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Timestamp            *protos.Timestamp `protobuf:"bytes,2,opt,name=timestamp" json:"timestamp,omitempty"`
	PreviousBlockHash    common.Hash       `protobuf:"bytes,3,opt,name=previousBlockHash,proto3" json:"previousBlockHash,omitempty"`
	BlockNumber          uint64            `protobuf:"varint,4,opt,name=blockNumber" json:"blockNumber,omitempty"`
	DSCoinbase           []common.Address  `protobuf:"bytes,5,rep,name=dSCoinBase,proto3" json:"dSCoinBase,omitempty"`
	ShardingLeadCoinBase []common.Address  `protobuf:"bytes,6,rep,name=shardingLeadCoinBase,proto3" json:"shardingLeadCoinBase,omitempty"`
	StateRoot            common.Hash       `protobuf:"bytes,7,opt,name=stateRoot,proto3" json:"stateRoot,omitempty"`
	TxRoot               common.Hash       `protobuf:"bytes,8,opt,name=txRoot,proto3" json:"txRoot,omitempty"`
	GasLimit             uint64            `protobuf:"varint,9,opt,name=gasLimit" json:"gasLimit,omitempty"`
	GasUsed              uint64            `protobuf:"varint,10,opt,name=gasUsed" json:"gasUsed,omitempty"`
	DSBlockNum           uint64            `protobuf:"varint,11,opt,name=dSBlockNum" json:"dSBlockNum,omitempty"`
	DSBlockHash          common.Hash       `protobuf:"bytes,12,opt,name=dSBlockHash,proto3" json:"dSBlockHash,omitempty"`
	Miner                *RPCPeerEndpoint  `protobuf:"bytes,13,opt,name=miner" json:"miner,omitempty"`
	TxNum                uint64            `protobuf:"varint,14,opt,name=txNum" json:"txNum,omitempty"`
	BoolMap              []bool            `protobuf:"varint,15,rep,packed,name=boolMap" json:"boolMap,omitempty"`
	Signature            []byte            `protobuf:"bytes,16,opt,name=signature,proto3" json:"signature,omitempty"`
	MultiPubKey          []byte            `protobuf:"bytes,17,opt,name=multiPubKey,proto3" json:"multiPubKey,omitempty"`
}

type RPCTxBlockBody struct {
	NumberOfMicroBlock uint32            `protobuf:"varint,1,opt,name=numberOfMicroBlock" json:"numberOfMicroBlock,omitempty"`
	MicroBlockHashes   []common.Hash     `protobuf:"bytes,2,rep,name=microBlockHashes,proto3" json:"microBlockHashes,omitempty"`
	Transactions       []*RPCTransaction `protobuf:"bytes,3,rep,name=Transactions" json:"Transactions,omitempty"`
	CurrentBlockHash   common.Hash       `protobuf:"bytes,4,opt,name=currentBlockHash,proto3" json:"currentBlockHash,omitempty"`
}

type RPCTxBlock struct {
	Header *RPCTxBlockHeader
	Body   *RPCTxBlockBody
}

//type RPCTxBlock struct {
//	Version              uint32            `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
//	Timestamp            *protos.Timestamp `protobuf:"bytes,2,opt,name=timestamp" json:"timestamp,omitempty"`
//	PreviousBlockHash    common.Hash            `protobuf:"bytes,4,opt,name=previousBlockHash,proto3" json:"previousBlockHash,omitempty"`
//	BlockNumber          uint64            `protobuf:"varint,5,opt,name=blockNumber" json:"blockNumber,omitempty"`
//	Coinbase             []byte            `protobuf:"bytes,6,opt,name=coinbase,proto3" json:"coinbase,omitempty"`
//	StateRoot            common.Hash            `protobuf:"bytes,7,opt,name=stateRoot,proto3" json:"stateRoot,omitempty"`
//	TxRoot               common.Hash            `protobuf:"bytes,8,opt,name=txRoot,proto3" json:"txRoot,omitempty"`
//	GasLimit             uint64            `protobuf:"varint,10,opt,name=gasLimit" json:"gasLimit,omitempty"`
//	GasUsed              uint64            `protobuf:"varint,11,opt,name=gasUsed" json:"gasUsed,omitempty"`
//	DSBlockNum           uint64            `protobuf:"varint,12,opt,name=dSBlockNum" json:"dSBlockNum,omitempty"`
//	DSBlockHash          common.Hash            `protobuf:"bytes,13,opt,name=dSBlockHash,proto3" json:"dSBlockHash,omitempty"`
//	TxNum                uint64            `protobuf:"varint,14,opt,name=txNum" json:"txNum,omitempty"`
//	NumberOfMicroBlock   uint32            `protobuf:"varint,1,opt,name=numberOfMicroBlock" json:"numberOfMicroBlock,omitempty"`
//	MicroBlockHashes     []common.Hash          `protobuf:"bytes,2,rep,name=microBlockHashes,proto3" json:"microBlockHashes,omitempty"`
//	ShardingLeaderPubKey [][]byte          `protobuf:"bytes,3,rep,name=shardingLeaderPubKey,proto3" json:"shardingLeaderPubKey,omitempty"`
//	ConsensusMetadata    []byte            `protobuf:"bytes,4,opt,name=consensusMetadata,proto3" json:"consensusMetadata,omitempty"`
//	Transactions         []*RPCTransaction `protobuf:"bytes,5,rep,name=Transactions" json:"Transactions,omitempty"`
//	CurrentBlockHash     common.Hash            `protobuf:"bytes,6,opt,name=currentBlockHash,proto3" json:"currentBlockHash,omitempty"`
//	Signature            []byte            `protobuf:"bytes,7,opt,name=signature,proto3" json:"signature,omitempty"`
//}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        common.Hash       `json:"blockHash"`
	BlockNumber      uint64            `json:"blockNumber"`
	TransactionIndex uint64            `json:"transactionIndex"`
	From             common.Address    `json:"from"`
	Hash             common.Hash       `json:"hash"`
	Version          uint32            `protobuf:"varint,1,opt,name=version" json:"version,omitempty"`
	Nonce            uint64            `protobuf:"varint,2,opt,name=nonce" json:"nonce,omitempty"`
	SenderPubKey     []byte            `protobuf:"bytes,3,opt,name=senderPubKey,proto3" json:"senderPubKey,omitempty"`
	ToAddr           common.Address    `protobuf:"bytes,4,opt,name=toAddr,proto3" json:"toAddr,omitempty"`
	Amount           uint64            `protobuf:"varint,5,opt,name=amount" json:"amount,omitempty"`
	Signature        []byte            `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
	GasPrice         uint64            `protobuf:"varint,7,opt,name=gasPrice" json:"gasPrice,omitempty"`
	GasLimit         uint64            `protobuf:"varint,8,opt,name=gasLimit" json:"gasLimit,omitempty"`
	Code             []byte            `protobuf:"bytes,9,opt,name=code,proto3" json:"code,omitempty"`
	Data             []byte            `protobuf:"bytes,10,opt,name=data,proto3" json:"data,omitempty"`
	Timestamp        *protos.Timestamp `protobuf:"bytes,15,opt,name=timestamp" json:"timestamp,omitempty"`
}

// newRPCTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCTransaction(tx *protos.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCTransaction {
	//var signer types.Signer = types.FrontierSigner{}
	//if tx.Protected() {
	//	signer = types.NewEIP155Signer(tx.ChainId())
	//}
	//from, _ := types.Sender(signer, tx)
	//v, r, s := tx.RawSignatureValues()

	result := &RPCTransaction{
		From:         common.BytesToAddress(crypto.Keccak256(tx.SenderPubKey[1:])[12:]),
		Version:      tx.Version,
		Nonce:        tx.Nonce,
		SenderPubKey: tx.SenderPubKey,
		ToAddr:       common.BytesToAddress(tx.ToAddr),
		Amount:       tx.Amount,
		Signature:    tx.Signature,
		GasPrice:     tx.GasPrice,
		GasLimit:     tx.GasLimit,
		Code:         tx.Code,
		Data:         tx.Data,
		Timestamp:    tx.Timestamp,
		Hash:         common.BytesToHash(tx.Hash().Bytes()),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = common.BytesToHash(blockHash.Bytes())
		result.BlockNumber = blockNumber
		result.TransactionIndex = index
	}
	return result
}
func (s *ApiBackend) GetTransactionByHash(ctx context.Context, addr string) (*RPCTransaction, error) {
	// Try to return an already finalized transaction
	if tx, blockHash, number, idx := txblockchain.ReadTransaction(s.peer.GetTxChainDB(), common.BytesToHash(common.FromHex(addr))); tx != nil {
		return newRPCTransaction(tx, blockHash, number, idx), nil
	}
	// No finalized transaction, try to retrieve it from the pool
	if tx := s.peer.GetTxPool().Get(common.BytesToHash(common.FromHex(addr))); tx != nil {
		return newRPCTransaction(tx, common.Hash{}, 0, 0), nil
	}
	// Transaction unknown, return as such
	//logger.Debugf("no such transaction, hash=%s", addr)
	return nil, errors.New(fmt.Sprintf("can not get transaction by txid [%s]", addr))
}

func (s *ApiBackend) GetTransactionsFromTXPool(ctx context.Context, num uint64) ([]*RPCTransaction, error) {
	trans := s.peer.GetTxPool().PendingTxList()
	if trans == nil {
		trans = protos.Transactions{}
	}
	var res []*RPCTransaction
	if num > 100 {
		num = 100
	}
	trans_n := uint64(len(trans))
	if trans_n < num {
		trans2 := s.peer.GetTxPool().QueueTxList()
		m := num - trans_n
		m2 := uint64(len(trans2))
		if m2 < m {
			m = m2
		}
		trans = append(trans, trans2[:m]...)
	}
	n := uint64(len(trans))
	if n > num {
		n = num
	}
	for i := uint64(0); i < n; i++ {
		res = append(res, newRPCTransaction(trans[i], common.Hash{}, 0, 0))
	}
	return res, nil

}

//func (s *ApiBackend) GetRecentTransactions(ctx context.Context, Number uint64) ([]*protos.Transaction, error) {
//	block := s.peer.TxBlockChain().CurrentBlock()
//	if block == nil {
//		return nil, errors.New("get currentTxBlock failed")
//	}
//	trans_n := Number
//	block_number := int64(block.Header.BlockNumber)
//	block_hash := block.Hash()
//	var max_trans uint64 = 20
//	if trans_n > max_trans {
//		trans_n = max_trans
//	}
//	var i uint64 = 0
//	var transactions []*protos.Transaction
//	for ; block_number >= 0; block_number-- {
//		block = s.peer.TxBlockChain().GetBlock(block_hash, uint64(block_number))
//		if block == nil {
//			return nil, errors.New(fmt.Sprintf("can not get TxBlock by number [%d] and hash [%s]", block_number, block_hash.String()))
//		}
//		var len int = len(block.Body.Transactions)
//		for j := len - 1; j >= 0; j-- {
//			trans := block.Body.Transactions[j]
//			transactions = append(transactions, trans)
//			i++
//			if i == trans_n {
//				return transactions, nil
//			}
//		}
//		block_hash = block.Header.ParentHash()
//	}
//	return transactions, nil
//}
func (s *ApiBackend) GetLatestDsBlock(ctx context.Context) (*RPCDsBlock, error) {
	return newRPCDsBlock(s.peer.DsBlockChain().CurrentBlock().(*protos.DSBlock)), nil
}

//func (s *ApiBackend) GetLatestDsBlock(ctx context.Context) (map[string]map[string]interface{}, error) {
//	dsblock := map[string]map[string]interface{}{
//		"header": {
//			"version":           s.peer.DsBlockChain().CurrentBlock().Header.Version,
//			"timestamp":         s.peer.DsBlockChain().CurrentBlock().Header.Timestamp.String(),
//			"previousBlockHash": hex.EncodeToString(s.peer.DsBlockChain().CurrentBlock().Header.PreviousBlockHash),
//			"shardingSum":       s.peer.DsBlockChain().CurrentBlock().Header.ShardingSum,
//			"winnerPubKey":      hex.EncodeToString(s.peer.DsBlockChain().CurrentBlock().Header.WinnerPubKey),
//			"winnerNonce":       s.peer.DsBlockChain().CurrentBlock().Header.WinnerNonce,
//			"blockNumber":       s.peer.DsBlockChain().CurrentBlock().Header.BlockNumber,
//			"powDifficulty":     s.peer.DsBlockChain().CurrentBlock().Header.PowDifficulty,
//			"newLeader":         s.peer.DsBlockChain().CurrentBlock().Header.NewLeader.String(),
//		},
//		"body": {
//			"shardingNodes":    nil,
//			"currentBlockHash": hex.EncodeToString(s.peer.DsBlockChain().CurrentBlock().Body.CurrentBlockHash),
//			"signature":        s.peer.DsBlockChain().CurrentBlock().Body.Signature,
//		},
//	}
//	var shardingNodes []string
//	for _, node := range s.peer.DsBlockChain().CurrentBlock().Body.ShardingNodes {
//		shardingNodes = append(shardingNodes, node.String())
//	}
//	dsblock["body"]["shardingNodes"] = shardingNodes
//	return dsblock, nil
//}
func (s *ApiBackend) GetLatestTxBlock(ctx context.Context) (*RPCTxBlock, error) {
	return newRPCTxBlock(s.peer.TxBlockChain().CurrentBlock().(*protos.TxBlock)), nil
}

func (s *ApiBackend) GetLatestDsBlockMsg(ctx context.Context) (map[string]interface{}, error) {
	block := s.peer.DsBlockChain().CurrentBlock().(*protos.DSBlock)
	return map[string]interface{}{
		"blockNumber":      block.Header.BlockNumber,
		"currentBlockHash": common.BytesToHash(block.Body.CurrentBlockHash),
	}, nil
}

func (s *ApiBackend) GetLatestTxBlockMsg(ctx context.Context) (map[string]interface{}, error) {
	block := s.peer.TxBlockChain().CurrentBlock().(*protos.TxBlock)
	return map[string]interface{}{
		"blockNumber":      block.Header.BlockNumber,
		"currentBlockHash": common.BytesToHash(block.Body.CurrentBlockHash),
	}, nil
}

var baseaddr_nonce uint64 = 0
var baseaddr_nonce_lock sync.Mutex

func (s *ApiBackend) RegisterAccount(ctx context.Context, Address string) (bool, error) {
	//pub:="0x0404e6ed7dbcead92ca9860519affd046659f7b1fc6c3c294f7a27ae0e782fe17e1ee907351c4d68570fc40599fcf66dcfcd2ba52d9a037ba9324eab9b9ee0fd2c"
	//pri:="0x734eb4d507effa3fae1eda3e0875e19d1b999b2fd5821a602dc26ccc4f13ff78"
	state, err := s.peer.TxBlockChain().State()
	if err != nil {
		fmt.Println(err)
		return false, err
	}
	from := common.BytesToAddress(common.FromHex("0xe8c994f91c2a00e4e9d5d0449f1896ae3d2a371f"))
	to := common.BytesToAddress(common.FromHex(Address))
	var gas uint64 = 1000000
	var gasPrice uint64 = 1
	var nonce = state.GetNonce(from)
	baseaddr_nonce_lock.Lock()
	defer baseaddr_nonce_lock.Unlock()
	if baseaddr_nonce <= nonce {
		baseaddr_nonce = nonce
	} else {
		nonce = baseaddr_nonce
	}
	var value uint64 = 100000000
	trans := &SendTxArgs{
		From:     from,
		To:       &to,
		Gas:      &gas,
		GasPrice: &gasPrice,
		Nonce:    &nonce,
		Value:    &value,
	}
	account := accounts.Account{Address: from}
	wallet, err := s.peer.AccountManager().Find(account)
	fmt.Println("mywallets:", s.peer.AccountManager().Wallets())
	if err != nil {
		return false, err
	}
	if trans.Value == nil {
		trans.Value = new(uint64)
		*trans.Value = 0
	}
	if trans.Nonce == nil {
		trans.Nonce = new(uint64)
		*trans.Nonce = s.peer.GetTxPool().State().GetNonce(trans.From)
	}
	if trans.Gas == nil {
		err := errors.New("Invalid Tx: gas is nil")
		return false, err
	}
	if trans.GasPrice == nil {
		err := errors.New("Invalid Tx: gasPrice is nil")
		return false, err
	}
	tx := trans.ToTransaction()
	sign, err := wallet.SignTxWithPassphrase(account, viper.GetString("keystorepassword"), tx)
	if err != nil {
		return false, err
	}
	tx.Signature = sign
	fmt.Println(trans)
	err = s.peer.GetTxPool().AddLocal(tx)
	if err != nil {
		return false, errors.New("There are too many users signing up, please try again later")
	}
	rpcLogger.Info("registerAccount:", Address, "done")
	baseaddr_nonce++
	return true, nil
}

func (s *ApiBackend) GetTransactionsByAccount(ctx context.Context, Number uint64, Address string) ([]*RPCTransaction, error) {
	block := s.peer.TxBlockChain().CurrentBlock().(*protos.TxBlock)
	if block == nil {
		return nil, errors.New("get currentTxBlock failed")
	}
	trans_n := Number
	block_number := int64(block.Header.BlockNumber)
	block_hash := block.Hash()
	var max_trans uint64 = 20
	if trans_n > max_trans {
		trans_n = max_trans
	}
	var i uint64 = 0
	var transactions []*RPCTransaction
	signer := protos.P256Signer{}
	for ; block_number >= 0; block_number-- {
		tmpBlk := s.peer.TxBlockChain().GetBlock(block_hash, uint64(block_number))
		if tmpBlk == nil {
			return nil, errors.New(fmt.Sprintf("can not get TxBlock by number [%d] and hash [%s]", block_number, block_hash.String()))
		} else {
			block = tmpBlk.(*protos.TxBlock)
		}
		var len = len(block.Body.Transactions)
		for j := len - 1; j >= 0; j-- {
			trans := block.Body.Transactions[j]

			if bytes.Equal(signer.BytePubKeyToAddress(trans.SenderPubKey).Bytes(), common.FromHex(Address)) || bytes.Equal(trans.ToAddr, common.FromHex(Address)) {
				transactions = append(transactions, newRPCTransaction(trans, common.BytesToHash(block.Body.CurrentBlockHash), block.Header.BlockNumber, uint64(j)))
				i++
				if i == trans_n {
					return transactions, nil
				}
			}

		}
		block_hash = block.Header.ParentHash()
	}
	return transactions, nil
}

func (s *ApiBackend) SendTransaction(ctx context.Context, args SendTxArgs) (common.Hash, error) {
	// txpool.Add(r)

	//r := &protos.Transaction{
	//	SenderPubKey: args.From.Bytes(),
	//	ToAddr:       args.To.Bytes(),
	//	Amount:       args.Value.ToInt().Uint64(),
	//	GasLimit:     uint64(*args.Gas),
	//	GasPrice:     args.GasPrice.ToInt().Uint64(),
	//	Nonce:        uint64(*args.Nonce),
	//}
	rpcLogger.Debugf("1 SendTxArgs: %+v\n", args)
	if args.Value == nil {
		args.Value = new(uint64)
		*args.Value = 0
	}

	if args.Nonce == nil {
		args.Nonce = new(uint64)
		*args.Nonce = s.peer.GetTxPool().State().GetNonce(args.From)
	}

	if args.Gas == nil {
		err := errors.New("Invalid Tx: gas is nil")
		return common.Hash{}, err
	}

	if args.GasPrice == nil {
		err := errors.New("Invalid Tx: gasPrice is nil")
		return common.Hash{}, err
	}

	r := args.ToTransaction()
	//state := s.peer.GetTxPool().State()

	r.Timestamp = protos.CreateUtcTimestamp()
	//SenderPubKey should be a public key, XXX, use crypto pkg to turn it to address
	//r.Nonce = state.GetNonce(common.BytesToAddress(r.GetSenderPubKey()))
	//signature

	err := s.peer.GetTxPool().AddLocal(r)
	if err != nil {
		return common.Hash{}, err
	}

	return r.Hash(), nil
}

func (s *ApiBackend) Add(ctx context.Context, a, b int) (int, error) {
	rpcLogger.Infof("addfunc, a: %d, b: %d", a, b)
	return a + b, nil
}

func (s *ApiBackend) GetTransactionReceipt(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	tx, blockHash, blockNumber, index := txblockchain.ReadTransaction(s.peer.GetTxChainDB(), hash)
	if tx == nil {
		return nil, nil
	}
	receipts := s.peer.TxBlockChain().GetReceiptsByHash(blockHash)

	if len(receipts) <= int(index) {
		return nil, nil
	}
	receipt := receipts[index]

	fields := map[string]interface{}{
		"blockHash":         blockHash,
		"blockNumber":       hexutil.Uint64(blockNumber),
		"transactionHash":   hash,
		"transactionIndex":  hexutil.Uint64(index),
		"from":              common.BytesToAddress(tx.SenderPubKey),
		"to":                tx.To(),
		"gasUsed":           hexutil.Uint64(receipt.GasUsed),
		"cumulativeGasUsed": hexutil.Uint64(receipt.CumulativeGasUsed),
		"contractAddress":   nil,
		"logs":              receipt.Logs,
		"logsBloom":         receipt.Bloom,
	}

	// Assign receipt status or post state.
	if len(receipt.PostState) > 0 {
		fields["root"] = hexutil.Bytes(receipt.PostState)
	} else {
		fields["status"] = hexutil.Uint(receipt.Status)
	}
	if receipt.Logs == nil {
		fields["logs"] = [][]*types.Log{}
	}
	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if receipt.ContractAddress != (common.Address{}) {
		fields["contractAddress"] = receipt.ContractAddress
	}
	return fields, nil
}

func (s *ApiBackend) SendRawTransaction(ctx context.Context, r *protos.Transaction) (common.Hash, error) {
	err := s.peer.GetTxPool().AddLocal(r)
	if err != nil {
		return common.Hash{}, err
	}
	return r.Hash(), nil
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *uint64         `json:"gas"`
	GasPrice *uint64         `json:"gasPrice"`
	Value    *uint64         `json:"value"`
	Data     []byte          `json:"data"`
}

func (s *ApiBackend) StateAndHeaderByNumber() (*state.StateDB, *protos.TxBlockHeader, error) {
	// Otherwise resolve the block number and return its state
	header := s.peer.TxBlockChain().CurrentBlock().(*protos.TxBlock).GetHeader()
	if header == nil {
		return nil, nil, fmt.Errorf("Can not get current block header")
	}
	stateDb, err := s.peer.TxBlockChain().StateAt(header.Root())
	return stateDb, header, err
}

func (s *ApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *protos.TxBlockHeader, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), new(big.Int).SetUint64(math.MaxUint64))
	vmError := func() error { return nil }
	bc := s.peer.TxBlockChain()
	context := core.NewEVMContext(msg, header, bc, nil)
	return vm.NewEVM(context, state, bc.Config(), vmCfg), vmError, nil
}

func (s *ApiBackend) doCall(ctx context.Context, args CallArgs, vmCfg vm.Config, timeout time.Duration) ([]byte, uint64, bool, error) {
	defer func(start time.Time) { rpcLogger.Info("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := s.StateAndHeaderByNumber()
	if state == nil || err != nil {
		return nil, 0, false, err
	}
	// Set default gas & gas price if none were set
	gas, gasPrice := *args.Gas, *args.GasPrice
	if gas == 0 {
		gas = math.MaxUint64 / 2
	}
	if gasPrice == 0 {
		gasPrice = defaultGasPrice
	}

	// Create new call message
	msg := protos.NewMsg(args.From, args.To, 0, new(big.Int).SetUint64(*args.Value), gas, new(big.Int).SetUint64(gasPrice), args.Data, false)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	evm, vmError, err := s.GetEVM(ctx, msg, state, header, vmCfg)
	if err != nil {
		return nil, 0, false, err
	}
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	res, gas, failed, err := core.ApplyMessage(evm, msg, gp)
	if err := vmError(); err != nil {
		return nil, 0, false, err
	}
	return res, gas, failed, err
}

// Call executes the given transaction on the state for the given block number.
// It doesn't make and changes in the state/blockchain and is useful to execute and retrieve values.
func (s *ApiBackend) Call(ctx context.Context, args CallArgs) ([]byte, error) {
	result, _, _, err := s.doCall(ctx, args, vm.Config{}, 5*time.Second)
	return result, err
}
func (s *ApiBackend) GasPrice(ctx context.Context) (uint64, error) {
	return 1, nil

}

//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_add","params":[1, 100],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getLatestDsBlock","params":[],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getLatestTxBlock","params":[],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014

//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getAccount","params":["0xA7f3dFed2BCF7b35A8824e11Ae8f723650EDFb58"],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_sendTransaction","params":[{"from":"0x94dc66c8be8393e41791dfd7b8a47fe43f3e0890","to":"0xA7f3dFed2BCF7b35A8824e11Ae8f723650EDFb58","gas": 100000,"gasPrice": 1, "value": 1000000, "nonce":0}],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_registerAccount","params":["0xA7f3dFed2BCF7b35A8824e11Ae8f723650EDFb58"],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getTransaction","params":["0xA7f3dFed2BCF7b35A8824e11Ae8f723650EDFb58"],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getRecentTransactions","params":[20],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014

//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getDsBlock","params":[1],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getTxBlock","params":[1],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
//curl -X POST --data '{"jsonrpc":"2.0","method":"okchain_getTransactionReceipt","params":["0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"],"id":1}' -H "Content-Type: application/json" http://127.0.0.1:16014
