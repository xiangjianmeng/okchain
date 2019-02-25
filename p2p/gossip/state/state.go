// Copyright The go-okchain Authors 2018,  All rights reserved.

// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package state

import (
	"bytes"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/core/blockchain"
	"github.com/ok-chain/okchain/p2p/gossip/comm"
	"github.com/ok-chain/okchain/p2p/gossip/common"
	"github.com/ok-chain/okchain/p2p/gossip/discovery"
	"github.com/ok-chain/okchain/p2p/gossip/util"
	"github.com/ok-chain/okchain/protos"
	"github.com/pkg/errors"
)

// GossipStateProvider is the interface to acquire sequences of the ledger blocks
// capable to full fill missing blocks by running state replication and
// sending request to get missing block to other nodes
type GossipStateProvider interface {
	// AddPayload(payload *protos.Payload) error
	MaxAvailableLedgerHeight(isDsBlock bool) uint64
	// Stop terminates state transfer object
	Stop()
}

const (
	defAntiEntropyInterval             = 5 * time.Second
	defAntiEntropyStateResponseTimeout = 3 * time.Second
	defAntiEntropyBatchSize            = 10

	defChannelBufferSize     = 100
	defAntiEntropyMaxRetries = 3

	defMaxBlockDistance = 100

	blocking    = true
	nonBlocking = false

	enqueueRetryInterval = time.Millisecond * 100
)

// GossipAdapter defines gossip/communication required interface for state provider
type GossipAdapter interface {
	// Send sends a message to remote peers
	Send(msg *protos.GossipMessage, peers ...*comm.RemotePeer)

	// Accept returns a dedicated read-only channel for messages sent by other nodes that match a certain predicate.
	// If passThrough is false, the messages are processed by the gossip layer beforehand.
	// If passThrough is true, the gossip layer doesn't intervene and the messages
	// can be used to send a reply back to the sender
	Accept(acceptor common.MessageAcceptor, passThrough bool) (<-chan *protos.GossipMessage, <-chan protos.ReceivedMessage)

	// UpdateLedgerHeight updates the ledger height the peer
	// publishes to other peers in the channel
	UpdateDsLedgerHeight(height uint64, chainID common.ChainID)

	// UpdateLedgerHeight updates the ledger height the peer
	// publishes to other peers in the channel
	UpdateTxLedgerHeight(height uint64, chainID common.ChainID)

	// PeersOfChannel returns the NetworkMembers considered alive
	// and also subscribed to the channel given
	PeersOfChannel(common.ChainID) []discovery.NetworkMember
}

// GossipStateProviderImpl the implementation of the GossipStateProvider interface
// the struct to handle in memory sliding window of
// new ledger block to be acquired by hyper ledger
type GossipStateProviderImpl struct {
	// Chain id
	chainID  string
	mediator GossipAdapter

	// Queue of payloads which wasn't acquired yet
	// dsPayloads PayloadsBuffer
	// txPayloads PayloadsBuffer

	dsLedger *blockchain.DSBlockChain
	txLedger *blockchain.TxBlockChain

	commChan        <-chan protos.ReceivedMessage
	dsBlockCh       chan<- protos.DSBlock
	txBlockCh       chan<- protos.TxBlock
	stateResponseCh chan protos.ReceivedMessage
	stateRequestCh  chan protos.ReceivedMessage
	stopCh          chan struct{}

	done sync.WaitGroup
	once sync.Once

	stateTransferActive int32
}

var logger = util.GetLogger(util.LoggingStateModule, "")

