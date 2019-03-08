// Copyright The go-okchain Authors 2018,  All rights reserved.

package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/ok-chain/okchain/accounts"
	"github.com/ok-chain/okchain/accounts/keystore"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/event"
	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/core"
	"github.com/ok-chain/okchain/core/blockchain"
	"github.com/ok-chain/okchain/core/blockchain/dsblockchain"
	"github.com/ok-chain/okchain/core/blockchain/txblockchain"
	"github.com/ok-chain/okchain/core/consensus/mine"
	"github.com/ok-chain/okchain/core/database"
	"github.com/ok-chain/okchain/core/txpool"
	"github.com/ok-chain/okchain/crypto/multibls"
	gossip_common "github.com/ok-chain/okchain/p2p/gossip/common"
	"github.com/ok-chain/okchain/p2p/gossip/gossip"
	"github.com/ok-chain/okchain/p2p/gossip/state"
	pb "github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/util"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// PeerServer implementation of the Peer service
type PeerServer struct {
	StateContainerFactoryType
	RoleContainerFactoryType

	IConsensusLeadFactoryType
	IConsensusBackupFactoryType

	///////////////////////////////////////
	// external member
	ShardingID    uint32
	Committee     pb.PeerEndpointList // all ds, head is the lead
	ShardingNodes pb.PeerEndpointList // all shard nodes, lead? sorted by public key
	ShardingMap   pb.PeerEndpointMap  // shard id -> members in this shardingid
	LookupNodes   pb.PeerEndpointList // all lookup nodes
	SelfNode      *pb.PeerEndpoint

	///////////////////////////////////////
	// internal member
	peerHandlerMap *remotePeerManager
	isLookup       bool
	reconnectOnce  sync.Once
	mode           string
	Name           string
	mux            sync.Mutex
	commNodeInfo   sync.Map
	lock           sync.Mutex
	currentRole    IRole

	roleContainer *RoleContainer
	Gossip        gossip.Gossip
	Miner         mine.Miner
	PublicKey     []byte
	PrivateKey    *multibls.PriKey
	CoinBase      []byte
	Registered    chan struct{}
	Signer        pb.CSigner
	MsgSinger     pb.BLSSigner

	shardingID uint32
	transPool  []*pb.Transaction
	msg        MsgChannel
	totalTXNum int64

	VcChan        chan string
	ConsensusData ConsensusData
	Flag          bool
	NotifyChannel chan *pb.Message

	// db handler for ds and tx blockchain
	dsBlockChain   *blockchain.DSBlockChain
	txBlockChain   *blockchain.TxBlockChain
	txpool         *txpool.TxPool
	txsCh          chan core.NewTxsEvent
	txsSub         event.Subscription
	accountManager *accounts.Manager

	initialDsAndLookupNodes []string

	Gsp state.GossipStateProvider
}

func NewPeer() (peer *PeerServer) {
	peer = &PeerServer{
		StateContainerFactoryType:   StateContainerFactory,
		RoleContainerFactoryType:    RoleContainerFactory,
		IConsensusBackupFactoryType: ConsensusBackupFactory,
		IConsensusLeadFactoryType:   ConsensusLeadFactory,
	}

	peer.totalTXNum = 0
	peer.ShardingMap = make(pb.PeerEndpointMap)
	peer.msg = MsgChannel{}
	peer.NotifyChannel = make(chan *pb.Message)
	peer.VcChan = make(chan string)
	peer.ConsensusData = ConsensusData{
		CurrentDSBlock:    &pb.DSBlock{},
		CurrentFinalBlock: &pb.TxBlock{},
		CurrentMicroBlock: &pb.MicroBlock{},
		CurrentVCBlock:    &pb.VCBlock{},
		PoWSubList:        pb.SyncPowSubmissions{},
		MicroBlockList:    pb.SyncMicroBlockSubmissions{},
	}
	peer.Flag = false
	var sharding []*pb.PeerEndpoint
	peer.ShardingNodes = sharding
	peer.peerHandlerMap = &remotePeerManager{m: make(map[string]IP2PHandler)}

	peer.initialDsAndLookupNodes = make([]string, 0, 600)
	peer.Registered = make(chan struct{}, 1)
	peer.roleContainer = peer.produceRoleContainer(peer)
	newStateContainer(peer.StateContainerFactoryType)
	peer.readConfig()
	return peer
}

