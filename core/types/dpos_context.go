package types

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/trie"
	"github.com/ok-chain/okchain/crypto/sha3"
	"github.com/ok-chain/okchain/log"
)

type DposContext struct {
	epochTrie     *trie.Trie //记录每个周期的验证人列表
	delegateTrie  *trie.Trie //记录验证人以及对应投票人的列表
	voteTrie      *trie.Trie //记录投票人对应验证人
	candidateTrie *trie.Trie //记录候选人列表
	mintCntTrie   *trie.Trie //记录验证人在周期内的出块数目

	db database.Database
}

var (
	epochPrefix     = []byte("epoch-")
	delegatePrefix  = []byte("delegate-")
	votePrefix      = []byte("vote-")
	candidatePrefix = []byte("candidate-")
	mintCntPrefix   = []byte("mintCnt-")
)

func NewEpochTrie(root common.Hash, db database.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, epochPrefix, db)
}

func NewDelegateTrie(root common.Hash, db database.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, delegatePrefix, db)
}

func NewVoteTrie(root common.Hash, db database.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, votePrefix, db)
}

func NewCandidateTrie(root common.Hash, db database.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, candidatePrefix, db)
}

func NewMintCntTrie(root common.Hash, db database.Database) (*trie.Trie, error) {
	return trie.NewTrieWithPrefix(root, mintCntPrefix, db)
}

func NewDposContext(db database.Database) (*DposContext, error) {
	epochTrie, err := NewEpochTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	delegateTrie, err := NewDelegateTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	voteTrie, err := NewVoteTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(common.Hash{}, db)
	if err != nil {
		return nil, err
	}
	return &DposContext{
		epochTrie:     epochTrie,
		delegateTrie:  delegateTrie,
		voteTrie:      voteTrie,
		candidateTrie: candidateTrie,
		mintCntTrie:   mintCntTrie,
		db:            db,
	}, nil
}

func NewDposContextFromProto(db database.Database, ctxProto *DposContextProto) (*DposContext, error) {
	epochTrie, err := NewEpochTrie(ctxProto.EpochHash, db)
	if err != nil {
		return nil, err
	}
	delegateTrie, err := NewDelegateTrie(ctxProto.DelegateHash, db)
	if err != nil {
		return nil, err
	}
	voteTrie, err := NewVoteTrie(ctxProto.VoteHash, db)
	if err != nil {
		return nil, err
	}
	candidateTrie, err := NewCandidateTrie(ctxProto.CandidateHash, db)
	if err != nil {
		return nil, err
	}
	mintCntTrie, err := NewMintCntTrie(ctxProto.MintCntHash, db)
	if err != nil {
		return nil, err
	}
	return &DposContext{
		epochTrie:     epochTrie,
		delegateTrie:  delegateTrie,
		voteTrie:      voteTrie,
		candidateTrie: candidateTrie,
		mintCntTrie:   mintCntTrie,
		db:            db,
	}, nil
}

func (d *DposContext) Copy() *DposContext {
	epochTrie := *d.epochTrie
	delegateTrie := *d.delegateTrie
	voteTrie := *d.voteTrie
	candidateTrie := *d.candidateTrie
	mintCntTrie := *d.mintCntTrie
	return &DposContext{
		epochTrie:     &epochTrie,
		delegateTrie:  &delegateTrie,
		voteTrie:      &voteTrie,
		candidateTrie: &candidateTrie,
		mintCntTrie:   &mintCntTrie,
	}
}

func (d *DposContext) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, d.epochTrie.Hash())
	rlp.Encode(hw, d.delegateTrie.Hash())
	rlp.Encode(hw, d.candidateTrie.Hash())
	rlp.Encode(hw, d.voteTrie.Hash())
	rlp.Encode(hw, d.mintCntTrie.Hash())
	hw.Sum(h[:0])
	return h
}

func (d *DposContext) Snapshot() *DposContext {
	return d.Copy()
}

func (d *DposContext) RevertToSnapShot(snapshot *DposContext) {
	d.epochTrie = snapshot.epochTrie
	d.delegateTrie = snapshot.delegateTrie
	d.candidateTrie = snapshot.candidateTrie
	d.voteTrie = snapshot.voteTrie
	d.mintCntTrie = snapshot.mintCntTrie
}