// NewGossipStateProvider creates state provider with coordinator instance
// to orchestrate arrival of private rwsets and blocks before committing them into the ledger.
func NewGossipStateProvider(chainID string, services GossipAdapter,
	dsLedger *blockchain.DSBlockChain,
	txLedger *blockchain.TxBlockChain,
	dsBlock chan protos.DSBlock,
	txBlock chan protos.TxBlock) GossipStateProvider {

	remoteStateMsgFilter := func(message interface{}) bool {
		receivedMsg := message.(protos.ReceivedMessage)
		msg := receivedMsg.GetGossipMessage()
		if !(msg.IsRemoteStateMessage() || msg.GetPrivateData() != nil) {
			return false
		}
		// Ensure we deal only with messages that belong to this channel
		if !bytes.Equal(msg.Channel, []byte(chainID)) {
			return false
		}

		return true
	}

	// Filter message which are only relevant for nodeMetastate transfer
	_, commChan := services.Accept(remoteStateMsgFilter, true)

	txHeight := txLedger.Height()
	if txHeight == 0 {
		// Panic here since this is an indication of invalid situation which should not happen in normal
		// code path.
		logger.Panic("Tx ledger height cannot be zero, ledger should include at least one block (genesis).")
	}
	dsHeight := dsLedger.Height()
	if dsHeight == 0 {
		// Panic here since this is an indication of invalid situation which should not happen in normal
		// code path.
		logger.Panic("Ds ledger height cannot be zero, ledger should include at least one block (genesis).")
	}

	s := &GossipStateProviderImpl{
		// MessageCryptoService
		mediator: services,

		// Chain ID
		chainID: chainID,

		// Channel to read direct messages from other peers
		commChan: commChan,

		// Create a queue for payload received
		// dsPayloads: NewPayloadsBuffer(dsHeight),
		// txPayloads: NewPayloadsBuffer(txHeight),

		dsLedger: dsLedger,
		txLedger: txLedger,

		dsBlockCh: dsBlock,
		txBlockCh: txBlock,

		stateResponseCh: make(chan protos.ReceivedMessage, defChannelBufferSize),

		stateRequestCh: make(chan protos.ReceivedMessage, defChannelBufferSize),

		stopCh: make(chan struct{}, 1),

		stateTransferActive: 0,

		once: sync.Once{},
	}

	// logger.Infof("Updating metadata information, "+
	// 	"current dsledger sequence is at = %d, next expected dsblock is = %d", dsHeight-1, s.dsPayloads.Next())
	logger.Debug("Updating gossip dsledger height to", txHeight)
	services.UpdateDsLedgerHeight(dsHeight, common.ChainID(s.chainID))

	// logger.Infof("Updating metadata information, "+
	// 	"current txledger sequence is at = %d, next expected txblock is = %d", txHeight-1, s.txPayloads.Next())
	logger.Debug("Updating gossip txledger height to", txHeight)
	services.UpdateTxLedgerHeight(txHeight, common.ChainID(s.chainID))

	s.done.Add(4)

	// Listen for incoming communication
	go s.listen()

	// Deliver in order messages into the incoming channel
	// go s.deliverDsPayloads()

	// Execute anti entropy to fill missing gaps
	go s.antiEntropyDs()
	// Taking care of state request messages
	go s.processStateRequests()

	time.Sleep(5 * time.Second)
	// Deliver in order messages into the incoming channel
	// go s.deliverPayloads()
	// Execute anti entropy to fill missing gaps
	go s.antiEntropy()

	return s
}

func (s *GossipStateProviderImpl) listen() {
	defer s.done.Done()

	for {
		select {
		case msg := <-s.commChan:
			logger.Debug("Dispatching a message", msg)
			// Check type of the message
			if msg.GetGossipMessage().IsRemoteStateMessage() {
				logger.Debug("Handling direct state transfer message")
				// Got state transfer request response
				go s.directMessage(msg)
			}
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			logger.Debug("Stop listening for new messages")
			return
		}
	}
}

func (s *GossipStateProviderImpl) directMessage(msg protos.ReceivedMessage) {
	logger.Debug("[ENTER] -> directMessage")
	defer logger.Debug("[EXIT] ->  directMessage")

	if msg == nil {
		logger.Error("Got nil message via end-to-end channel, should not happen!")
		return
	}

	if !bytes.Equal(msg.GetGossipMessage().Channel, []byte(s.chainID)) {
		logger.Warning("Received state transfer request for channel",
			string(msg.GetGossipMessage().Channel), "while expecting channel", s.chainID, "skipping request...")
		return
	}

	incoming := msg.GetGossipMessage()

	if incoming.GetStateRequest() != nil {
		if len(s.stateRequestCh) < defChannelBufferSize {
			// Forward state request to the channel, if there are too
			// many message of state request ignore to avoid flooding.
			s.stateRequestCh <- msg
		}
	} else if incoming.GetStateResponse() != nil {
		// If no state transfer procedure activate there is
		// no reason to process the message
		if atomic.LoadInt32(&s.stateTransferActive) == 1 {
			// Send signal of state response message
			s.stateResponseCh <- msg
		}
	}
}

func (s *GossipStateProviderImpl) processStateRequests() {
	defer s.done.Done()

	for {
		select {
		case msg := <-s.stateRequestCh:
			s.handleStateRequest(msg)
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			return
		}
	}
}

