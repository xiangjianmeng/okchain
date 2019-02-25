// Copyright The go-okchain Authors 2018,  All rights reserved.

package main

import (
	"github.com/ok-chain/okchain/cmd"
	"github.com/ok-chain/okchain/cmd/node"
	"github.com/spf13/cobra"
)

func main() {
	okchaincmd.Main("okchaind", insertDaemonCommand)
}

func insertDaemonCommand(mainCmd *cobra.Command) {
	mainCmd.AddCommand(node.Cmd())
}
