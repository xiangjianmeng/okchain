// Copyright The go-okchain Authors 2018,  All rights reserved.

package protos

import (
	"math/big"
	"time"
)

func CreateUtcTimestamp() *Timestamp {
	now := time.Now().UTC()
	return &(Timestamp{Second: uint64(now.Unix()),
		Nanosecond: uint64(now.Nanosecond())})
}

func Timestamp2Time(protoTS *Timestamp) *time.Time {
	t := time.Unix(int64(protoTS.Second), int64(protoTS.Nanosecond))
	return &t
}

func Uint64ToBigInt(u uint64) *big.Int {
	return new(big.Int).SetUint64(u)
}

func (b *DSBlock) Dump() {

	logger.Infof("==========DSBlock Dump============")
	defer logger.Infof("========DSBlock Dump End=========")

	logger.Infof("NewLeader: %s", b.Header.NewLeader.Id)
	logger.Infof("BlockNumber: %d", b.Header.BlockNumber)
	logger.Infof("ShardingSum: %d", b.Header.ShardingSum)
	logger.Infof("WinnerPubKey: %x", b.Header.WinnerPubKey)
	logger.Infof("ShardingNodes:")
	PeerEndpointList(b.Body.ShardingNodes).Dump()

}
