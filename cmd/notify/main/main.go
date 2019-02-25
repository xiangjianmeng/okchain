// Copyright The go-okchain Authors 2018,  All rights reserved.

package main

import (
	"github.com/ok-chain/okchain/cmd"
	"github.com/ok-chain/okchain/cmd/notify"
	"github.com/spf13/cobra"
)

func main() {
	okchaincmd.Main("notify", insertClientCommand)
}

func insertClientCommand(mainCmd *cobra.Command) {

	mainCmd.AddCommand(notify.GetCmd())
}