func (d *DposContext) FromProto(dcp *DposContextProto) error {
	var err error
	d.epochTrie, err = NewEpochTrie(dcp.EpochHash, d.db)
	if err != nil {
		return err
	}
	d.delegateTrie, err = NewDelegateTrie(dcp.DelegateHash, d.db)
	if err != nil {
		return err
	}
	d.candidateTrie, err = NewCandidateTrie(dcp.CandidateHash, d.db)
	if err != nil {
		return err
	}
	d.voteTrie, err = NewVoteTrie(dcp.VoteHash, d.db)
	if err != nil {
		return err
	}
	d.mintCntTrie, err = NewMintCntTrie(dcp.MintCntHash, d.db)
	return err
}

type DposContextProto struct {
	EpochHash     common.Hash `json:"epochRoot"        gencodec:"required"`
	DelegateHash  common.Hash `json:"delegateRoot"     gencodec:"required"`
	CandidateHash common.Hash `json:"candidateRoot"    gencodec:"required"`
	VoteHash      common.Hash `json:"voteRoot"         gencodec:"required"`
	MintCntHash   common.Hash `json:"mintCntRoot"      gencodec:"required"`
}

func (d *DposContext) ToProto() *DposContextProto {
	return &DposContextProto{
		EpochHash:     d.epochTrie.Hash(),
		DelegateHash:  d.delegateTrie.Hash(),
		CandidateHash: d.candidateTrie.Hash(),
		VoteHash:      d.voteTrie.Hash(),
		MintCntHash:   d.mintCntTrie.Hash(),
	}
}

func (p *DposContextProto) Root() (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, p.EpochHash)
	rlp.Encode(hw, p.DelegateHash)
	rlp.Encode(hw, p.CandidateHash)
	rlp.Encode(hw, p.VoteHash)
	rlp.Encode(hw, p.MintCntHash)
	hw.Sum(h[:0])
	return h
}

func (d *DposContext) KickoutCandidate(candidateAddr common.Address) error {
	fmt.Println("[DEBUG]DPOS选举调试 KickoutCandidate")
	candidate := candidateAddr.Bytes()

	//TODO
	candidateInTrie, _ := d.candidateTrie.TryGet(candidate)
	fmt.Println("[DEBUG]DPOS选举调试 candidateInTrie after:", candidateInTrie)

	err := d.candidateTrie.TryDelete(candidate)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}
	root, _ := d.CandidateTrie().Commit(nil)
	d.CandidateTrie().TrieDB().Commit(root, true)

	//TODO
	candidateInTrie, _ = d.candidateTrie.TryGet(candidate)
	fmt.Println("[DEBUG]DPOS选举调试 candidateInTrie brfor:", candidateInTrie)

	iter := trie.NewIterator(d.delegateTrie.PrefixIterator(candidate))
	for iter.Next() {
		delegator := iter.Value
		key := append(candidate, delegator...)
		err = d.delegateTrie.TryDelete(key)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		root, _ := d.DelegateTrie().Commit(nil)
		d.DelegateTrie().TrieDB().Commit(root, true)

		v, err := d.voteTrie.TryGet(delegator)
		if err != nil {
			if _, ok := err.(*trie.MissingNodeError); !ok {
				return err
			}
		}
		if err == nil && bytes.Equal(v, candidate) {
			err = d.voteTrie.TryDelete(delegator)
			if err != nil {
				if _, ok := err.(*trie.MissingNodeError); !ok {
					return err
				}
			}
			root, _ := d.VoteTrie().Commit(nil)
			d.VoteTrie().TrieDB().Commit(root, true)
		}
	}
	return nil
}

