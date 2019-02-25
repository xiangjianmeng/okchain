// Copyright IBM Corp. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package util

import (
	"sync"

	logging "github.com/ok-chain/okchain/log"
)

// Module names for logger initialization.
const (
	LoggingChannelModule   = "gossip/channel"
	LoggingCommModule      = "gossip/comm"
	LoggingDiscoveryModule = "gossip/discovery"
	LoggingGossipModule    = "gossip/gossip"
	LoggingPullModule      = "gossip/pull"
	//LoggingServiceModule   = "gossip/service"
	LoggingStateModule = "gossip/state"
	//LoggingPrivModule      = "gossip/privdata"
)

var LoggersByModules = make(map[string]*logging.Logger)
var lock = sync.Mutex{}
var testMode bool = true

// defaultTestSpec is the default logging level for gossip tests
var defaultTestSpec = "WARNING"

// GetLogger returns a logger for given gossip module and peerID
func GetLogger(module string, peerID string) *logging.Logger {
	if peerID != "" && testMode {
		module = module + "#" + peerID
	}

	lock.Lock()
	defer lock.Unlock()

	if lgr, ok := LoggersByModules[module]; ok {
		return lgr
	}

	// Logger doesn't exist, create a new one
	lgr := logging.MustGetLogger(module)
	LoggersByModules[module] = lgr
	return lgr
}

// SetupTestLogging sets the default log levels for gossip unit tests
func SetupTestLogging() {
	testMode = true
	logging.InitFromSpec(defaultTestSpec)
}
