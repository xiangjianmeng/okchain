// Copyright The go-okchain Authors 2018,  All rights reserved.

package notify

import (
	"fmt"

	"github.com/spf13/cobra"
)

const cmdName = "notify"

func GetCmd() *cobra.Command {
	initCmd.AddCommand(startPoWCmd())
	initCmd.AddCommand(setPrimaryCmd())
	initCmd.AddCommand(getRevertCmd())
	return initCmd
}

var initCmd = &cobra.Command{
	Use:   cmdName,
	Short: fmt.Sprintf("Notify daemon to start pow or set Ds Lead"),
	Long:  fmt.Sprintf("Notify daemon to start pow or set Ds Lead"),
}
