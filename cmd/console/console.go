// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"fmt"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	version string = "1.0-beta"
)

func GetConsoleCmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := consoleStartCmd.Flags()
	flags.String("url", "http://localhost:16001", "required, an avaliable okchain node address")
	//flags.Bool("istestnet", false, "is testnet")
	//flags.String("jsonrpcport", "16066", "jsonrpc port")
	viper.BindPFlags(flags)
	return consoleStartCmd
}

var consoleStartCmd = &cobra.Command{
	Use:   "console",
	Short: "Start an interactive environment",
	Long:  `Start an interactive environment`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return StartConsole(nil)
	},
}

func StartConsole(postrun func() error) error {
	fmt.Printf("okchain-prompt command line. Version-%s\n", version)
	fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")
	defer fmt.Println("Bye!")
	p := prompt.New(
		Executor,
		Completer,
		prompt.OptionTitle("okchain-prompt: interactive okchain client"),
		prompt.OptionPrefix(">>> "),
		prompt.OptionInputTextColor(prompt.Yellow),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	)
	go GetTxReceiptLoop()
	p.Run()
	return nil
}
