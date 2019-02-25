// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"fmt"
	"math/big"

	"net"
	"os"
	"strings"
	"time"

	"github.com/ok-chain/okchain/common"
	logging "github.com/ok-chain/okchain/log"
	"github.com/spf13/viper"
)

const CmdRoot = "okchain"
const CfgFileName = "okchain"

var coreLogger = logging.MustGetLogger("core")

var configurationCached = false
var ipaddress string
var securityEnabled bool
var defaultMinerType = "pow"

var (
	TIMEOUT_POW_SUBMISSION = 5 * time.Second
)
var (
	MainnetGenesisHash = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3") // Mainnet genesis hash to enforce below configs on
	TestnetGenesisHash = common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d") // Testnet genesis hash to enforce below configs on

	MainnetDSGenesisHash = common.HexToHash("0x77348c839b385703bebb9f2e5d01e6c3f018339037459a7ac9fc7904e2c6bb96")
)

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	DposChainConfig = &ChainConfig{
		ChainId: big.NewInt(5),
		Dpos:    &DposConfig{},
	}

	MainnetChainConfig = &ChainConfig{
		ChainId: big.NewInt(5),
		Dpos:    &DposConfig{},
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	TestnetChainConfig = &ChainConfig{
		ChainId: big.NewInt(5),
		Dpos:    &DposConfig{},
	}

	// RinkebyChainConfig contains the chain parameters to run a node on the Rinkeby test network.
	RinkebyChainConfig = &ChainConfig{
		ChainId: big.NewInt(4),
		Clique: &CliqueConfig{
			Period: 15,
			Epoch:  30000,
		},
		Dpos: &DposConfig{},
	}

	// FLT. 2018/09/19
	// MainnetDSChainConfig is the chain parameters to run a node on the main network.
	MainnetDSChainConfig = &ChainConfig{
		ChainId: big.NewInt(5),
		DsPBFT:  &DSPBFTConfig{},
	}

	MainnetFBChainConfig = &ChainConfig{
		ChainId: big.NewInt(5),
		TxPBFT:  &FinalBlockPBFTConfig{},
	}

	// TestnetChainConfig contains the chain parameters to run a node on the Ropsten test network.
	TestnetDSChainConfig = &ChainConfig{
		ChainId: big.NewInt(10),
		DsPBFT:  &DSPBFTConfig{},
	}

	TestnetFBChainConfig = &ChainConfig{
		ChainId: big.NewInt(10),
		TxPBFT:  &FinalBlockPBFTConfig{},
	}
	//

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllEthashProtocolChanges = &ChainConfig{big.NewInt(1337), new(EthashConfig), nil, nil, nil, nil}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	//
	// This configuration is intentionally not using keyed fields to force anyone
	// adding flags to the config to also have to set these fields.
	AllCliqueProtocolChanges = &ChainConfig{big.NewInt(1337), nil, &CliqueConfig{Period: 0, Epoch: 30000}, nil, nil, nil}

	TestChainConfig = &ChainConfig{big.NewInt(1), new(EthashConfig), nil, nil, nil, nil}
	TestRules       = TestChainConfig.Rules(new(big.Int))
)

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainId *big.Int `json:"chainId"` // Chain id identifies the current chain and is used for replay protection

	// Various consensus engines
	Ethash *EthashConfig `json:"ethash,omitempty"`
	Clique *CliqueConfig `json:"clique,omitempty"`

	Dpos *DposConfig `json:"dpos,omitempty"`

	// FLT. 2018/09/19. Add Config.
	DsPBFT *DSPBFTConfig         `json:"dspbft,omitempty"`
	TxPBFT *FinalBlockPBFTConfig `json:"txpbft,omitempty"`
}

// DposConfig is the consensus engine configs for delegated proof-of-stake based sealing.
type DposConfig struct {
	Validators []common.Address `json:"validators"` // Genesis validator list
}

// String implements the stringer interface, returning the consensus engine details.
func (d *DposConfig) String() string {
	return "dpos"
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// FLT. 2018/09/19. Add Config for DS BlockChain & FinalBlockChain

type DSPBFTConfig struct{}

func (c *DSPBFTConfig) String() string {
	return "DSPBFT"
}

type FinalBlockPBFTConfig struct{}

func (c *FinalBlockPBFTConfig) String() string {
	return "FinalBlockPBFT"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	case c.Dpos != nil:
		engine = c.Dpos
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v  Engine: %v}",
		c.ChainId,
		engine,
	)
}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	return GasTableHomestead
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntatic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainId *big.Int
	// IsHomestead bool
	// IsByzantium bool
}

func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainId := c.ChainId
	if chainId == nil {
		chainId = new(big.Int)
	}
	return Rules{ChainId: new(big.Int).Set(chainId)}
}

