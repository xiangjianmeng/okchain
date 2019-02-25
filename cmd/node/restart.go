// Copyright The go-okchain Authors 2019,  All rights reserved.

package node

import (
	"fmt"

	ps "github.com/ok-chain/okchain/core/server"
	"github.com/ok-chain/okchain/core/server/role"
)

func revertPeerServerState(srv *ps.PeerServer) {
	//fmt.Println("revertPubKeyToCoinBaseMap")
	//revertPubKeyToCoinBaseMap(srv)
	fmt.Println("RevertRole")
	role.RevertRole(srv)
}

//func revertPubKeyToCoinBaseMap(srv *ps.PeerServer){
//	curNumber:=srv.DsBlockChain().CurrentBlock().NumberU64()
//	var i uint64
//	for i=0;i<=curNumber;i++{
//
//		dsBlock:=srv.DsBlockChain().GetBlockByNumber(i).(*protos.DSBlock)
//		fmt.Printf("dsblock is %+v", dsBlock)
//		fmt.Println(dsBlock.Body.PubKeyToCoinBaseMap)
//		for k, v := range dsBlock.Body.PubKeyToCoinBaseMap {
//			srv.ConsensusData.PubKeyToCoinBaseMap[k] = v
//		}
//	}
//	fmt.Println("PubKeyToCoinBaseMap",srv.ConsensusData.PubKeyToCoinBaseMap)
//}
