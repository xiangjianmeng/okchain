// Copyright The go-okchain Authors 2018,  All rights reserved.

package grpcserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"log"
	"net"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/core/blockchain/txblockchain"
	"github.com/ok-chain/okchain/core/server"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var rpcLogger = logging.MustGetLogger("grpcApiServer")

type grpcApiServer struct {
	peer *server.PeerServer
}

func NewGrpcApiServer(peer *server.PeerServer) *grpcApiServer {
	return &grpcApiServer{peer}
}

func (s *grpcApiServer) Notify(ctx context.Context, r *protos.Message) (*protos.ConfigRpcResponse, error) {
	return s.processMessage(r)
}

func (s *grpcApiServer) processMessage(r *protos.Message) (resp *protos.ConfigRpcResponse, err error) {

	defer func() {
		if recover() != nil {
			err = fmt.Errorf("Notification channel closed! ")
			resp = nil
		}
	}()

	s.peer.NotifyChannel <- r
	resp = &protos.ConfigRpcResponse{
		Code: 1,
	}
	err = nil

	return resp, err
}

func (s *grpcApiServer) GetNetworkInfo(ctx context.Context, r *protos.NetworkInfoRequest) (*protos.NetworkInfoResponse, error) {

	resp := &protos.NetworkInfoResponse{}

	resp.RemotePeerList = s.peer.NetworkInfo()

	return resp, nil
}

func (s *grpcApiServer) GetAccount(ctx context.Context, r *protos.AccountRequest) (*protos.AccountResponse, error) {
	state, err := s.peer.TxBlockChain().State()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	address := common.BytesToAddress(r.Address)
	return &protos.AccountResponse{
		Account: &protos.Account{
			Nonce:       state.GetNonce(address),
			Balance:     state.GetBalance(address).Uint64(),
			CodeHash:    state.GetCode(address),
			StakeWeight: state.GetWeight(address).Uint64(),
		},
	}, nil
}
func (s *grpcApiServer) GetDsBlock(ctx context.Context, r *protos.DsBlockRequest) (*protos.DSBlock, error) {
	num := r.Number
	if block := s.peer.DsBlockChain().GetBlockByNumber(num); block != nil {
		return block.(*protos.DSBlock), nil
	} else {
		return nil, errors.New("DSBlock Not found")
	}
}

func (s *grpcApiServer) GetTxBlock(ctx context.Context, in *protos.TxBlockRequest) (*protos.TxBlock, error) {
	num := in.Number
	if block := s.peer.TxBlockChain().GetBlockByNumber(num); block != nil {
		return block.(*protos.TxBlock), nil
	} else {
		return nil, errors.New("TxBlock Not found")
	}
}

func (s *grpcApiServer) GetTransaction(ctx context.Context, in *protos.TransactionRequest) (*protos.Transaction, error) {
	// Try to return an already finalized transaction
	fmt.Println("GetTransaction")
	if tx, _, _, _ := txblockchain.ReadTransaction(s.peer.GetTxChainDB(), common.BytesToHash(in.Addr)); tx != nil {
		return tx, nil
	}
	// No finalized transaction, try to retrieve it from the pool
	//if tx := s.b.GetPoolTransaction(hash); tx != nil {
	//	return newRPCPendingTransaction(tx)
	//}
	// Transaction unknown, return as such
	return nil, errors.New(fmt.Sprintf("can not get transaction by txid [%s]", common.BytesToHash(in.Addr).String()))
}

func (s *grpcApiServer) GetRecentTransactions(ctx context.Context, in *protos.RecentTransactionsRequest) (*protos.RecentTransactionsResponse, error) {
	blk := s.peer.TxBlockChain().CurrentBlock()
	if blk == nil {
		return nil, errors.New("get currentTxBlock failed")
	}
	block := blk.(*protos.TxBlock)
	trans_n := in.Number
	block_number := int64(block.Header.BlockNumber)
	block_hash := block.Hash()
	var max_trans uint64 = 20
	if trans_n > max_trans {
		trans_n = max_trans
	}
	var i uint64 = 0
	var transactions []*protos.Transaction
	for ; block_number >= 0; block_number-- {
		tmpBlk := s.peer.TxBlockChain().GetBlock(block_hash, uint64(block_number))
		if tmpBlk == nil {
			return nil, errors.New(fmt.Sprintf("can not get TxBlock by number [%d] and hash [%s]", block_number, block_hash.String()))
		}
		block := tmpBlk.(*protos.TxBlock)
		var len int = len(block.Body.Transactions)
		for j := len - 1; j >= 0; j-- {
			trans := block.Body.Transactions[j]
			transactions = append(transactions, trans)
			i++
			if i == trans_n {
				return &protos.RecentTransactionsResponse{Transactions: transactions}, nil
			}
		}
		block_hash = block.Header.ParentHash()
	}
	return &protos.RecentTransactionsResponse{Transactions: transactions}, nil
}
func (s *grpcApiServer) GetLatestDsBlock(ctx context.Context, in *protos.EmptyRequest) (*protos.DSBlock, error) {
	return s.peer.DsBlockChain().CurrentBlock().(*protos.DSBlock), nil
}
func (s *grpcApiServer) GetLatestTxBlock(ctx context.Context, in *protos.EmptyRequest) (*protos.TxBlock, error) {
	return s.peer.TxBlockChain().CurrentBlock().(*protos.TxBlock), nil
}

