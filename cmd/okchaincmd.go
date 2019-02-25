// Copyright The go-okchain Authors 2018,  All rights reserved.

package okchaincmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ok-chain/okchain/config"
	"github.com/ok-chain/okchain/log"
	logging "github.com/ok-chain/okchain/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var logger = logging.MustGetLogger("main")

func Main(name string, insertCommandFunc func(*cobra.Command)) {

	var mainCmd = &cobra.Command{
		Use: name,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			peerCommand := getPeerCommandFromCobraCommand(cmd)
			log.LoggingInit(peerCommand)

			return config.CacheConfiguration()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	// For environment variables.
	config.SetDefaultViperConfig()
	viper.SetEnvPrefix(config.CmdRoot)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	// Define command-line flags that are valid for all peer commands and
	// subcommands.
	mainFlags := mainCmd.PersistentFlags()
	//mainFlags.BoolVarP(&versionFlag, "version", "v", false, "Display current version of okchain")
	//mainFlags.String("logging-level", "", "Default logging level and overrides, see okchain.yaml for full syntax")
	//mainFlags.String("config", "", "config file")
	//mainFlags.String("account.keystoreDir", "", "the keystore dir")
	//mainFlags.String("peer.dataDir", "", "default dir:$GOPATH/src/github.com/ok-chain/okchain/dev/data/")
	//mainFlags.String("peer.jsonrpcAddress", "", "jsonrpcAddress")
	//mainFlags.String("peer.listenAddress", "", "listenAddress")
	//mainFlags.String("peer.lookupNodeUrl", "", "lookupNodeUrl")
	//mainFlags.Int("peer.syncType", 0, "sync Type")
	mainFlags.String("datadir", viper.GetString("datadir"), "required, data directory for the databases and keystore")
	viper.BindPFlags(mainFlags)

	//testCoverProfile := ""
	//mainFlags.StringVarP(&testCoverProfile, "test.coverprofile", "", "coverage.cov", "Done")

	var alternativeCfgPath = os.Getenv("PEER_CFG_PATH")
	if alternativeCfgPath != "" {
		logger.Infof("User defined config file path: %s", alternativeCfgPath)
		viper.AddConfigPath(alternativeCfgPath) // Path to look for the config file in
	} else {
		viper.AddConfigPath("./") // Path to look for the config file in
		// Path to look for the config file in based on GOPATH
		gopath := os.Getenv("GOPATH")
		for _, p := range filepath.SplitList(gopath) {
			peerpath := filepath.Join(p, "src/github.com/ok-chain/okchain/config")
			viper.AddConfigPath(peerpath)
		}
	}

	// Now set the configuration file.
	viper.SetConfigName(config.CfgFileName) // Name of config file (without extension)

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		if name != "okchaincli" {
			panic(fmt.Errorf("fatal error when reading %s config file: %s\n", config.CfgFileName, err))
		}
	}

	insertCommandFunc(mainCmd)

	if mainCmd.Execute() != nil {
		os.Exit(1)
	}

}

// getPeerCommandFromCobraCommand retreives the peer command from the cobra command struct.
// i.e. for a command of `peer node start`, this should return "node"
// For the main/root command this will return the root name (i.e. peer)
// For invalid commands (i.e. nil commands) this will return an empty string
func getPeerCommandFromCobraCommand(command *cobra.Command) string {
	var commandName string
	if command == nil {
		return commandName
	}

	if peerCommand, ok := findChildOfRootCommand(command); ok {
		commandName = peerCommand.Name()
	} else {

		commandName = command.Name()
	}

	return commandName
}

func findChildOfRootCommand(command *cobra.Command) (*cobra.Command, bool) {
	for command.HasParent() {
		if !command.Parent().HasParent() {
			return command, true
		}

		command = command.Parent()
	}

	return nil, false
}
