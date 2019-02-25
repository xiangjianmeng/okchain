// Copyright The go-okchain Authors 2018,  All rights reserved.

package transaction

import (
	"fmt"

	server "github.com/ok-chain/okchain/api/jsonrpc_server"
	"github.com/spf13/cobra"

	"errors"
	"io/ioutil"

	"github.com/ok-chain/okchain/cmd/account"
	"github.com/ok-chain/okchain/cmd/console"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/rlp"
	"github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/rpc"
)

const transactionName = "transaction"

//var accountcmdlogger = logging.MustGetLogger("accountcmd")

// Cmd returns the cobra command for Node
func GetTransactionCmd() *cobra.Command {
	transactionCmd.AddCommand(buildsubmitTransactionCmd())
	transactionCmd.AddCommand(buildquerytransactionCmd())
	return transactionCmd
}

var transactionCmd = &cobra.Command{
	Use:   transactionName,
	Short: fmt.Sprintf("Manage transactions"),
	Long:  fmt.Sprintf("Manage transactions"),
}

func buildsubmitTransactionCmd() *cobra.Command {
	flags := submitTransactionCmd.Flags()
	flags.String("url", "http://localhost:16001", "required, an avaliable okchain node address")
	flags.String("signedtx", "", "required, the file path that stores a signed transaction hex")
	return submitTransactionCmd
}

var submitTransactionCmd = &cobra.Command{
	Use:   "submit",
	Short: "submit a signed transaction to the network",
	Long:  "submit a signed transaction to the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return account.InvalidInputArgs(args)
		}
		return submittransaction(cmd)
	},
}

func submittransaction(cmd *cobra.Command) error {
	flags := cmd.Flags()
	nodeaddr, err := flags.GetString("url")
	if err != nil {
		return err
	}
	file, err := flags.GetString("signedtx")
	if err != nil {
		return err
	}
	if file == "" {
		return errors.New("expected flag: signedtx")
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	trans := &protos.Transaction{}
	err = rlp.DecodeBytes(common.FromHex(string(data)), trans)
	if err != nil {
		return err
	}
	client, err := rpc.DialHTTP(nodeaddr)
	if err != nil {
		return err
	}
	type submit_output struct {
		TransactionId string
	}
	var resp common.Hash
	if err := client.Call(&resp, "okchain_sendRawTransaction", trans); err != nil {
		return err
	}
	output := submit_output{
		TransactionId: resp.String(),
	}
	console.PrettyPrint(output)
	return nil
}

func buildquerytransactionCmd() *cobra.Command {
	flags := querytransactionCmd.Flags()
	flags.String("txid", "", "required, transaction id")
	flags.String("url", "http://localhost:16001", "required, an avaliable okchain node address")
	return querytransactionCmd
}

var querytransactionCmd = &cobra.Command{
	Use:   "query",
	Short: "display the specified transaction infomation",
	Long:  "display the specified transaction infomation",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return account.InvalidInputArgs(args)
		}
		return querytransaction(cmd)
	},
}

func querytransaction(cmd *cobra.Command) error {
	flags := cmd.Flags()
	nodeaddr, err := flags.GetString("url")
	if err != nil {
		return err
	}
	hash, err := flags.GetString("txid")
	if err != nil {
		return err
	}
	if hash == "" {
		return errors.New("expected flag: txid")
	}
	client, err := rpc.DialHTTP(nodeaddr)
	if err != nil {
		return err
	}
	var resp *server.RPCTransaction
	if err := client.Call(&resp, "okchain_getTransactionByHash", hash); err != nil {
		return err
	}
	console.PrettyPrint(resp)
	return nil
}