// handleStateRequest handles state request message, validate batch size, reads current leader state to
// obtain required blocks, builds response message and send it back
func (s *GossipStateProviderImpl) handleStateRequest(msg protos.ReceivedMessage) {
	if msg == nil {
		return
	}
	request := msg.GetGossipMessage().GetStateRequest()

	batchSize := request.EndSeqNum - request.StartSeqNum
	if batchSize > defAntiEntropyBatchSize {
		logger.Errorf("Requesting blocks batchSize size (%d) greater than configured allowed"+
			" (%d) batching for anti-entropy. Ignoring request...", batchSize, defAntiEntropyBatchSize)
		return
	}

	if request.StartSeqNum > request.EndSeqNum {
		logger.Errorf("Invalid sequence interval [%d...%d), ignoring request...", request.StartSeqNum, request.EndSeqNum)
		return
	}

	currentHeight := s.txLedger.Height()
	if request.IsDsBlock {
		currentHeight = s.dsLedger.Height()
	}
	if currentHeight < request.EndSeqNum {
		logger.Warningf("Received state request to transfer blocks with sequence numbers higher  [%d...%d) "+
			"than available in ledger (%d)", request.StartSeqNum, request.StartSeqNum, currentHeight)
	}

	endSeqNum := min(currentHeight, request.EndSeqNum)

	response := &protos.RemoteStateResponse{Payloads: make([]*protos.Payload, 0), IsDsBlock: request.IsDsBlock}
	for seqNum := request.StartSeqNum; seqNum < endSeqNum; seqNum++ {
		if request.IsDsBlock {
			logger.Debug("Reading dsblock ", seqNum, " from the coordinator service")
			block := s.dsLedger.GetBlockByNumber(seqNum)
			if block == nil {
				logger.Errorf("Wasn't able to read dsblock with sequence number %d from dsledger, skipping....", seqNum)
				continue
			}

			blockBytes, err := pb.Marshal(block.(*protos.DSBlock))
			if err != nil {
				logger.Errorf("Could not marshal dsblock: %+v", errors.WithStack(err))
				continue
			}
			// Appending result to the response
			response.Payloads = append(response.Payloads, &protos.Payload{
				SeqNum: seqNum,
				Data:   blockBytes,
			})
		} else {
			logger.Debug("Reading txblock ", seqNum, " from the coordinator service")
			block := s.txLedger.GetBlockByNumber(seqNum)
			if block == nil {
				logger.Errorf("Wasn't able to read txblock with sequence number %d from txledger, skipping....", seqNum)
				continue
			}

			blockBytes, err := pb.Marshal(block.(*protos.TxBlock))
			if err != nil {
				logger.Errorf("Could not marshal txblock: %+v", errors.WithStack(err))
				continue
			}
			// Appending result to the response
			response.Payloads = append(response.Payloads, &protos.Payload{
				SeqNum: seqNum,
				Data:   blockBytes,
			})
		}
	}

	logger.Debugf("Sending back response with %d missing blocks to %s", len(response.Payloads), msg.GetConnectionInfo().Endpoint)
	// Sending back response with missing blocks
	msg.Respond(&protos.GossipMessage{
		// Copy nonce field from the request, so it will be possible to match response
		Nonce:   msg.GetGossipMessage().Nonce,
		Tag:     protos.GossipMessage_CHAN_OR_ORG,
		Channel: []byte(s.chainID),
		Content: &protos.GossipMessage_StateResponse{StateResponse: response},
	})
}

func (s *GossipStateProviderImpl) handleStateResponse(msg protos.ReceivedMessage) (uint64, error) {
	max := uint64(0)
	// Send signal that response for given nonce has been received
	response := msg.GetGossipMessage().GetStateResponse()
	// Extract payloads, verify and push into buffer
	if len(response.GetPayloads()) == 0 {
		return uint64(0), errors.New("Received state transfer response without payload")
	}
	for _, payload := range response.GetPayloads() {
		logger.Debugf("Received payload with sequence number %d.", payload.SeqNum)
		if max < payload.SeqNum {
			max = payload.SeqNum
		}

		err := s.addPayload(payload, blocking, response.IsDsBlock)
		if err != nil {
			logger.Warningf("Block [%d] received from block transfer wasn't added to payload buffer: %v", payload.SeqNum, err)
		}
	}
	return max, nil
}