func (p *PeerServer) initBlockChains() (err error) {
	DSBlockChainDB := "ds-chaindata"
	TxBlockChainDB := "tx-chaindata"

	prefixDBDir := viper.GetString("ledger.baseDir")
	txGenesisConf := viper.GetString("ledger.txblockchain.genesisConf")
	if prefixDBDir == "" {
		prefixDBDir = "./data_unittest/okchain/"
	}
	if txGenesisConf == "" {
		txGenesisConf = "./core/txblockchain/data/genesis.json"
	}
	dsChainDB, err := database.NewRDBDatabase(prefixDBDir+DSBlockChainDB, 16, 1024)

	if err != nil {
		panic(fmt.Errorf("Open DsBlockDB Error: <%s>", err.Error()))
	}

	txChainDB, err := database.NewRDBDatabase(prefixDBDir+TxBlockChainDB, 16, 1024)
	if err != nil {
		panic(fmt.Errorf("Open TxBlockDB Error: <%s>", err.Error()))
	}

	// 2018/09/18. FLT.  Fake params is passed to New_XXX(*, *)
	dsblockchain.SetupDefaultDSGenesisBlock(dsChainDB)
	p.dsBlockChain, err = blockchain.NewDSBlockChain(dsChainDB, nil, &p.MsgSinger)

	// 2018/09/19. FLT. New a finalblock Chain.
	config, _, err := txblockchain.SetupTxBlock(txChainDB, txGenesisConf)
	if err != nil {
		peerLogger.Fatal(err)
	}
	if p.txBlockChain, err = blockchain.NewTxBlockChain(config, txChainDB, blockchain.GetDefaultCacheConfig()); err != nil {
		peerLogger.Fatal(err)
	}
	validator := blockchain.NewBlockValidator(config, p.txBlockChain, p.dsBlockChain)
	p.txBlockChain.SetValidator(validator)
	p.txBlockChain.Processor().SetPeerServer(p)

	if err != nil {
		peerLogger.Error("InitBlockChains Init ERROR: " + err.Error())
	} else {
		peerLogger.Infof("InitBlockChains Init Successfully.")
	}

	backends := []accounts.Backend{
		keystore.NewKeyStore(viper.GetString("peer.dataDir")+"/keystore",
			keystore.StandardScryptN, keystore.StandardScryptP),
	}
	p.accountManager = accounts.NewManager(backends...)

	peerLogger.Debug("genesis block=", p.TxBlockChain().CurrentBlock())

	return err
}

func (p *PeerServer) Init(gossipPort int, listenAddr string, grpcServer *grpc.Server) {
	// Init TX & DS blockchains
	p.initBlockChains()

	txpool.CreateTxPool(txpool.DefaultTxPoolConfig, config.TestChainConfig, p.txBlockChain)
	transactionPool := txpool.GetTxPool()
	p.txsCh = make(chan core.NewTxsEvent, txChanSize)
	p.txsSub = transactionPool.SubscribeNewTxsEvent(p.txsCh)
	go p.txBroadcastLoop()

	port := strings.Split(listenAddr, ":")[len(strings.Split(listenAddr, ":"))-1]
	err := p.loadKey(port)
	if err != nil {
		panic(err.Error())
	}

	// todo, set by wallet or update by json rpc

	newpeer := &pb.PeerEndpoint{}
	newpeer.Pubkey = p.PublicKey
	newpeer.Id = &pb.PeerID{Name: listenAddr}
	newpeer.Address = strings.Split(listenAddr, ":")[0]
	iport, _ := strconv.Atoi(port)
	newpeer.Port = uint32(iport)
	// update self node in peerserver

	p.CoinBase = []byte("coinbase:" + newpeer.Id.Name)
	newpeer.Coinbase = p.CoinBase
	p.SelfNode = newpeer

	//peer.ConsensusData.PubKeyToCoinBaseMap = make(map[string][]byte)

	peerLogger.Debugf("coinbase addr is %s", common.BytesToAddress(p.CoinBase).Hex())
	peerLogger.Debugf("my node info is %+v", p.SelfNode)
	p.Signer = pb.MakeSigner("")

	p.ChangeRole(PeerRole_Idle, STATE_IDLE)
	mode := viper.GetString("peer.mode")
	if mode == "lookup" {
		p.ChangeRole(PeerRole_Lookup, STATE_LOOKUP)
	}

	luAddList = strings.Split(strings.Trim(strings.Trim(viper.GetString("peer.lookupNodeUrl"), " "), "|"), "|")
	var lookup []*pb.PeerEndpoint
	var lookupIds []int
	for k, lu := range luAddList {
		ipPort := strings.Split(lu, ":")
		if len(ipPort) <= 1 {
			continue
		}
		if ipPort[0] == "" {
			ipPort[0] = GetLocalIP()
		}
		port, err := strconv.Atoi(ipPort[1])
		if err != nil {
			peerLogger.Debugf("lookup node port error %s", err.Error())
		} else {
			lookupIds = append(lookupIds, port%100)
		}

		lu = ipPort[0] + ":" + ipPort[1]
		lookup = append(lookup,
			&pb.PeerEndpoint{
				Id:      &pb.PeerID{Name: lu},
				Address: ipPort[0],
				Port:    uint32(port),
			})
		luAddList[k] = lu
	}
	p.LookupNodes = lookup
	peerLogger.Debugf("lookup node id: %d", lookupIds)

	isBoot := false
	if mode == "lookup" {
		isBoot = true
	}
	p.Gossip = gossip.NewGossipInstance(gossipPort-gossipPort%100, gossipPort%100, 1000, isBoot, grpcServer)

	p.Gossip.JoinChan(&gossip.JoinChanMsg{}, gossip_common.ChainID("A"))
	p.Gossip.UpdateDsLedgerHeight(1, gossip_common.ChainID("A"))
	p.Gossip.UpdateTxLedgerHeight(1, gossip_common.ChainID("A"))

	needLedger := viper.GetBool("peer.lookup.needLedger")
	if mode != "lookup" || needLedger {
		acceptChan, _ := p.Gossip.Accept(gossip.AcceptData, false)
		go p.handleGossipMsg(acceptChan)

		txChan, _ := p.Gossip.Accept(gossip.AcceptTransactionData, false)
		go p.handleTxMsg(txChan)

		go p.Sync()
	}
	if mode != "lookup" {
		p.chatWithLookup()
		go p.handleConfigRpc()
		go p.registerLookup()
	}
}