var gopath string
var home string
var top_dir string
var script_dir string
var data_dir string
var node_dir string
var keystore_dir string
var version string

func SetDefaultViperConfig() error {
	ip := getLocalIpAddr()
	gopath = os.Getenv("GOPATH")
	home = os.Getenv("HOME")
	top_dir = gopath + "/src/github.com/ok-chain/okchain"
	script_dir = top_dir + "/dev"
	data_dir = script_dir + "/data"
	node_dir = data_dir + "/node"
	keystore_dir = script_dir + "/keystore"
	jsonrpc_port := "16066"
	viper.SetDefault("account.keystoreDir", keystore_dir)

	viper.SetDefault("peer.topDir", top_dir)
	viper.SetDefault("peer.dataDir", node_dir)
	viper.SetDefault("peer.logpath", node_dir+"/node.log")
	viper.SetDefault("peer.mode", "new")
	viper.SetDefault("peer.lookupnodeurl", ip+":14005")
	viper.SetDefault("peer.listenAddress", ip+":15066")
	viper.SetDefault("peer.dslistenaddress", "")
	viper.SetDefault("peer.jsonrpcaddress", ip+":"+jsonrpc_port)
	viper.SetDefault("peer.localip", ip)
	viper.SetDefault("peer.roleid", "node")
	viper.SetDefault("peer.syncType", 0)
	viper.SetDefault("peer.ipcendpointdir", data_dir+"/jsonrpc_ipc_endpoint")
	viper.SetDefault("peer.ipcendpoint", data_dir+"/jsonrpc_ipc_endpoint/"+jsonrpc_port+".ipc")

	viper.SetDefault("ledger.baseDir", node_dir+"/db/")
	viper.SetDefault("ledger.txblockchain.genesisConf", script_dir+"/genesis.json")

	viper.SetDefault("logging.node", "info:ledger=debug:gossip=debug:peer=debug:txpool=debug:VM=debug:CORE=debug:rpc=debug")

	viper.SetDefault("top", top_dir)
	viper.SetDefault("path", "github.com/ok-chain/okchain")
	viper.SetDefault("dev.script.top", script_dir)
	viper.SetDefault("peer.nat", "none")
	viper.SetDefault("datadir", home+"/.okchain")
	viper.SetDefault("keystorepassword", "okchain")

	return nil
}

// CacheConfiguration caches configuration settings so that reading the yaml
// file can be avoided on future requests
func CacheConfiguration() error {
	securityEnabled = viper.GetBool("security.enabled")
	version = "0.1.0"
	ipaddress = getLocalIpAddr()
	configurationCached = true

	mingerType := viper.GetString("peer.minerType")
	if len(mingerType) > 0 {
		defaultMinerType = mingerType
	}

	return nil
}

func GetVersion() string {
	if !configurationCached {
		cacheConfiguration()
	}
	return version
}

func cacheConfiguration() {
	if err := CacheConfiguration(); err != nil {
		coreLogger.Errorf("Execution continues after CacheConfiguration() failure : %s", err)
	}
}

// SecurityEnabled returns true if security is enabled
func SecurityEnabled() bool {
	if !configurationCached {
		cacheConfiguration()
	}
	return securityEnabled
}

func GetMinerType() string {
	return defaultMinerType
}

// SecurityEnabled returns true if security is enabled
func GetListenAddress() string {
	if !configurationCached {
		cacheConfiguration()
	}

	listenAddr := viper.GetString("peer.listenAddress")
	port := strings.Split(listenAddr, ":")[len(strings.Split(listenAddr, ":"))-1]
	return ipaddress + ":" + port
}

func GetListenPort() string {
	if !configurationCached {
		cacheConfiguration()
	}

	listenAddr := viper.GetString("peer.listenAddress")
	port := strings.Split(listenAddr, ":")[len(strings.Split(listenAddr, ":"))-1]
	return port
}

func GetKeyStoreDir() string {
	if !configurationCached {
		cacheConfiguration()
	}
	return keystore_dir
}

func GetDataDir() string {
	if !configurationCached {
		cacheConfiguration()
	}
	return data_dir
}

func getLocalIpAddr() string {

	addrs, err := net.InterfaceAddrs()

	ipaddr := ""
	if err != nil {
		fmt.Println(err)
		ipaddr = "127.0.0.1"
	} else {

		segment := viper.GetString("peer.networkSegment")
		for _, address := range addrs {

			if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {

				addr := ipnet.IP.String()

				if ipnet.IP.To4() != nil && strings.HasPrefix(addr, segment) {
					ipaddr = addr
				}
			}
		}
	}

	expectedListenAddr := viper.GetString("peer.localip")
	if len(expectedListenAddr) > 0 {
		ipaddr = expectedListenAddr
	}

	//coreLogger.Infof("Local Ip Address: %s\n", ipaddr)
	return ipaddr
}