// Stop function sends halting signal to all go routines
func (s *GossipStateProviderImpl) Stop() {
	// Make sure stop won't be executed twice
	// and stop channel won't be used again
	s.once.Do(func() {
		s.stopCh <- struct{}{}
		// Make sure all go-routines has finished
		s.done.Wait()
		// Close all resources
		close(s.stateRequestCh)
		close(s.stateResponseCh)
		close(s.stopCh)
		close(s.dsBlockCh)
		close(s.txBlockCh)
	})
}

// func (s *GossipStateProviderImpl) deliverPayloads() {
// 	defer s.done.Done()

// 	for {
// 		select {
// 		// Wait for notification that next seq has arrived
// 		case <-s.txPayloads.Ready():
// 			logger.Debugf("[%s] Ready to transfer payloads (txblocks) to the txledger, next txblock number is = [%d]", s.chainID, s.txPayloads.Next())
// 			// Collect all subsequent payloads
// 			for payload := s.txPayloads.Pop(); payload != nil; payload = s.txPayloads.Pop() {
// 				rawTxBlock := &protos.TxBlock{}
// 				if err := pb.Unmarshal(payload.Data, rawTxBlock); err != nil {
// 					logger.Errorf("Error getting txblock with seqNum = %d due to (%+v)...dropping txblock", payload.SeqNum, errors.WithStack(err))
// 					continue
// 				}
// 				if rawTxBlock.Body == nil || rawTxBlock.Header == nil {
// 					logger.Errorf("TxBlock with claimed sequence %d has no header (%v) or data (%v)",
// 						payload.SeqNum, rawTxBlock.Header, rawTxBlock.Body)
// 					continue
// 				}
// 				logger.Debugf("[%s] Transferring txblock [%d] with %d transaction(s) to the txledger", s.chainID, payload.SeqNum, len(rawTxBlock.Body.Transactions))

// 				s.txBlockCh <- *rawTxBlock
// 			}
// 		case <-s.stopCh:
// 			s.stopCh <- struct{}{}
// 			logger.Debug("State provider has been stopped, finishing to push new txblocks.")
// 			return
// 		}
// 	}
// }

// func (s *GossipStateProviderImpl) deliverDsPayloads() {
// 	defer s.done.Done()

// 	for {
// 		select {
// 		// Wait for notification that next seq has arrived
// 		case <-s.dsPayloads.Ready():
// 			logger.Debugf("[%s] Ready to transfer payloads (dsblocks) to the dsledger, next dsblock number is = [%d]", s.chainID, s.dsPayloads.Next())
// 			// Collect all subsequent payloads
// 			for payload := s.dsPayloads.Pop(); payload != nil; payload = s.dsPayloads.Pop() {
// 				rawDsBlock := &protos.DSBlock{}
// 				if err := pb.Unmarshal(payload.Data, rawDsBlock); err != nil {
// 					logger.Errorf("Error getting dsblock with seqNum = %d due to (%+v)...dropping dsblock", payload.SeqNum, errors.WithStack(err))
// 					continue
// 				}
// 				if rawDsBlock.Body == nil || rawDsBlock.Header == nil {
// 					logger.Errorf("DsBlock with claimed sequence %d has no header (%v) or data (%v)",
// 						payload.SeqNum, rawDsBlock.Header, rawDsBlock.Body)
// 					continue
// 				}
// 				logger.Debugf("[%s] Transferring dsblock [%d] to the ledger", s.chainID, payload.SeqNum)

// 				s.dsBlockCh <- *rawDsBlock
// 			}
// 		case <-s.stopCh:
// 			s.stopCh <- struct{}{}
// 			logger.Debug("State provider has been stopped, finishing to push new dsblocks.")
// 			return
// 		}
// 	}
// }

func (s *GossipStateProviderImpl) antiEntropy() {
	defer s.done.Done()
	defer logger.Debug("State Provider stopped, stopping anti entropy procedure.")

	for {
		select {
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			return
		case <-time.After(defAntiEntropyInterval):
			ourHeight := s.txLedger.Height()
			if ourHeight == 0 {
				logger.Error("TxLedger reported txblock height of 0 but this should be impossible")
				continue
			}

			maxHeight := s.MaxAvailableLedgerHeight(false)
			logger.Debugf("My Txledger height=%d, max available ledger height=%d", ourHeight, maxHeight)
			if ourHeight >= maxHeight {
				continue
			}

			s.requestBlocksInRange(uint64(ourHeight), uint64(maxHeight), false)
		}
	}
}

