// Copyright The go-okchain Authors 2018,  All rights reserved.

package blockchain

import (
	"fmt"
	"sort"

	"github.com/hashicorp/golang-lru"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/core/blockchain/dsblockchain"
	"github.com/ok-chain/okchain/core/consensus"
	"github.com/ok-chain/okchain/core/database"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
)

type Pubkey2Coinbase struct {
	currentBlock *protos.DSBlock
	cache        *lru.Cache
	notifyCh     chan *protos.DSBlock
	stopCh       chan interface{}
}

func NewPubkey2Coinbase() *Pubkey2Coinbase {
	pc := Pubkey2Coinbase{}
	pc.notifyCh = make(chan *protos.DSBlock, 1)
	pc.stopCh = make(chan interface{}, 1)
	pc.cache, _ = lru.New(20480)

	go pc.update()

	return &pc
}

func (pc *Pubkey2Coinbase) loadPk2Coinbase(l protos.PeerEndpointList) {
	for i := 0; i < l.Len(); i++ {
		p := l[i]
		spk := hexutil.Encode(p.Pubkey)
		scb := hexutil.Encode(p.Coinbase)
		dsLogger.Debugf("hex(pk): %s, hex(cb): %s", spk, scb)
		pc.cache.Add(spk, scb)
	}
}

func (pc *Pubkey2Coinbase) Reset(b *protos.DSBlock) (err error) {

	dsLogger.Debugf("Reseting pk2cb Cache with DS-%d", b.NumberU64())

	pc.currentBlock = b
	pc.loadPk2Coinbase(b.GetBody().GetCommittee())
	pc.loadPk2Coinbase(b.GetBody().GetShardingNodes())
	return nil
}

func (pc *Pubkey2Coinbase) GetCoinbaseByPubkey(pubkey []byte) (r []byte, ok bool) {
	spk := common.ToHex(pubkey)
	scb, ok := pc.cache.Get(spk)

	if scb != nil && ok {
		r = hexutil.MustDecode(scb.(string))
		dsLogger.Debugf("ok: %+v, hex(pk): %s, hex(cb): %s, r: %s", ok, spk, scb, common.Bytes2Hex(r))
		return r, true
	} else {
		return nil, false
	}
}

func (pc *Pubkey2Coinbase) GetCoinbaseByDsNum(dsNum uint64, pubkey []byte) (r []byte, ok bool) {
	if pc.currentBlock == nil || dsNum != pc.currentBlock.NumberU64() {
		return nil, false
	}

	return pc.GetCoinbaseByPubkey(pubkey)
}

func (pc *Pubkey2Coinbase) update() {

	defer func() {
		r := recover()
		if r != nil {
			dsLogger.Errorf("Pubkey2Coinbase Caching updating terminated bcoz of %+v", r)
		} else {
			dsLogger.Info("Pubkey2Coinbase Caching updating terminated normally")
		}
	}()

	defer close(pc.notifyCh)
	defer close(pc.stopCh)

	for {
		select {
		case o := <-pc.notifyCh:
			dsBlk := o
			if err := pc.Reset(dsBlk); err != nil {
				dsLogger.Error(err)
			} else {
				dsLogger.Infof("pk2cb Caching updated to DS-%d, CacheLenght: %d", dsBlk.NumberU64(), pc.cache.Len())
			}

		case <-pc.stopCh:
			break
		}
	}
}

type DSBlockChain struct {
	*BlockChain
	blsSigner *protos.BLSSigner
	pk2cb     *Pubkey2Coinbase
}

var dsLogger = logging.MustGetLogger("DSBlockChain")

func NewDSBlockChain(db database.Database, engine consensus.Engine, blsSigner *protos.BLSSigner) (*DSBlockChain, error) {

	baseBC, err := NewBlockChain(db, engine)
	if err != nil {
		return nil, err
	}

	pk2cb := NewPubkey2Coinbase()

	bc := DSBlockChain{
		baseBC,
		blsSigner,
		pk2cb,
	}

	bc.chainFeed.Subscribe(bc.pk2cb.notifyCh)

	bc.impl = &bc

	if err = bc.Start(); err != nil {
		dsLogger.Errorf("Fail to Start a DSBlockChain, Error: %s", err.Error())
		return nil, err
	}

	return &bc, nil
}

func (bc *DSBlockChain) Start() error {
	if err := bc.BlockChain.Start(); err != nil {
		return err
	}

	bc.PostChainEvents([]interface{}{bc.CurrentBlock()})
	return nil
}