func (p *PeerServer) loadKey(port string) error {
	dataDir := viper.GetString("peer.dataDir")
	wallet := viper.New()
	wallet.SetConfigFile(dataDir + "/wallet.yaml")
	wallet.ReadInConfig()
	var err error

	multibls.BLSInit()

	if !wallet.IsSet("wallet." + port) {
		p.Miner, err = mine.NewMiner(config.GetMinerType(), "")
		if err != nil {
			peerLogger.Debugf("init key pairs failed, error: %s", err.Error())
			return err
		}
		priKeyInString, _ := p.Miner.GetKeyPairs()
		wallet.Set("wallet."+port+".privateKey", priKeyInString)
		dataDir := viper.GetString("peer.dataDir")
		wallet.WriteConfigAs(dataDir + "/wallet.yaml")
		priv := &multibls.PriKey{}
		priv.SetHexString(priKeyInString)
		priv.GetPublicKey().Serialize()
		p.PublicKey = priv.GetPublicKey().Serialize()
		p.PrivateKey = priv
		p.MsgSinger.SetKey(p.PrivateKey)
		peerLogger.Debugf("pubkey is %+v", p.PublicKey)
		peerLogger.Debugf("prikey is %+v", p.PrivateKey)
	} else {
		priKeyStr := wallet.GetString("wallet." + port + ".privatekey")
		p.Miner, err = mine.NewMiner(config.GetMinerType(), priKeyStr)
		if err != nil {
			peerLogger.Errorf("load key pairs failed, error: %s", err.Error())
			return err
		}
		priv := &multibls.PriKey{}
		priv.SetHexString(priKeyStr)
		priv.GetPublicKey().Serialize()
		p.PublicKey = priv.GetPublicKey().Serialize()
		p.PrivateKey = priv
		p.MsgSinger.SetKey(p.PrivateKey)
		peerLogger.Debugf("pubkey is %+v", p.PublicKey)
		peerLogger.Debugf("prikey is %+v", p.PrivateKey)
	}
	return nil
}

func (p *PeerServer) handleConfigRpc() {
	select {
	case msg := <-p.NotifyChannel:
		defer close(p.NotifyChannel)
		p.GetCurrentRole().ProcessMsg(msg, nil)
		return
	}
}

func (p *PeerServer) DumpRoleAndState() {
	p.GetCurrentRole().DumpCurrentState()
}

func (p *PeerServer) ChangeRole(targetRole PeerRole_Type, targetState StateType) IRole {
	p.lock.Lock()
	defer p.lock.Unlock()

	role, _ := p.roleContainer.GetRole(targetRole)

	state, _ := GetStateContainer().GetState(targetState)
	role.ChangeState(targetState)

	p.currentRole = role

	peerLogger.Debugf("CurRole<%s>, CurState<%s>, cur role obj<%+v>",
		reflect.TypeOf(p.currentRole), reflect.TypeOf(state), p.currentRole)

	return role

}

func (p *PeerServer) GetPubkey() []byte {
	return p.PublicKey
}

func (p *PeerServer) GetBrokenPeers() ([]*pb.PeerEndpoint, error) {
	peers := make([]*pb.PeerEndpoint, 0, 10)

	p.peerHandlerMap.RLock()
	defer p.peerHandlerMap.RUnlock()

	for _, v := range p.initialDsAndLookupNodes {
		if _, ok := p.peerHandlerMap.m[v]; !ok {
			peers = append(peers, &pb.PeerEndpoint{Address: v})
		}
	}
	return peers, nil
}

func (p *PeerServer) chatWithLookup() {
	// start the function to ensure we are connected
	p.reconnectOnce.Do(func() {
		//go p.ensureConnected()
	})

	myAddr := config.GetListenAddress()
	peerLogger.Debugf("myaddr is %+v", myAddr)

	for _, lu := range luAddList {
		if myAddr != lu && len(lu) > 0 {
			p.initialDsAndLookupNodes = append(p.initialDsAndLookupNodes, lu)
			go p.ChatWithPeer("Lookup node", lu)
		}
	}
}

