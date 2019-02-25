// Copyright The go-okchain Authors 2018,  All rights reserved.

package main

import (
	"github.com/ok-chain/okchain/cmd"
	"github.com/ok-chain/okchain/cmd/account"
	"github.com/ok-chain/okchain/cmd/console"
	"github.com/ok-chain/okchain/cmd/license"
	"github.com/ok-chain/okchain/cmd/transaction"
	"github.com/ok-chain/okchain/cmd/version"
	"github.com/spf13/cobra"
)

func main() {

	okchaincmd.Main("okchaincli", insertClientCommand)
}

func insertClientCommand(mainCmd *cobra.Command) {

	mainCmd.AddCommand(account.GetAccountCmd())
	mainCmd.AddCommand(console.GetConsoleCmd())
	mainCmd.AddCommand(license.GetLicenseCmd())
	mainCmd.AddCommand(version.GetVersionCmd())
	mainCmd.AddCommand(transaction.GetTransactionCmd())
}
