// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"bytes"

	"github.com/c-bata/go-prompt"
)

const (
	//wallet flags
	walletFlag     = "wallet"
	newAccountFlag = "newAccount"
	unlockFlag     = "unlock"
	lockFlag       = "lock"
	accountsFlag   = "listAccounts"

	//okc flags
	okcFlag                   = "okc"
	getAccountFlag            = "getAccount"
	sendTxFalg                = "sendTransaction"
	getTxFlag                 = "getTransaction"
	getLatestTxBlockFlag      = "getLatestTxBlock"
	getLatestDsBlockFlag      = "getLatestDsBlock"
	deployContractFlag        = "deployContract"
	getTransactionReceiptFlag = "getTransactionReceipt"
	contractAtFlag            = "contractAt"
)

func optionCompleter(args []string, long bool) []prompt.Suggest {
	// l := len(args)
	// if l <= 1 {
	// 	if long {
	// 		return prompt.FilterHasSuffix(optionHelp, ".", false)
	// 	}
	// 	return optionHelp
	// }

	var suggests []prompt.Suggest
	commandArgs := excludeOptions(args)
	//switch commandArgs[0] {
	//case walletFlag:
	//	suggests = walletSub
	//case okcFlag:
	//	suggests = okcSub
	//default:
	//	suggests = optionHelp
	//}
	if v, ok := promptSuggests[commandArgs[0]]; ok {
		suggests = v
	} else {
		suggests = optionHelp
	}
	return suggests
}

var optionHelp = []prompt.Suggest{
	{Text: "-h"},
	{Text: "--help"},
}
var okcSub = []prompt.Suggest{
	{Text: CmdOptionsFlag(okcFlag, getAccountFlag, []string{"address"}), Description: "get the account balance of the specified address"},
	{Text: CmdOptionsFlag(okcFlag, sendTxFalg, []string{"from:0x94dc66c8be8393e41791dfd7b8a47fe43f3e0890", "to:0xD8104E3E6dE811DD0cc07d32cCcE2f4f4B38403a", "gas:1000000", "gasPrice:1", "value:100000", "nonce:0", "passwd:okchain"}), Description: "SendTransaction, gas, gasPrice, nonce"},
	{Text: CmdOptionsFlag(okcFlag, getTxFlag, []string{"tx_hash"}), Description: "Get transaction by tx's hash"},
	{Text: CmdOptionsFlag(okcFlag, getLatestTxBlockFlag, nil), Description: "Get latest Tx Block"},
	{Text: CmdOptionsFlag(okcFlag, getLatestDsBlockFlag, nil), Description: "Get latest Ds Block"},
	{Text: CmdOptionsFlag(okcFlag, deployContractFlag, []string{"from:0x94dc66c8be8393e41791dfd7b8a47fe43f3e0890", "gas:1000000", "gasPrice:1", "value:0", "nonce:0", "passwd:okchain", "path:./test", "contractName:test", "initargs:"}), Description: "Deploy a contract, gas, gasPrice, value, nonce are optional"},
	{Text: CmdOptionsFlag(okcFlag, getTransactionReceiptFlag, []string{"tx_hash"}), Description: "get transaction receipt by tx's hash"},
	{Text: CmdOptionsFlag(okcFlag, contractAtFlag, []string{"address:0x84eaab7ecc07c333123a9a51976d596496d9e2a0", "contractName:test", "path:./test"}), Description: "find deployed contract at address"},
}

var flagGlobal = []prompt.Suggest{
	{Text: "--alsologtostderr", Description: "log to standard error as well as files"},
	{Text: "--certificate-authority", Description: "Path to a cert. file for the certificate authority."},
}

var walletSub = []prompt.Suggest{
	{Text: CmdOptionsFlag(walletFlag, newAccountFlag, []string{"okchain"}), Description: "Generate a new account using wallet."},
	{Text: CmdOptionsFlag(walletFlag, unlockFlag, []string{"okchain"}), Description: "Unlock an account before send transaction."},
	{Text: CmdOptionsFlag(walletFlag, lockFlag, []string{}), Description: "Unlock an account before send transaction."},
	{Text: CmdOptionsFlag(walletFlag, accountsFlag, []string{}), Description: "List all local accounts"},
}

var promptSuggests = map[string][]prompt.Suggest{
	okcFlag:    okcSub,
	walletFlag: walletSub,
}

//CmdOptionsFlag utils for prompt, efficient
func CmdOptionsFlag(cmd, subcmd string, options []string) string {
	if options == nil {
		options = []string{}
	}
	var buf bytes.Buffer
	buf.WriteString(cmd)
	buf.WriteString(".")
	buf.WriteString(subcmd)
	buf.WriteString("(")
	for i, opt := range options {
		buf.WriteString(opt)
		if i < len(options)-1 {
			buf.WriteString(",")
		}
	}
	buf.WriteString(")")
	return buf.String()
}

func CmdAttrFlag(cmd, subcmd string) string {
	var buf bytes.Buffer
	buf.WriteString(cmd)
	buf.WriteString(".")
	buf.WriteString(subcmd)
	return buf.String()
}