func (p *PeerServer) ChatWithDs() {
	// start the function to ensure we are connected
	p.reconnectOnce.Do(func() {
		//go p.ensureConnected()
	})

	myAddr := config.GetListenAddress()
	peerLogger.Debugf("myaddr is %+v", myAddr)

	// now dsInitList is 0 item
	for _, ds := range dsInitList {
		if myAddr != ds && len(ds) > 0 {
			peerLogger.Debugf("try to connect ds %+v", ds)
			p.initialDsAndLookupNodes = append(p.initialDsAndLookupNodes, ds)

			go p.ChatWithPeer("DS node", ds)
		}
	}
}

func (p *PeerServer) ensureConnected() {
	touchPeriod := viper.GetDuration("peer.discovery.period")
	peerLogger.Debugf("Starting Peer reconnect service (touch service), with period = %s", touchPeriod)

	tickChan := time.NewTicker(touchPeriod).C
	for {
		// Simply loop and check if need to reconnect
		<-tickChan
		// currently connected ones
		brokenPeers, err := p.GetBrokenPeers()
		if err != nil {
			peerLogger.Errorf("Error in touch service: %s", err.Error())
		}
		if len(brokenPeers) > 0 {
			peerLogger.Debugf("attempting to reconnect<%+v>", brokenPeers)
			for _, v := range brokenPeers {
				go p.ChatWithPeer("reconnecting DS or lookup node", v.Address)
			}
		}
	}
}

// for init nodes, update all nodes role after received startpow or setprimay command
func (p *PeerServer) UpdateRoleList() {
	peerLogger.Debugf("update role list...")
	myAddr := config.GetListenAddress()

	var ds []*pb.PeerEndpoint
	var lookup []*pb.PeerEndpoint

	if p.SelfNode == nil {
		addrPort := strings.Split(myAddr, ":")
		port, _ := strconv.Atoi(addrPort[1])
		newpeer := &pb.PeerEndpoint{}
		newpeer.Pubkey = p.PublicKey
		newpeer.Id = &pb.PeerID{Name: myAddr}
		newpeer.Address = addrPort[0]
		newpeer.Port = uint32(port)

		// update self node in peerserver
		p.SelfNode = newpeer
		peerLogger.Debugf("my node info is %+v", p.SelfNode)
	}

	// set lookup nodes
	for _, v := range luAddList {
		for k := range p.peerHandlerMap.m {
			if k == v && k != myAddr {
				t, ok := p.commNodeInfo.Load(k)
				if ok {
					lookup = append(lookup, t.(*pb.PeerEndpoint))
				}
				break
			}
		}
	}

	// set ds nodes
	for _, v := range dsInitList {
		if v != myAddr {
			for k := range p.peerHandlerMap.m {
				if k == v {
					t, ok := p.commNodeInfo.Load(k)
					if ok {
						ds = append(ds, t.(*pb.PeerEndpoint))
					}
					break
				}
			}
		}
		if v == myAddr {
			ds = append(ds, p.SelfNode)
		}
	}

	p.Committee = ds
	p.LookupNodes = lookup
	peerLogger.Debugf("current committee are %+v", p.Committee)
	peerLogger.Debugf("current lookup nodes are %+v", p.LookupNodes)
}

func (p *PeerServer) NewOpenchainDiscoveryHello() (*pb.Message, error) {
	helloMessage, err := p.newHelloMessage()
	if err != nil {
		return nil, fmt.Errorf("error getting new HelloMessage: %s", err)
	}

	data, err := proto.Marshal(helloMessage)
	if err != nil {
		return nil, fmt.Errorf("error marshalling HelloMessage: %s", err)
	}

	// Need to sign the Discovery Hello message
	newDiscoveryHelloMsg := &pb.Message{Type: pb.Message_Peer_Hello, Payload: data, Timestamp: pb.CreateUtcTimestamp()}
	return newDiscoveryHelloMsg, nil
}

func (p *PeerServer) Chat(stream pb.Peer_ChatServer) error {
	return p.handleChat(stream.Context(), stream, "", false)
}

func (p *PeerServer) Multicast(msg *pb.Message, peerList []*pb.PeerEndpoint) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	count := 0
	for _, peer := range peerList {

		if reflect.DeepEqual(peer.Pubkey, p.SelfNode.Pubkey) {
			peerLogger.Debugf("Multicast ignore self node: %+v", peer.Id)
			continue
		}

		peerLogger.Debugf("Multicast: %+v", peer.Id)

		if len(peer.Id.Name) > 0 {
			if _, ok := p.peerHandlerMap.m[peer.Id.Name]; ok {
				err := p.peerHandlerMap.m[peer.Id.Name].SendMessage(msg)
				if err != nil {
					count++
					peerLogger.Errorf("send message to peer %s failed, error: %s", peer.Id.Name, err.Error())
				}
			} else {
				p.lock.Lock()
				peerLogger.Debugf("node to %s connection is not ready, try to connect and send message after hello msg", peer.Id.Name)

				if msg.Type == pb.Message_DS_PoWSubmission {
					p.msg.retStr = "pow"
				} else {
					p.msg.retStr = "consensus"
				}
				p.msg.data = msg
				peerLogger.Debugf("waitgroup add one!!!!")
				p.msg.wg.Add(1)
				go p.ChatWithPeer("shardingnode", peer.Id.Name)
				p.msg.wg.Wait()
				p.msg.data = nil
				// get error msg from peer.msg
				err := p.msg.retStr
				if len(err) > 0 {
					count++
					peerLogger.Errorf(err)
				}
				p.lock.Unlock()
			}
		} else {
			count++
			peerLogger.Errorf("peer ID name is nil")
		}
	}

	if count == len(peerList) {
		return errors.New("all messages send failed")
	}

	return nil
}

