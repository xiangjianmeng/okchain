// Copyright The go-okchain Authors 2018,  All rights reserved.

/* peerendpoint.go - the main structure of PeerEndpointList  */
/*
* Revise history
* --------------------
* 2018/11/26, by Zhong Qiu, create
 */
/*
* DESCRIPTION

 */

package protos

import (
	"encoding/binary"
	"reflect"
	"sort"
	"sync"
)

type PoWSubListSlice []*PoWSubmission

func (p PoWSubListSlice) Len() int {
	return len(p)
}

func (p PoWSubListSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type ByNounce struct{ PoWSubListSlice }

func (b ByNounce) Less(i, j int) bool {
	randi := binary.BigEndian.Uint64(b.PoWSubListSlice[i].PublicKey)
	randj := binary.BigEndian.Uint64(b.PoWSubListSlice[j].PublicKey)
	return b.PoWSubListSlice[i].Nonce*10000+(randi>>48)&0xFFF > b.PoWSubListSlice[j].Nonce*10000+(randj>>48)&0xFFF
}

type PeerEndpointList []*PeerEndpoint

func (list PeerEndpointList) Dump() {

	logger.Debugf("------------------")
	for _, p := range list {
		logger.Debugf("%+v", p.Id)
	}
}
func (list PeerEndpointList) Length() int {
	return len(list)
}

func (list PeerEndpointList) Has(e *PeerEndpoint) bool {

	res := false
	for _, peer := range list {
		if reflect.DeepEqual(peer.Pubkey, e.Pubkey) {
			res = true
			break
		}
	}

	return res
}

func (list PeerEndpointList) Index(e *PeerEndpoint) int {
	for i, peer := range list {
		//if reflect.DeepEqual(peer.Pubkey, e.Pubkey) {
		if peer.Id.Name == e.Id.Name {
			return i
		}
	}
	return -1
}

func (list PeerEndpointList) EnqueueAndDequeueTail(e *PeerEndpoint) PeerEndpointList {
	var new PeerEndpointList
	if list.Length() > 1 {
		new = append(PeerEndpointList{e}, list[0:list.Length()-1]...)
	} else {
		new = PeerEndpointList{e}
	}
	return new
}

func (p PeerEndpointList) Len() int { return len(p) }

func (p PeerEndpointList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

type ByPubkey struct{ PeerEndpointList }

func (b ByPubkey) Less(i, j int) bool {
	return binary.BigEndian.Uint64([]byte(b.PeerEndpointList[i].Pubkey)) < binary.BigEndian.Uint64([]byte(b.PeerEndpointList[j].Pubkey))
}

type PeerEndpointMap map[uint32]PeerEndpointList

func (peerMap PeerEndpointMap) Dump() {

	logger.Debugf("------------------")
	for k, v := range peerMap {
		logger.Debugf("	key<%d>:", k)
		v.Dump()
	}
}

type SyncPowSubmissions struct {
	lock     sync.Mutex
	powArray []*PoWSubmission
}

func NewSyncPowSubmissions() *SyncPowSubmissions {
	res := &SyncPowSubmissions{}
	res.powArray = make([]*PoWSubmission, 0)
	return res
}

func (p *SyncPowSubmissions) Append(pow *PoWSubmission) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.powArray = append(p.powArray, pow)
}

func (p *SyncPowSubmissions) Len() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return len(p.powArray)
}

func (p *SyncPowSubmissions) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.powArray = make([]*PoWSubmission, 0)
}

func (p *SyncPowSubmissions) Get(index int) *PoWSubmission {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.powArray) <= index {
		return nil
	}
	return p.powArray[index]
}

func (p *SyncPowSubmissions) Has(e *PeerEndpoint) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	res := false
	for _, peer := range p.powArray {
		if reflect.DeepEqual(peer.Peer.Pubkey, e.Pubkey) {
			res = true
			break
		}
	}

	return res
}

func (p *SyncPowSubmissions) Sort() {
	p.lock.Lock()
	defer p.lock.Unlock()
	sort.Sort(ByNounce{p.powArray})
}

func (p *SyncPowSubmissions) Dump() {
	for i := 0; i < len(p.powArray); i++ {
		logger.Debugf("powArrage<%d>: %+v", i, p.powArray[i])
	}
}

type SyncMicroBlockSubmissions struct {
	lock    sync.Mutex
	mbArray []*MicroBlock
}

func NewSyncMicroBlockSubmissions() *SyncMicroBlockSubmissions {
	res := &SyncMicroBlockSubmissions{}
	res.mbArray = make([]*MicroBlock, 0)
	return res
}

func (p *SyncMicroBlockSubmissions) Append(mb *MicroBlock) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.mbArray = append(p.mbArray, mb)
}

func (p *SyncMicroBlockSubmissions) Len() int {
	p.lock.Lock()
	defer p.lock.Unlock()
	return len(p.mbArray)
}

func (p *SyncMicroBlockSubmissions) Clear() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.mbArray = make([]*MicroBlock, 0)
}

func (p *SyncMicroBlockSubmissions) Get(index int) *MicroBlock {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.mbArray) <= index {
		return nil
	}
	return p.mbArray[index]
}

func (p *SyncMicroBlockSubmissions) Has(e *PeerEndpoint) bool {
	p.lock.Lock()
	defer p.lock.Unlock()
	res := false
	for _, peer := range p.mbArray {
		if reflect.DeepEqual(peer.Miner.Pubkey, e.Pubkey) {
			res = true
			break
		}
	}

	return res
}

func (p *SyncMicroBlockSubmissions) Dump() {
	logger.Debugf("------------------")
	for i := 0; i < len(p.mbArray); i++ {
		logger.Debugf("micro block <%d>: %+v", i, p.mbArray[i])
	}
}
