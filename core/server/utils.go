// Copyright The go-okchain Authors 2018,  All rights reserved.

package server

import (
	"fmt"
	"net"
	"sync"

	pb "github.com/ok-chain/okchain/protos"
	"google.golang.org/grpc"
)

func newPeerClientConnectionWithAddress(peerAddress string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	opts = append(opts, grpc.WithTimeout(defaultTimeout))
	opts = append(opts, grpc.WithBlock())

	conn, err := grpc.Dial(peerAddress, opts...)
	if err != nil {
		return nil, err
	}
	return conn, err
}

// GetLocalIP returns the non loopback local IP of the host
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback then display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

type remotePeerManager struct {
	sync.RWMutex
	m map[string]IP2PHandler
}

type MsgChannel struct {
	data   *pb.Message
	wg     sync.WaitGroup
	retStr string
}

type ConsensusData struct {
	CurrentDSBlock    *pb.DSBlock
	CurrentFinalBlock *pb.TxBlock
	CurrentMicroBlock *pb.MicroBlock
	CurrentVCBlock    *pb.VCBlock
	PoWSubList        pb.SyncPowSubmissions
	MicroBlockList    pb.SyncMicroBlockSubmissions
	CurrentStage      pb.ConsensusType
	LastStage         pb.ConsensusType
	//PubKeyToCoinBaseMap map[string][]byte
}

//createTxMsg create the pb.Message of a transaction
//func createTxMsg(tx *pb.Transaction) *pb.Message {
//	message := &pb.Message{Type: pb.Message_Node_BroadcastTransactions}
//	message.Timestamp = pb.CreateUtcTimestamp()
//
//	data, _ := proto.Marshal(tx)
//	message.Payload = data
//	message.Signature = []byte("xxx")
//	return message
//}

//getTargetNodes get the sharding nodes which the transaction should be processed
//func getTargetNodes(p *PeerServer, tx *pb.Transaction) []*pb.PeerEndpoint {
//	shardsNum := p.DsBlockChain().CurrentBlock().GetHeader().ShardingSum
//	from, _ := p.signer.Sender(tx)
//	toShardId := CorrespondShardID(from[:], shardsNum)
//
//	peers := p.ShardingMap[toShardId]
//	peerLogger.Debugf("broadcast info: from=%s, toShardId=%d, sharding_PeerPubkey[%d]=%+v", common.Bytes2Hex(from[:]), toShardId, len(peers), peers)
//	return peers
//}

//CorrespondShardID addr to corresponding shardId
//func CorrespondShardID(buf []byte, n uint32) uint32 {
//	if n == 0 {
//		return 0
//	}
//	addr2BigNum := new(big.Int).SetBytes(buf)
//	res := addr2BigNum.Mod(addr2BigNum, new(big.Int).SetUint64(uint64(n)))
//	return uint32(res.Uint64())
//}

func getPHHandlerKey(peerMessageHandler IP2PHandler) (*pb.PeerID, error) {
	peerEndpoint, err := peerMessageHandler.To()
	if err != nil {
		return &pb.PeerID{}, fmt.Errorf("error getting messageHandler key: %s", err)
	}

	id := &pb.PeerID{Name: peerEndpoint.Id.Name}
	return id, nil
}