func (p *PeerServer) SendMsg(msg *pb.Message, peerID string) error {
	if _, ok := p.peerHandlerMap.m[peerID]; ok {
		//peerLogger.Debugf("connection to %s is already created, waiting to send message....", peerID)
		err := p.peerHandlerMap.m[peerID].SendMessage(msg)
		if err != nil {
			return fmt.Errorf("send message to peer %s failed, error: %s", peerID, err.Error())
		}
	} else {
		return fmt.Errorf("no connection be created")
	}
	return nil
}

func (p *PeerServer) ChatWithPeer(nodeType string, address string) error {
	if len(address) == 0 {
		return fmt.Errorf("Error empty address")
	}
	peerLogger.Debugf("%s: Connecting to <%s>, <%s>", util.GId, address, nodeType)

	var conn *grpc.ClientConn
	var err error

	for i := 0; i < 10; i++ {

		conn, err = newPeerClientConnectionWithAddress(address)
		if err == nil {
			peerLogger.Debugf("Succefully connected to peer <%s>", address)
			break
		} else {
			peerLogger.Warningf("Retry <%d> to connect to peer <%s>", i+1, address)
		}
	}

	if err != nil {
		peerLogger.Errorf("error creating connection to peer address %s: %s", address, err)
		return err
	}

	serverClient := pb.NewPeerClient(conn)
	ctx := context.Background()
	stream, err := serverClient.Chat(ctx, grpc.MaxCallRecvMsgSize(rpcMessageSize), grpc.MaxCallSendMsgSize(rpcMessageSize))
	if err != nil {
		peerLogger.Errorf("error establishing chat with peer address %s: %s", address, err)
		return err
	}
	peerLogger.Debugf("Succefully established Chat with peer address: %s", address)
	err = p.handleChat(ctx, stream, address, true)
	stream.CloseSend()
	if err != nil {
		peerLogger.Errorf("ending Chat with peer address %s due to error: %s", address, err)
		return err
	}
	return nil
}

// Chat implementation of the the Chat bidi streaming RPC function
func (p *PeerServer) handleChat(ctx context.Context, stream ChatStream, address string, active bool) error {
	handler, err := newP2PHandler(p, stream, active)
	if err != nil {
		return fmt.Errorf("error creating handler during handleChat initiation: %s", err)
	}

	defer handler.Stop(active)

	for {
		in, err := stream.Recv()
		if err == io.EOF {
			peerLogger.Errorf("%s: received EOF, ending Chat", util.GId)
			return nil
		}

		if err != nil {
			peerLogger.Errorf("%s: error during Chat, stopping Chat handler: %s", util.GId, err.Error())
			return err
		}

		err = handler.HandleMessage(in)
		if err != nil {
			if !handler.Registered() {
				peerLogger.Errorf("%s: Error handling message: <%s>. Stopping chat.", util.GId, err.Error())
				return err
			}
			peerLogger.Warningf("%s: Error handling message: <%s>. No chat ending.", util.GId, err.Error())
		}
	}
}

// registerLookup register my info to lookup
func (p *PeerServer) registerLookup() {
	peerLogger.Debug("waiting for register to lookup...")
	for {
		select {
		case <-p.Registered:
			return
		case <-time.After(5 * time.Second):
			// Currently, default one lookup
			err := p.SendMsg(&pb.Message{Type: pb.Message_Peer_Register, Peer: p.SelfNode}, p.LookupNodes[0].Id.Name)
			if err != nil {
				peerLogger.Errorf("register to lookup failed:%s", err)
			}
		}
	}
}

func (p *PeerServer) GetTxPool() *txpool.TxPool {
	return txpool.GetTxPool()
}

func (p *PeerServer) Sharding() []*pb.PeerEndpoint {
	return p.ShardingNodes
}

//txBroadcastLoop broadcast transactions to other nodes.
func (p *PeerServer) txBroadcastLoop() {
	for {
		select {
		case event := <-p.txsCh:
			p.BroadcastTxs(event.Txs)
		case <-p.txsSub.Err():
			return
		}
	}
}

