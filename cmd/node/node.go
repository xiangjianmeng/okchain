// Copyright The go-okchain Authors 2018,  All rights reserved.

package node

import (
	"fmt"
)

import (
	logging "github.com/ok-chain/okchain/log"
	"github.com/spf13/cobra"
)

const nodeFuncName = "node"

var (
	stopPidFile string
)
var logger = logging.MustGetLogger("nodeCmd")

// Cmd returns the cobra command for Node
func Cmd() *cobra.Command {
	nodeCmd.AddCommand(startCmd())
	return nodeCmd
}

var nodeCmd = &cobra.Command{
	Use:   nodeFuncName,
	Short: fmt.Sprintf("Start the Okchain daemon"),
	Long:  fmt.Sprintf("Start the Okchain daemon"),
}
