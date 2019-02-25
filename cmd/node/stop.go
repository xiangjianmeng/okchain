// Copyright The go-okchain Authors 2018,  All rights reserved.

package node

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func stopCmd() *cobra.Command {
	nodeStopCmd.Flags().StringVar(&stopPidFile, "stop-peer-pid-file",
		viper.GetString("peer.fileSystemPath"),
		"Location of peer pid local file, for forces kill")

	return nodeStopCmd
}

var nodeStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the running node.",
	Long:  `Stops the running node, disconnecting from the network.`,
	Run: func(cmd *cobra.Command, args []string) {
		stop()
	},
}

func stop() (err error) {
	logger.Info("stopping peer using grpc")
	return nil
}