//BroadcatTxs multicast tx to its corresponding shard nodes
func (p *PeerServer) NetworkInfo() []*pb.PeerEndpoint {
	eplist := make([]*pb.PeerEndpoint, 0)

	for _, handler := range p.peerHandlerMap.m {
		ep, err := handler.To()
		ep.Pubkey = []byte("")
		ep.Port = 0
		if err != nil {
			continue
		}
		eplist = append(eplist, &ep)
	}
	return eplist
}

//BroadcatTxs multicast tx to its corresponding shard nodes
func (p *PeerServer) BroadcastTxs(txs []*pb.Transaction) {
	for _, tx := range txs {
		p.Gossip.Gossip(gossip.CreateTransactionMsg(tx, gossip_common.ChainID("A")))
	}
}

//CorrespondShardID addr to corresponding shardId
func CorrespondShardID(buf []byte, n uint32) uint32 {
	if n == 0 {
		return 0
	}
	addr2BigNum := new(big.Int).SetBytes(buf)
	res := addr2BigNum.Mod(addr2BigNum, new(big.Int).SetUint64(uint64(n)))
	return uint32(res.Uint64())
}

//filter filter transactions by network sharding
func (p *PeerServer) FilterTxs(txs pb.Transactions) pb.Transactions {
	var savedTxs, dropedTx pb.Transactions
	for _, tx := range txs {
		if !p.IsFiltered(tx) {
			savedTxs = append(savedTxs, tx)
		} else {
			dropedTx = append(dropedTx, tx)
		}
	}
	if len(dropedTx) != 0 {
		peerLogger.Debugf("rebroadcast droped txs, nums=%d", len(dropedTx))
		//go d.peerServer.BroadcastTxs(dropedTx)
	}
	return savedTxs
}

//filter filter transactions by network sharding
func (p *PeerServer) IsFiltered(tx *pb.Transaction) bool {
	shardsNum := p.DsBlockChain().CurrentBlock().(*pb.DSBlock).GetHeader().ShardingSum
	from, _ := p.Signer.Sender(tx)
	peerLogger.Debugf("shardingID=%d, txShouldShardId=%d, shardsNum=%d", p.ShardingID, CorrespondShardID(from[:], shardsNum), shardsNum)
	if p.ShardingID == CorrespondShardID(from[:], shardsNum) {
		return false
	}
	return true
}

//validateTx valid the tx's signature and corresponding shard
func (p *PeerServer) IsTxValidated(tx *pb.Transaction) bool {
	h := tx.Hash()
	valid, err := p.Signer.VerifyHash(h[:], tx.Signature, tx.SenderPubKey)
	if err != nil || !valid {
		return false
	}
	return !p.IsFiltered(tx)
}

// RegisterHandler register a MessageHandler with this coordinator
func (p *PeerServer) RegisterPHHandler(messageHandler IP2PHandler) error {
	key, err := getPHHandlerKey(messageHandler)
	if err != nil {
		return fmt.Errorf("error registering handler: %s", err)
	}

	p.peerHandlerMap.Lock()
	defer p.peerHandlerMap.Unlock()
	if _, ok := p.peerHandlerMap.m[key.Name]; ok {
		peerLogger.Warningf("duplicate PHHandler, key: %+v", key)
		// return fmt.Errorf("Duplicate PHHandler")
		return nil
	}

	p.peerHandlerMap.m[key.Name] = messageHandler
	peerLogger.Debugf("registered PHHandler with key: %+v", key)
	return nil
}

// DeregisterHandler deregisters an already registered MessageHandler for this coordinator
func (p *PeerServer) DeregisterPHHandler(messageHandler IP2PHandler) error {
	key, err := getPHHandlerKey(messageHandler)
	if err != nil {
		return fmt.Errorf("error deregistering handler: %s", err)
	}

	p.peerHandlerMap.Lock()
	defer p.peerHandlerMap.Unlock()
	if _, ok := p.peerHandlerMap.m[key.Name]; !ok {
		// Handler NOT found
		return fmt.Errorf("Failed to find handler with key: %s", key)
	}

	delete(p.peerHandlerMap.m, key.Name)
	peerLogger.Debugf("%s: Deregistered handler with key: %s", util.GId, key)
	return nil
}

func (p *PeerServer) newHelloMessage() (*pb.PeerEndpoint, error) {
	addr := config.GetListenAddress()

	id := &pb.PeerID{Name: addr}
	addrPort := strings.Split(addr, ":")
	port, _ := strconv.Atoi(addrPort[1])

	peerLogger.Debugf("newHelloMessage: addr: %+v", addrPort)

	roleid := viper.GetString("peer.roleid")

	return &pb.PeerEndpoint{
		Id:       id,
		Address:  addrPort[0],
		Port:     uint32(port),
		Pubkey:   p.PublicKey,
		Roleid:   roleid,
		Coinbase: p.CoinBase,
	}, nil
}

func (p *PeerServer) TxBlockChain() *blockchain.TxBlockChain {
	return p.txBlockChain
}