func (bc *DSBlockChain) InsertBlock(block protos.IBlock) (err error) {

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	if err = bc.BlockChain.InsertBlock(block); err != nil {
		dsLogger.Errorf("Fail to InsertBlock: %+v, error: %s", block, err.Error())
		return err
	}

	bc.PostChainEvents([]interface{}{block})
	return nil
}

func (bc *DSBlockChain) GetBlsSigner() *protos.BLSSigner {
	return bc.blsSigner
}

func (bc *DSBlockChain) getDSBlock(blk protos.IBlock) *protos.DSBlock {

	var dsBlock *protos.DSBlock = nil
	var err error

	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("Convert IBlock to DSBlock panic, %+v", e)
			dsLogger.Error(err.Error())
		} else {
			err = nil
		}
	}()

	dsBlock = blk.(*protos.DSBlock)
	return dsBlock
}

func (bc *DSBlockChain) validateBlock(blk *protos.DSBlock) error {

	// 0. Check timestamp
	newHeader := blk.GetHeader()
	if (newHeader.Timestamp == nil) ||
		(newHeader.Timestamp.GetNanosecond() == 0 || newHeader.Timestamp.GetSecond() == 0) {
		return ErrTimestamp
	}

	// 1. Check parents & block number
	pHash := blk.ParentHash()
	pBlock := bc.GetBlock(pHash, blk.NumberU64()-1)
	if pBlock == nil {
		return ErrUnknownAncestor
	}

	// 2. Check if inserted.
	if bc.CurrentBlock().NumberU64() >= blk.NumberU64() {
		return ErrKnownBlock
	}

	// 3. check multi pubkey & signature.
	if blk.NumberU64() > 1 && bc.GetBlsSigner() != nil {
		baseBlk := bc.getDSBlock(pBlock)
		if baseBlk == nil {
			return ErrConvertionFail
		}

		dsLogger.Debugf("ParentDSBlock: %+v, Before Sort Committee: %+v, BoolMap: %+v, BaseBlockNO: %d", baseBlk, baseBlk.Body.GetCommittee(), baseBlk.GetHeader().GetBoolMap(), baseBlk.NumberU64())
		composedPubkeys := blk.Header.GetMultiPubKey()
		sortedCommittee := []*protos.PeerEndpoint{}
		for i := 0; i < len(baseBlk.Body.GetCommittee()); i++ {
			sortedCommittee = append(sortedCommittee, baseBlk.Body.GetCommittee()[i])
		}
		sort.Sort(protos.ByPubkey{PeerEndpointList: sortedCommittee})
		err := protos.BlsMultiPubkeyVerify(blk.Header.GetBoolMap(), sortedCommittee, composedPubkeys)
		if err != nil {
			dsLogger.Errorf("ERROR: %s, After Sort Committee: %+v", err.Error(), sortedCommittee)
			return ErrMismatchMultiPubkey
		}

		ret, err := bc.blsSigner.VerifyHash(blk.MHash().Bytes(), blk.Header.GetSignature(), composedPubkeys)
		if err != nil || !ret {
			return ErrMismatchMultiSignature
		}

		// Verify Bitmap in Header
		if blk.GetVoteTickets() < GetToleranceSize(baseBlk.GetBody().GetCommittee()) {
			return ErrBitmapTolerence
		}

	}

	return nil
}

func (bc *DSBlockChain) WriteBlock(blk protos.IBlock, db database.Putter) (err error) {
	defer func() error {
		if e := recover(); e != nil {
			return fmt.Errorf("WriteBlock panic, %+v", e)
		} else {
			return err
		}
	}()

	err = dsblockchain.WriteDsBlock(db, blk.(*protos.DSBlock))
	return err
}

func (bc *DSBlockChain) ReadBlock(hash common.Hash, number uint64) protos.IBlock {
	if blk := dsblockchain.GetDsBlock(bc.GetDatabase(), hash, number); blk != nil {
		return blk
	}
	return nil
}

func (bc *DSBlockChain) ValidateBlock(blk protos.IBlock) error {
	return bc.validateBlock(blk.(*protos.DSBlock))
}

func (bc *DSBlockChain) IsFutureBlockErr(err error) (bool, error) {
	return err == ErrUnknownAncestor, err
}

// TODO: Should not expose PostChainEvents. The chain events should be posted in WriteBlock.
func (bc *DSBlockChain) PostChainEvents(events []interface{}) {
	for _, event := range events {
		switch ev := event.(type) {
		case *protos.DSBlock:
			bc.chainFeed.Send(ev)
		}
	}
}

func (bc *DSBlockChain) GetPubkey2CoinbaseCache() *Pubkey2Coinbase {
	return bc.pk2cb
}