func (d *DposContext) BecomeCandidate(candidateAddr common.Address) error {
	//TODO
	fmt.Println("[DEBUG]DPOS选举调试 BecomeCandidate")
	candidateInTrie, _ := d.candidateTrie.TryGet(candidateAddr.Bytes())
	fmt.Println("[DEBUG]DPOS选举调试 candidateInTrie after:", candidateInTrie)

	candidate := candidateAddr.Bytes()
	err := d.candidateTrie.TryUpdate(candidate, candidate)
	if err != nil {
		return err
	}
	root, err := d.CandidateTrie().Commit(nil)
	d.CandidateTrie().TrieDB().Commit(root, true)
	//TODO
	candidateInTrie, _ = d.candidateTrie.TryGet(candidateAddr.Bytes())
	fmt.Println("[DEBUG]DPOS选举调试 candidateInTrie befor:", candidateInTrie)
	return err
}

func (d *DposContext) Delegate(delegatorAddr, candidateAddr common.Address) error {
	//TODO
	fmt.Println("[DEBUG]DPOS选举调试 Delegate")

	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()

	// the candidate must be candidate
	candidateInTrie, err := d.candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to delegate")
	}

	// delete old candidate if exists
	oldCandidate, err := d.voteTrie.TryGet(delegator)
	if err != nil {
		if _, ok := err.(*trie.MissingNodeError); !ok {
			return err
		}
	}

	//TODO
	oldDelegateTrie, err := d.delegateTrie.TryGet(append(oldCandidate, delegator...))
	fmt.Println("[DEBUG]DPOS选举调试 delegateTrie after:", oldDelegateTrie)
	fmt.Println("[DEBUG]DPOS选举调试 voteTrie after:", oldCandidate)

	if oldCandidate != nil {
		d.delegateTrie.Delete(append(oldCandidate, delegator...))
	}
	if err = d.delegateTrie.TryUpdate(append(candidate, delegator...), delegator); err != nil {
		return err
	}
	root, err := d.DelegateTrie().Commit(nil)
	d.DelegateTrie().TrieDB().Commit(root, true)

	if err = d.voteTrie.TryUpdate(delegator, candidate); err != nil {
		return err
	}

	root, err = d.VoteTrie().Commit(nil)
	d.VoteTrie().TrieDB().Commit(root, true)

	//TODO
	newDelegateTrie, err := d.delegateTrie.TryGet(append(candidate, delegator...))
	fmt.Println("[DEBUG]DPOS选举调试 delegateTrie befor:", newDelegateTrie)

	newVoteTrie, _ := d.voteTrie.TryGet(delegator)
	fmt.Println("[DEBUG]DPOS选举调试 voteTrie befor:", newVoteTrie)

	return nil
}

func (d *DposContext) UnDelegate(delegatorAddr, candidateAddr common.Address) error {
	//TODO
	fmt.Println("[DEBUG]DPOS选举调试 UnDelegate")

	delegator, candidate := delegatorAddr.Bytes(), candidateAddr.Bytes()

	// the candidate must be candidate
	candidateInTrie, err := d.candidateTrie.TryGet(candidate)
	if err != nil {
		return err
	}
	if candidateInTrie == nil {
		return errors.New("invalid candidate to undelegate")
	}

	oldCandidate, err := d.voteTrie.TryGet(delegator)
	if err != nil {
		return err
	}
	if !bytes.Equal(candidate, oldCandidate) {
		return errors.New("mismatch candidate to undelegate")
	}

	//TODO
	oldDelegateTrie, err := d.delegateTrie.TryGet(append(candidate, delegator...))
	fmt.Println("[DEBUG]DPOS选举调试 delegateTrie after:", oldDelegateTrie)
	fmt.Println("[DEBUG]DPOS选举调试 voteTrie after:", oldCandidate)

	if err = d.delegateTrie.TryDelete(append(candidate, delegator...)); err != nil {
		return err
	}

	root, err := d.DelegateTrie().Commit(nil)
	d.DelegateTrie().TrieDB().Commit(root, true)

	if err = d.voteTrie.TryDelete(delegator); err != nil {
		return err
	}

	root, err = d.VoteTrie().Commit(nil)
	d.VoteTrie().TrieDB().Commit(root, true)

	//TODO
	newDelegateTrie, err := d.delegateTrie.TryGet(append(candidate, delegator...))
	fmt.Println("[DEBUG]DPOS选举调试 delegateTrie befor:", newDelegateTrie)

	newVoteTrie, _ := d.voteTrie.TryGet(delegator)
	fmt.Println("[DEBUG]DPOS选举调试 voteTrie befor:", newVoteTrie)

	return nil
}