func (p *PeerServer) DsBlockChain() blockchain.IBasicBlockChain {
	return p.dsBlockChain
}

func (p *PeerServer) GetTxChainDB() database.Database {
	return p.txBlockChain.GetDatabase()
}

func (p *PeerServer) AccountManager() *accounts.Manager { return p.accountManager }

func (peer *PeerServer) handleGossipMsg(ch <-chan *pb.GossipMessage) {
	for {
		select {
		case msg := <-ch:
			if peer.GetCurrentRole().GetName() == "Idle" {
				continue
			}
			peerLogger.Debugf("receive_gossip_msg seq_num: %d", msg.GetDataMsg().Payload.SeqNum)

			pbMsg := &pb.Message{}
			err := proto.Unmarshal(msg.GetDataMsg().Payload.Data, pbMsg)
			if err != nil {
				peerLogger.Errorf("proto_unmarshal_error: %s", err.Error())
				continue
			}

			if pbMsg.Type == pb.Message_Node_ProcessDSBlock || pbMsg.Type == pb.Message_Node_ProcessFinalBlock {
				if peer.GetCurrentRole().GetName() != "DsLead" && peer.GetCurrentRole().GetName() != "DsBackup" {
					peer.GetCurrentRole().ProcessMsg(pbMsg, nil)
				}
			}
		}
	}
}

func (peer *PeerServer) handleTxMsg(ch <-chan *pb.GossipMessage) {
	for {
		select {
		case msg := <-ch:
			if peer.GetCurrentRole().GetName() == "Idle" || peer.GetCurrentRole().GetName() == "Lookup" {
				continue
			}
			tx := msg.GetTransactionMsg().Transaction
			peer.GetCurrentRole().ProcessTransaction(tx)
		}
	}
}