func (s *GossipStateProviderImpl) antiEntropyDs() {
	defer s.done.Done()
	defer logger.Debug("State Provider stopped, stopping anti entropy procedure.")

	for {
		select {
		case <-s.stopCh:
			s.stopCh <- struct{}{}
			return
		case <-time.After(defAntiEntropyInterval):
			ourHeight := s.dsLedger.Height()
			if ourHeight == 0 {
				logger.Error("DsLedger reported dsblock height of 0 but this should be impossible")
				continue
			}

			maxHeight := s.MaxAvailableLedgerHeight(true)
			logger.Debugf("My Dsledger height=%d, max available ledger height=%d", ourHeight, maxHeight)
			if ourHeight >= maxHeight {
				continue
			}

			s.requestBlocksInRange(uint64(ourHeight), uint64(maxHeight), true)
		}
	}
}

// MaxAvailableLedgerHeight iterates over all available peers and checks advertised meta state to
// find maximum available ledger height across peers
func (s *GossipStateProviderImpl) MaxAvailableLedgerHeight(isDsBlock bool) uint64 {
	max := uint64(0)

	for _, p := range s.mediator.PeersOfChannel(common.ChainID(s.chainID)) {
		if p.Properties == nil {
			logger.Debug("Peer", p.PreferredEndpoint(), "doesn't have properties, skipping it")
			continue
		}

		peerHeight := p.Properties.TxLedgerHeight
		if isDsBlock {
			peerHeight = p.Properties.DsLedgerHeight
		}
		if max < peerHeight {
			max = peerHeight
		}
	}
	return max
}

// requestBlocksInRange capable to acquire blocks with sequence
// numbers in the range [start...end).
func (s *GossipStateProviderImpl) requestBlocksInRange(start uint64, end uint64, isDsBlock bool) {
	atomic.StoreInt32(&s.stateTransferActive, 1)
	defer atomic.StoreInt32(&s.stateTransferActive, 0)

	for prev := start; prev < end; {
		next := min(end, prev+defAntiEntropyBatchSize)

		gossipMsg := s.stateRequestMessage(prev, next, isDsBlock)

		responseReceived := false
		tryCounts := 0

		for !responseReceived {
			if tryCounts > defAntiEntropyMaxRetries {
				logger.Warningf("Wasn't  able to get blocks in range [%d...%d), after %d retries",
					prev, next, tryCounts)
				return
			}
			// Select peers to ask for blocks
			peer, err := s.selectPeerToRequestFrom(next, isDsBlock)
			if err != nil {
				logger.Warningf("Cannot send state request for blocks in range [%d...%d）, due to %+v",
					prev, next, errors.WithStack(err))
				return
			}

			logger.Debugf("State transfer, with peer %s, requesting blocks in range [%d...%d）, "+
				"for chainID %s", peer.Endpoint, prev, next, s.chainID)

			s.mediator.Send(gossipMsg, peer)
			tryCounts++

			// Wait until timeout or response arrival
			select {
			case msg := <-s.stateResponseCh:
				if msg.GetGossipMessage().Nonce != gossipMsg.Nonce {
					continue
				}
				// Got corresponding response for state request, can continue
				index, err := s.handleStateResponse(msg)
				if err != nil {
					logger.Warningf("Wasn't able to process state response from %s for "+
						"blocks [%d...%d), due to %+v", peer.Endpoint, prev, next, errors.WithStack(err))
					continue
				}
				prev = index + 1
				responseReceived = true
			case <-time.After(defAntiEntropyStateResponseTimeout):
				logger.Warningf("Waiting response from %s for "+
					"blocks [%d...%d) timeout", peer.Endpoint, prev, next)
			case <-s.stopCh:
				s.stopCh <- struct{}{}
				return
			}
		}
	}
}

// stateRequestMessage generates state request message for given blocks in range [beginSeq...endSeq]
func (s *GossipStateProviderImpl) stateRequestMessage(beginSeq uint64, endSeq uint64, isDsBlock bool) *protos.GossipMessage {
	return &protos.GossipMessage{
		Nonce:   util.RandomUInt64(),
		Tag:     protos.GossipMessage_CHAN_OR_ORG,
		Channel: []byte(s.chainID),
		Content: &protos.GossipMessage_StateRequest{
			StateRequest: &protos.RemoteStateRequest{
				StartSeqNum: beginSeq,
				EndSeqNum:   endSeq,
				IsDsBlock:   isDsBlock,
			},
		},
	}
}