//var baseaddr uint64 = 0
func (s *grpcApiServer) RegisterAccount(ctx context.Context, in *protos.RegisterAccountRequest) (*protos.RegisterAccountResponse, error) {
	//pub:="0x0404e6ed7dbcead92ca9860519affd046659f7b1fc6c3c294f7a27ae0e782fe17e1ee907351c4d68570fc40599fcf66dcfcd2ba52d9a037ba9324eab9b9ee0fd2c"
	//pri:="0x734eb4d507effa3fae1eda3e0875e19d1b999b2fd5821a602dc26ccc4f13ff78"
	addr := "0x94dc66c8be8393e41791dfd7b8a47fe43f3e0890"
	trans := &protos.Transaction{
		SenderPubKey: common.FromHex(addr),
		ToAddr:       in.Address,
		Amount:       1000000,
		Timestamp:    protos.CreateUtcTimestamp(),
	}
	//trans:=&protos.Transaction{
	//	Version:      0,
	//	SenderPubKey: common.FromHex(addr),
	//	ToAddr:       in.Address,
	//	Amount:       1000000,
	//	Nonce:        baseaddr,
	//	GasPrice:     1,
	//	GasLimit:     1000,
	//	Code:         []byte(""),
	//	Data:         []byte(""),
	//	Timestamp:	  protos.CreateUtcTimestamp(),
	//}
	//baseaddr++
	//sign,err:=secp256k1.Sign(trans.Hash().Bytes(),common.FromHex(pri))
	//if err!=nil{
	//	return &protos.RegisterAccountResponse{Code:1},errors.New(fmt.Sprintf("sign error:%s",err.Error()))
	//}
	//trans.Signature=sign
	fmt.Println("send done:")
	s.peer.GetTxPool().AddLocal(trans)
	return &protos.RegisterAccountResponse{Code: 0}, nil
}

func (s *grpcApiServer) GetTransactionsByAccount(ctx context.Context, in *protos.GetTransactionsByAccountRequest) (*protos.GetTransactionsByAccountResponse, error) {
	block := s.peer.TxBlockChain().CurrentBlock()
	if block == nil {
		return nil, errors.New("get currentTxBlock failed")
	}
	trans_n := in.Number
	block_number := int64(block.NumberU64())
	block_hash := block.Hash()
	var max_trans uint64 = 20
	if trans_n > max_trans {
		trans_n = max_trans
	}
	var i uint64 = 0
	var transactions []*protos.Transaction
	for ; block_number >= 0; block_number-- {
		tmpBlk := s.peer.TxBlockChain().GetBlock(block_hash, uint64(block_number))
		if tmpBlk == nil {
			return nil, errors.New(fmt.Sprintf("can not get TxBlock by number [%d] and hash [%s]", block_number, block_hash.String()))
		}
		block := tmpBlk.(*protos.TxBlock)
		var len int = len(block.Body.Transactions)
		for j := len - 1; j >= 0; j-- {
			trans := block.Body.Transactions[j]
			if bytes.Equal(trans.SenderPubKey, in.Address) || bytes.Equal(trans.ToAddr, in.Address) {
				transactions = append(transactions, trans)
				i++
				if i == trans_n {
					return &protos.GetTransactionsByAccountResponse{Transactions: transactions}, nil
				}
			}

		}
		block_hash = block.Header.ParentHash()
	}
	return &protos.GetTransactionsByAccountResponse{Transactions: transactions}, nil
}

func (s *grpcApiServer) SendTransaction(ctx context.Context, r *protos.Transaction) (*protos.TransactionResponse, error) {
	// txpool.Add(r)
	//state := s.peer.GetTxPool().State()

	r.Timestamp = protos.CreateUtcTimestamp()
	//SenderPubKey should be a public key, XXX, use crypto pkg to turn it to address
	//r.Nonce = state.GetNonce(common.BytesToAddress(r.GetSenderPubKey()))
	//signature

	err := s.peer.GetTxPool().AddLocal(r)
	if err != nil {
		return &protos.TransactionResponse{
			Hash: r.Hash().Hex(),
		}, err
	}
	return &protos.TransactionResponse{
		Hash: r.Hash().Hex(),
	}, nil
}

func Start(address string) { //should start by this
	if address == "" {
		address = "0.0.0.0:1234"
	}
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	//protos.RegisterTransactionSenderServer(s, &transactionSender{})
	protos.RegisterBackendServer(s, &grpcApiServer{})
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	log.Println("Miner started...")
}
