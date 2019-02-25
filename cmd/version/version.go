// Copyright The go-okchain Authors 2018,  All rights reserved.

package version

import (
	"fmt"

	"github.com/ok-chain/okchain/config"
	"github.com/spf13/cobra"
)

func GetVersionCmd() *cobra.Command {
	// Set the flags on the node start command.
	return versionCmd
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version number",
	Long:  `Display version number`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printVersion(nil)
	},
}

func printVersion(postrun func() error) error {
	//fmt.Println(strings.Title(clientIdentifier))
	fmt.Println("OkChain Version:", config.GetVersion())
	//if gitCommit != "" {
	//	fmt.Println("Git Commit:", gitCommit)
	//}
	//fmt.Println("Architecture:", runtime.GOARCH)
	//fmt.Println("Protocol Versions:", eth.ProtocolVersions)
	//fmt.Println("Network Id:", eth.DefaultConfig.NetworkId)
	//fmt.Println("Go Version:", runtime.Version())
	//fmt.Println("Operating System:", runtime.GOOS)
	//fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	//fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	return nil
}