func (p *PeerServer) Sync() {
	dsBlockCh := make(chan pb.DSBlock, 1)
	txBlockCh := make(chan pb.TxBlock, 1)

	p.Gsp = state.NewGossipStateProvider("A", p.Gossip, p.dsBlockChain, p.txBlockChain, dsBlockCh, txBlockCh)

	isSync := viper.GetBool("peer.allNodeSync")
	go func() {
		for {
			select {
			case b, ok := <-dsBlockCh:
				if !ok {
					return
				}
				if isSync || p.GetCurrentRole().GetName() == "Idle" {
					peerLogger.Debugf("I get a dsblock=%d, will process it", b.NumberU64())
					data, err := proto.Marshal(&b)
					if err != nil {
						continue
					}
					p.GetCurrentRole().ProcessDSBlock(&pb.Message{Payload: data}, nil)
				} else {
					peerLogger.Debugf("I get a dsblock=%d, but will ignore it", b.NumberU64())
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case b, ok := <-txBlockCh:
				if !ok {
					return
				}
				if isSync || p.GetCurrentRole().GetName() == "Idle" {
					peerLogger.Debugf("I get a txblock=%d, will process it", b.NumberU64())
					data, err := proto.Marshal(&b)
					if err != nil {
						continue
					}
					p.GetCurrentRole().ProcessFinalBlock(&pb.Message{Payload: data}, nil)
				} else {
					peerLogger.Debugf("I get a txblock=%d, but will ignore it", b.NumberU64())
				}
			}
		}
	}()
}

func (p *PeerServer) MsgVerify(message *pb.Message, hash []byte) error {
	ret, err := p.MsgSinger.VerifyHash(hash, message.Signature, message.Peer.Pubkey)
	if err != nil {
		return err
	}

	if !ret {
		return errors.New("check signature failed")
	}
	return nil
}

func (p *PeerServer) GetPeerStatus(context.Context, *pb.EmptyRequest) (*pb.PeerStatusResponse, error) {
	psRsp := pb.PeerStatusResponse{}
	psRsp.CrrRoleName = p.GetCurrentRole().GetName()
	psRsp.CrrFSMStatus = p.GetCurrentRole().GetStateName()

	psRsp.CssRole = 0
	psRsp.CssMode = 0

	psRsp.CrrGspPeersCnt = uint32(len(p.NetworkInfo()))
	psRsp.CrrPowCntInDSHandler = uint32(p.ConsensusData.PoWSubList.Len())
	psRsp.CrrMBCntInDSHandler = uint32(p.ConsensusData.MicroBlockList.Len())
	psRsp.CrrTxHeight = uint32(p.txBlockChain.CurrentBlock().NumberU64())
	psRsp.CrrDSHeight = uint32(p.dsBlockChain.CurrentBlock().NumberU64())
	psRsp.TotalTXNum = uint32(p.totalTXNum)
	if p.txpool != nil {
		pendingTxs, queuedTxs := p.txpool.Stats()
		psRsp.TxPending = uint32(pendingTxs)
		psRsp.TxQueued = uint32(queuedTxs)
	} else {
		psRsp.TxPending, psRsp.TxQueued = 0, 0
	}

	psRsp.ShardId = uint32(p.shardingID)
	peerLogger.Debugf("GetPeerStatus: %+v", psRsp)

	return &psRsp, nil
}

func (p *PeerServer) GetCurrentRole() IRole {
	p.lock.Lock()
	defer p.lock.Unlock()

	tmp := p.currentRole
	return tmp
}

func (p *PeerServer) readConfig() {
	powDifficulty = viper.GetInt64("consensus.powDifficulty")
	wait4PoWTime = viper.GetInt64("consensus.wait4PoWTime")
	viewchangeTimeOut = viper.GetInt64("consensus.viewchangeTimeOut")
	shardingSize = viper.GetInt("sharding.size")
}

func (p *PeerServer) GetPowDifficulty() int64 {
	if !viper.GetBool("mining.difficulty.auto") {
		return powDifficulty
	}
	var MinDiff = powDifficulty
	var TxSize int64 = 400
	var MaxMicroBlockTxNum int64 = 100
	var MaxMicroBlockSize int64 = MaxMicroBlockTxNum * TxSize
	var MaxFinalBlockSize int64 = 1 << 20
	var MinShardingSum int64 = 10
	MaxNodeNum := MaxFinalBlockSize / MaxMicroBlockSize * int64(shardingSize)
	var d float64 = 0
	curDsBlock := p.DsBlockChain().CurrentBlock().(*pb.DSBlock)

	if curDsBlock.NumberU64() == 0 {
		return powDifficulty
	}
	lastDiff := int64(curDsBlock.Header.PowDifficulty)
	max := func(a, b int64) int64 {
		if a > b {
			return a
		}
		return b
	}
	if int64(curDsBlock.Header.PowSubmissionNum) > MaxNodeNum {
		return max(lastDiff+(lastDiff/2048*(int64(curDsBlock.Header.PowSubmissionNum)-MaxNodeNum)/128), MinDiff)
	}
	if int64(curDsBlock.Header.ShardingSum) <= MinShardingSum {
		return max(lastDiff+(lastDiff/2048*(int64(curDsBlock.Header.ShardingSum)-MinShardingSum)*128), MinDiff)
	}
	i := int64((curDsBlock.NumberU64()-1))*int64(shardingSize) + 1
	var j int64 = 0
	for ; j < int64(shardingSize); j++ {
		finalBlock := p.TxBlockChain().GetBlockByNumber(uint64(i + j)).(*pb.TxBlock)
		transactions := finalBlock.Transactions()
		shardingNum := curDsBlock.Header.ShardingSum
		txNums := make([]int64, shardingNum)
		for _, tx := range transactions {
			from, _ := p.Signer.Sender(tx)
			txNums[CorrespondShardID(from[:], shardingNum)]++
		}
		var t float64 = 0
		for _, v := range txNums {
			t = t + float64(v)
		}
		t = t / float64(shardingNum)
		d += t
	}
	d /= float64(shardingSize)
	d = (0.8 - d/float64(MaxMicroBlockTxNum)) * 100
	return max(lastDiff+(lastDiff/2048*int64(d)), MinDiff)
}

func (p *PeerServer) GetWait4PoWTime() int64 {
	return wait4PoWTime
}

func (p *PeerServer) GetShardingSize() int {
	return shardingSize
}

func (p *PeerServer) GetViewchangeTimeOut() int64 {
	return viewchangeTimeOut
}

func (p *PeerServer) SetDsInit(dsList []*pb.PeerEndpoint) {
	for _, v := range dsList {
		dsInitList = append(dsInitList, v.Id.Name)
	}
}

func (p *PeerServer) GetShardId(tx *pb.Transaction, dsNum uint64) uint32 {
	dsBlock := p.DsBlockChain().GetBlockByNumber(dsNum).(*pb.DSBlock)
	shardsNum := dsBlock.GetHeader().ShardingSum
	if shardsNum == 0 {
		return 0
	}
	from, _ := p.Signer.Sender(tx)
	addr2BigNum := new(big.Int).SetBytes(from[:])
	res := addr2BigNum.Mod(addr2BigNum, new(big.Int).SetUint64(uint64(shardsNum)))
	shardId := uint32(res.Uint64())
	peerLogger.Debugf("shardingID=%d, txShouldShardId=%d, shardsNum=%d", p.ShardingID, shardId, shardsNum)
	return shardId
}

func (p *PeerServer) CalStateRoot(finalBlock *pb.TxBlock) ([]byte, uint64, error) {
	bc := p.TxBlockChain()
	statecatch, err := bc.State()
	if err != nil {
		peerLogger.Errorf("get state failed, error: %s", err.Error())
		return nil, 0, err
	}
	sc := statecatch.Copy()
	_, _, gasUsed, err := bc.Processor().Process(finalBlock, sc, bc.VmConfig())
	if err != nil {
		peerLogger.Errorf("process final block failed, error: %s", err.Error())
		return nil, 0, err
	}
	root := sc.IntermediateRoot(false)
	return root.Bytes(), gasUsed, nil
}