func (d *DposContext) Commit(onleaf trie.LeafCallback) (*DposContextProto, error) {
	epochRoot, err := d.epochTrie.Commit(onleaf)
	log.Info("epochRoot:", hex.EncodeToString(epochRoot[:]), err)
	if err != nil {
		return nil, err
	}
	if err := d.epochTrie.TrieDB().Commit(epochRoot, true); err != nil {
		return nil, err
	}

	delegateRoot, err := d.delegateTrie.Commit(onleaf)
	log.Info("delegateRoot:", hex.EncodeToString(delegateRoot[:]), err)
	if err != nil {
		return nil, err
	}
	if err := d.delegateTrie.TrieDB().Commit(delegateRoot, true); err != nil {
		return nil, err
	}

	voteRoot, err := d.voteTrie.Commit(onleaf)
	if err != nil {
		return nil, err
	}
	if err := d.voteTrie.TrieDB().Commit(voteRoot, true); err != nil {
		return nil, err
	}

	candidateRoot, err := d.candidateTrie.Commit(onleaf)
	if err != nil {
		return nil, err
	}
	if err := d.candidateTrie.TrieDB().Commit(candidateRoot, true); err != nil {
		return nil, err
	}

	mintCntRoot, err := d.mintCntTrie.Commit(onleaf)
	if err != nil {
		return nil, err
	}
	if err := d.mintCntTrie.TrieDB().Commit(mintCntRoot, true); err != nil {
		return nil, err
	}

	return &DposContextProto{
		EpochHash:     epochRoot,
		DelegateHash:  delegateRoot,
		VoteHash:      voteRoot,
		CandidateHash: candidateRoot,
		MintCntHash:   mintCntRoot,
	}, nil
}

func (d *DposContext) CandidateTrie() *trie.Trie          { return d.candidateTrie }
func (d *DposContext) DelegateTrie() *trie.Trie           { return d.delegateTrie }
func (d *DposContext) VoteTrie() *trie.Trie               { return d.voteTrie }
func (d *DposContext) EpochTrie() *trie.Trie              { return d.epochTrie }
func (d *DposContext) MintCntTrie() *trie.Trie            { return d.mintCntTrie }
func (d *DposContext) DB() database.Database              { return d.db }
func (dc *DposContext) SetEpoch(epoch *trie.Trie)         { dc.epochTrie = epoch }
func (dc *DposContext) SetDelegate(delegate *trie.Trie)   { dc.delegateTrie = delegate }
func (dc *DposContext) SetVote(vote *trie.Trie)           { dc.voteTrie = vote }
func (dc *DposContext) SetCandidate(candidate *trie.Trie) { dc.candidateTrie = candidate }
func (dc *DposContext) SetMintCnt(mintCnt *trie.Trie)     { dc.mintCntTrie = mintCnt }

func (dc *DposContext) GetValidators() ([]common.Address, error) {
	var validators []common.Address
	key := []byte("validator")
	validatorsRLP := dc.epochTrie.Get(key)
	if err := rlp.DecodeBytes(validatorsRLP, &validators); err != nil {
		return nil, fmt.Errorf("failed to decode validators: %s", err)
	}
	return validators, nil
}

func (dc *DposContext) SetValidators(validators []common.Address) error {
	key := []byte("validator")
	validatorsRLP, err := rlp.EncodeToBytes(validators)
	if err != nil {
		return fmt.Errorf("failed to encode validators to rlp bytes: %s", err)
	}
	dc.epochTrie.Update(key, validatorsRLP)
	return nil
}

//TODO
func (dc *DposContext) GetCandidate() ([]common.Address, error) {

	it := trie.NewIterator(dc.CandidateTrie().NodeIterator(nil))
	for it.Next() {
		add := dc.CandidateTrie().Get(it.Key)
		fmt.Println("##############################GetCandidate:", add)
	}

	return nil, nil
}