// selectPeerToRequestFrom selects peer which has required blocks to ask missing blocks from
func (s *GossipStateProviderImpl) selectPeerToRequestFrom(height uint64, isDsBlock bool) (*comm.RemotePeer, error) {
	// Filter peers which posses required range of missing blocks
	peers := s.filterPeers(s.hasRequiredHeight(height, isDsBlock))

	n := len(peers)
	if n == 0 {
		return nil, errors.New("there are no peers to ask for missing blocks from")
	}

	// Select peer to ask for blocks
	return peers[util.RandomInt(n)], nil
}

// filterPeers returns list of peers which aligns the predicate provided
func (s *GossipStateProviderImpl) filterPeers(predicate func(peer discovery.NetworkMember) bool) []*comm.RemotePeer {
	var peers []*comm.RemotePeer

	for _, member := range s.mediator.PeersOfChannel(common.ChainID(s.chainID)) {
		if predicate(member) {
			peers = append(peers, &comm.RemotePeer{Endpoint: member.PreferredEndpoint(), PKIID: member.PKIid})
		}
	}

	return peers
}

// hasRequiredHeight returns predicate which is capable to filter peers with ledger height above than indicated
// by provided input parameter
func (s *GossipStateProviderImpl) hasRequiredHeight(height uint64, isDsBlock bool) func(peer discovery.NetworkMember) bool {
	return func(peer discovery.NetworkMember) bool {
		if peer.Properties != nil {
			if isDsBlock {
				return peer.Properties.DsLedgerHeight >= height
			}
			return peer.Properties.TxLedgerHeight >= height
		}
		logger.Debug(peer.PreferredEndpoint(), "doesn't have properties")
		return false
	}
}

// addPayload adds new payload into state. It may (or may not) block according to the
// given parameter. If it gets a block while in blocking mode - it would wait until
// the block is sent into the payloads buffer.
// Else - it may drop the block, if the payload buffer is too full.
func (s *GossipStateProviderImpl) addPayload(payload *protos.Payload, blockingMode, isDsBlock bool) error {
	if payload == nil {
		return errors.New("Given payload is nil")
	}
	logger.Debugf("[%s] Adding payload to local buffer, blockNum = [%d]", s.chainID, payload.SeqNum)
	height := s.txLedger.Height()
	if isDsBlock {
		height = s.dsLedger.Height()
	}

	if !blockingMode && payload.SeqNum-height >= defMaxBlockDistance {
		return errors.Errorf("Ledger height is at %d, cannot enqueue block with sequence of %d", height, payload.SeqNum)
	}

	// for blockingMode && s.txPayloads.Size() > defMaxBlockDistance*2 {
	// 	time.Sleep(enqueueRetryInterval)
	// }

	if isDsBlock {
		// s.dsPayloads.Push(payload)
		rawDsBlock := &protos.DSBlock{}
		if err := pb.Unmarshal(payload.Data, rawDsBlock); err != nil {
			logger.Errorf("Error getting dsblock with seqNum = %d due to (%+v)...dropping dsblock", payload.SeqNum, errors.WithStack(err))
		}
		if rawDsBlock.Body == nil || rawDsBlock.Header == nil {
			logger.Errorf("DsBlock with claimed sequence %d has no header (%v) or data (%v)",
				payload.SeqNum, rawDsBlock.Header, rawDsBlock.Body)
		}
		logger.Debugf("[%s] Transferring dsblock [%d] to the ledger", s.chainID, payload.SeqNum)

		s.dsBlockCh <- *rawDsBlock
	} else {
		// s.txPayloads.Push(payload)
		rawTxBlock := &protos.TxBlock{}
		if err := pb.Unmarshal(payload.Data, rawTxBlock); err != nil {
			logger.Errorf("Error getting txblock with seqNum = %d due to (%+v)...dropping txblock", payload.SeqNum, errors.WithStack(err))
		}
		if rawTxBlock.Body == nil || rawTxBlock.Header == nil {
			logger.Errorf("TxBlock with claimed sequence %d has no header (%v) or data (%v)",
				payload.SeqNum, rawTxBlock.Header, rawTxBlock.Body)
		}
		logger.Debugf("[%s] Transferring txblock [%d] with %d transaction(s) to the txledger", s.chainID, payload.SeqNum, len(rawTxBlock.Body.Transactions))

		s.txBlockCh <- *rawTxBlock
	}
	return nil
}

func min(a uint64, b uint64) uint64 {
	return b ^ ((a ^ b) & (-(uint64(a-b) >> 63)))
}
