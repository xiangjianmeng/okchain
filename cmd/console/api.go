// Copyright The go-okchain Authors 2018,  All rights reserved.

package console

import (
	"errors"
	"fmt"
	"strconv"

	"encoding/hex"
	"io/ioutil"
	"math/big"
	"strings"

	"reflect"
	"time"

	"bytes"

	"github.com/c-bata/go-prompt"
	"github.com/ok-chain/okchain/accounts/abi"
	server "github.com/ok-chain/okchain/api/jsonrpc_server"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/common/hexutil"
	"github.com/ok-chain/okchain/protos"
	"github.com/op/go-logging"
)

var consoleLogger = logging.MustGetLogger("consoleLogger")
var UnknowCmd = errors.New("ERR: unknown request")
var contractAbis = map[string]abi.ABI{}
var contractAddrs = map[string]common.Address{}
var contractTxHash = map[string]common.Hash{}
var chanTxReceipt = make(chan ContractInfo, 100)

// the Keccak256 of nil
const emptyHash = string("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")

type ContractInfo struct {
	// hash of Tx deploying the contract
	Hash common.Hash
	// empty name indicates Tx for transfering between external accounts
	Name string
	// empty name indicates Tx for transfering between external accounts
	Abi abi.ABI
}

type request func(options []string) (interface{}, error)

var cmdFunction = map[string]request{
	walletFlag + newAccountFlag: NewAccount,
	walletFlag + accountsFlag:   ListAccounts,
	walletFlag + unlockFlag:     UnlockAccount,
	walletFlag + lockFlag:       LockAccount,

	okcFlag + getAccountFlag:            GetAccount,
	okcFlag + sendTxFalg:                SendTx,
	okcFlag + getLatestDsBlockFlag:      GetLatestDsBlock,
	okcFlag + getLatestTxBlockFlag:      GetLatestTxBlock,
	okcFlag + getTxFlag:                 GetTransaction,
	okcFlag + deployContractFlag:        DeployContract,
	okcFlag + getTransactionReceiptFlag: GetTransactionReceipt,
	okcFlag + contractAtFlag:            ContractAt,
}

// postRequest call function according to cmd and subcmd.
func postRequest(cmd, subcmd string, options []string) (interface{}, error) {
	consoleLogger.Debug("postRequest:", cmd, subcmd, options)
	var resp interface{}
	var err error
	client = getClient()
	if _, ok := contractAbis[cmd]; ok {
		resp, err = ContractCall(cmd, subcmd, options)
	} else if cmdFunction[cmd+subcmd] != nil {
		resp, err = cmdFunction[cmd+subcmd](options)
	} else {
		resp = UnknowCmd.Error()
	}
	return resp, err
}

// attributeCall gets attribute identified by subcmd of the contract name of cmd.
// TODO:code getting other attributes of contract can be added here.
func attributeCall(cmd, subcmd string) (interface{}, error) {
	if _, ok := contractAbis[cmd]; ok {
		if subcmd == "address" {
			return hex.EncodeToString(contractAddrs[cmd].Bytes()), nil
		}
	}
	return nil, fmt.Errorf("%s has no attribute of %s", cmd, subcmd)
}

// SendTx performs a JSON-RPC call which transfers between external accounts with
// empty data or execute contract code with code data.
// Console will receive a notification implying if tx is included by final block
// or not after while.
func SendTx(options []string) (interface{}, error) {
	args := &server.SendTxArgs{}
	var passwd string
	if !checkSubstrings(options, "from", "to", "passwd") {
		return nil, fmt.Errorf("args must contain from, to, passwd when send Tx")
	}

	for _, opt := range options {
		keyIndex := strings.Index(opt, ":")
		key, value := strings.TrimSpace(opt[:keyIndex]), strings.TrimSpace(opt[keyIndex+1:])
		switch key {
		case "from":
			if !strings.HasPrefix(value, "0x") {
				return nil, fmt.Errorf("from must begin with 0x")
			}
			from, err := hex.DecodeString(value[2:])
			if err != nil {
				return nil, fmt.Errorf("decode from err: %s", err.Error())
			}
			args.From = common.BytesToAddress(from)
		case "to":
			if !strings.HasPrefix(value, "0x") {
				return nil, fmt.Errorf("to must begin with 0x")
			}
			to, err := hex.DecodeString(value[2:])
			if err != nil {
				return nil, fmt.Errorf("decode to err: %s", err.Error())
			}
			toBytes := common.BytesToAddress(to)
			args.To = &toBytes
		case "gas":
			gas, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("gas must be uint64 err: %s", err.Error())
			}
			args.Gas = &gas
		case "gasPrice":
			gasPrice, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("gasPrice must be uing64 err: %s", err.Error())
			}
			args.GasPrice = &gasPrice
		case "value":
			val, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("value must be uint64 err: %s", err.Error())
			}
			args.Value = &val
		case "nonce":
			nonce, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("nonce must be uing64 err: %s", err.Error())
			}
			args.Nonce = &nonce
		case "passwd":
			passwd = value
		case "code":
			codehex, err := hex.DecodeString(value)
			if err != nil {
				return nil, fmt.Errorf("Failed to convert code, %v", err)
			}
			args.Data = codehex
		}
	}

	if len(args.Data) == 0 && args.Value == nil {
		return nil, fmt.Errorf("value can be ignored only when code is set")
	}

	// default value when they are not assigned
	var (
		gas      uint64 = 1000000
		gasPrice uint64 = 1
		value    uint64 = 0
	)

	if args.Gas == nil {
		args.Gas = &gas
	}
	if args.GasPrice == nil {
		args.GasPrice = &gasPrice
	}
	if args.Value == nil {
		args.Value = &value
	}

	var resp *common.Hash
	if client == nil {
		return nil, fmt.Errorf("client is nill")
	}

	if err := client.Call(&resp, "account_sendTransaction", args, passwd); err != nil {
		return nil, err
	}

	contractInfo := ContractInfo{
		Hash: *resp,
	}

	chanTxReceipt <- contractInfo

	return hexutil.Encode(resp.Bytes()), nil
}

func GetLatestDsBlock(options []string) (interface{}, error) {
	var resp *server.RPCDsBlock
	if err := client.Call(&resp, "okchain_getLatestDsBlock", nil); err != nil {
		return nil, err
	}
	return resp, nil
}

func GetLatestTxBlock(options []string) (interface{}, error) {
	var resp *server.RPCTxBlock
	if err := client.Call(&resp, "okchain_getLatestTxBlock", nil); err != nil {
		return nil, err
	}
	return resp, nil
}

func GetTransaction(options []string) (interface{}, error) {
	if len(options) != 1 {
		return nil, fmt.Errorf("GetTransaction arguments, want 1, got %d", len(options))
	}
	var resp *server.RPCTransaction
	if err := client.Call(&resp, "okchain_getTransactionByHash", options[0]); err != nil {
		return nil, err
	}
	return resp, nil
}

func GetAccount(options []string) (interface{}, error) {
	if len(options) != 1 {
		return nil, fmt.Errorf("GetAccount arguments, want 1, got %d", len(options))
	}
	var resp protos.Account
	if err := client.Call(&resp, "okchain_getAccount", options[0]); err != nil {
		return nil, err
	}
	return resp, nil
}

func NewAccount(options []string) (interface{}, error) {
	if len(options) != 1 {
		return nil, fmt.Errorf("NewAccount arguments, want 1, got %d", len(options))
	}
	var resp common.Address
	if err := client.Call(&resp, "account_newAccount", options[0]); err != nil {
		return nil, err
	}
	return resp.String(), nil
}

func ListAccounts(options []string) (interface{}, error) {
	//if len(options) != 1 {
	//	return nil, fmt.Errorf("ListAccounts arguments, want 1, got %d", len(options))
	//}
	var resp []common.Address
	if err := client.Call(&resp, "account_listAccounts"); err != nil {
		return nil, err
	}
	var res_str []string
	for i := 0; i < len(resp); i++ {
		res_str = append(res_str, resp[i].String())
	}
	return res_str, nil
}

// DeployContract deploys a contract with arguments by sending a transaction with data.
// Console will receive a notification implying if tx deploying contract is included by
// final block or not after while.
func DeployContract(options []string) (interface{}, error) {
	txArgs := &server.SendTxArgs{}
	var abiPath, binPath, contractName, passwd string

	if !checkSubstrings(options, "from", "path", "contractName", "passwd", "initargs") {
		return nil, fmt.Errorf("args must contain from, path, contractName, passwd, initargs when deploy contract")
	}
	for _, opt := range options {
		if strings.Contains(opt, "contractName") {
			keyIndex := strings.Index(opt, ":")
			value := strings.TrimSpace(opt[keyIndex+1:])
			if _, ok := contractAbis[value]; ok || value == "okc" || value == "okwallet" {
				return nil, fmt.Errorf("contract name can not be set okc/okwallet or %s has been deployed, please set another name", value)
			}
			contractName = value
		}
	}
	for _, opt := range options {
		keyIndex := strings.Index(opt, ":")
		key, value := strings.TrimSpace(opt[:keyIndex]), strings.TrimSpace(opt[keyIndex+1:])
		switch key {
		case "from":
			if !strings.HasPrefix(value, "0x") {
				return nil, fmt.Errorf("from must begin with 0x")
			}
			from, err := hex.DecodeString(value[2:])
			if err != nil {
				return nil, fmt.Errorf("decode from err: %s", err.Error())
			}
			txArgs.From = common.BytesToAddress(from)
		case "gas":
			gas, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("gas must be uint64 err: %s", err.Error())
			}
			txArgs.Gas = &gas
		case "gasPrice":
			gasPrice, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("gasPrice must be uing64 err: %s", err.Error())
			}
			txArgs.GasPrice = &gasPrice
		case "value":
			val, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("value must be uing64 err: %s", err.Error())
			}
			txArgs.Value = &val
		case "nonce":
			nonce, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("nonce must be uing64 err: %s", err.Error())
			}
			txArgs.Nonce = &nonce
		case "passwd":
			passwd = value
		case "path":
			path := strings.Split(value, "/")
			if strings.HasSuffix(value, "/") {
				contractDirName := path[len(path)-2]
				abiPath = value + contractDirName + ".abi"
				binPath = value + contractDirName + ".bin"
			} else {
				contractDirName := path[len(path)-1]
				abiPath = value + "/" + contractDirName + ".abi"
				binPath = value + "/" + contractDirName + ".bin"
			}

			binContents, err := ioutil.ReadFile(binPath)
			if err != nil {
				return nil, fmt.Errorf("read bincontents err: %s", err.Error())
			}
			txArgs.Data, err = hex.DecodeString(string(binContents))
			if err != nil {
				return nil, fmt.Errorf("decode data err: %s", err.Error())
			}
			abiContents, err := ioutil.ReadFile(abiPath)
			if err != nil {
				return nil, fmt.Errorf("read abicontents err: %s", err.Error())
			}
			//result := strings.Replace(string(contents),"\n","",1)
			fmt.Println(string(abiContents))
			contractAbi, err := abi.JSON(strings.NewReader(string(abiContents)))
			if err != nil {
				return nil, err
			}
			contractAbis[contractName] = contractAbi
		case "initargs":
			methodArgs := strings.Split(value, ",")
			if len(methodArgs) == 1 && methodArgs[0] == "" {
				methodArgs = []string{}
			}
			contractAbi := contractAbis[contractName]
			if len(methodArgs) != len(contractAbi.Constructor.Inputs) {
				return nil, fmt.Errorf("constructor need %d arguments, got %d",
					len(contractAbi.Constructor.Inputs), len(methodArgs))
			}
			args, err := ParseMethodInputs(contractAbi.Constructor.Inputs, methodArgs)
			if err != nil {
				return nil, err
			}
			initCode, err := contractAbi.Pack("", args...)
			if err != nil {
				return nil, err
			}
			var buffer bytes.Buffer
			buffer.Write(txArgs.Data)
			buffer.Write(initCode)
			txArgs.Data = buffer.Bytes()
		}

	}

	// default value when they are not assigned
	var (
		gas      uint64 = 1000000
		gasPrice uint64 = 1
		value    uint64 = 0
	)

	if txArgs.Gas == nil {
		txArgs.Gas = &gas
	}
	if txArgs.GasPrice == nil {
		txArgs.GasPrice = &gasPrice
	}
	if txArgs.Value == nil {
		txArgs.Value = &value
	}
	var resp *common.Hash
	if client == nil {
		return nil, fmt.Errorf("client is nill")
	}
	if err := client.Call(&resp, "account_sendTransaction", txArgs, passwd); err != nil {
		delete(contractAbis, contractName)
		return nil, err
	}

	contractInfo := ContractInfo{
		Hash: *resp,
		Name: contractName,
		Abi:  contractAbis[contractName],
	}

	chanTxReceipt <- contractInfo

	return hexutil.Encode(resp.Bytes()), nil
}

// GetTxReceiptLoop performs twenty JSON-RPC calls with the hash of Tx for
// getting the receipt of the Tx per 100s in the background.
// Tx is a common transaction when ch.Name is null and a transaction deploying
// contract when ch.Name is not null.
func GetTxReceiptLoop() {
	defer close(chanTxReceipt)
	for {
		ch := <-chanTxReceipt
		var resp = map[string]interface{}{}
		for i := 0; i < 20; i++ {
			if err := client.Call(&resp, "okchain_getTransactionReceipt", &ch.Hash); err != nil {
				fmt.Printf("get receipt of Tx: 0x%s err : %s", hex.EncodeToString(ch.Hash.Bytes()), err.Error())
				break
			}
			if len(resp) != 0 {
				break
			}
			time.Sleep(time.Second * 5)
		}

		if v, ok := resp["status"]; ok {
			if v == "0x0" {
				if ch.Name != "" {
					delete(contractAbis, ch.Name)
					fmt.Printf("the Tx 0x%s deploying contract failed!\n\n\n", hex.EncodeToString(ch.Hash.Bytes()))
				} else {
					fmt.Printf("the Tx 0x%s failed!\n\n\n", hex.EncodeToString(ch.Hash.Bytes()))
				}

			} else if v == "0x1" {
				if ch.Name != "" {
					var contractSub []prompt.Suggest
					contractStr, ok := resp["contractAddress"].(string)
					if !ok {
						fmt.Printf(" error  resp[\"contractAddress\"] is not string\n\n\n")
						continue
					}
					contractAddrs[ch.Name] = common.HexToAddress(contractStr[2:])
					contractSub = append(contractSub, prompt.Suggest{Text: CmdAttrFlag(ch.Name, "address"), Description: "contract address"})
					for methodName, method := range ch.Abi.Methods {
						var methodArgs = []string{"from:0x94dc66c8be8393e41791dfd7b8a47fe43f3e0890", "gas:1000000", "gasPrice:1", "value:100000", "nonce:0", "passwd:okchain"}
						var args = string("callargs:")
						for i, typ := range method.Inputs {
							methodArg := fmt.Sprintf("%s", typ.Type)
							if i == 0 {
								args = args + methodArg
							}
							args = args + " " + methodArg
						}
						methodArgs = append(methodArgs, args)
						contractSub = append(contractSub, prompt.Suggest{Text: CmdOptionsFlag(ch.Name, methodName, methodArgs), Description: "contract call: from, args are required"})
					}
					promptSuggests[ch.Name] = contractSub
					contractAbis[ch.Name] = ch.Abi
					var contractDecp string
					contractDecp = fmt.Sprintf("the contract instance of %s", ch.Name)
					commands = append(commands, prompt.Suggest{Text: ch.Name, Description: contractDecp})
					fmt.Printf("the Tx 0x%s deploying contraStrct succeed! contract address : 0x%s\n\n\n", hex.EncodeToString(ch.Hash.Bytes()), hex.EncodeToString(contractAddrs[ch.Name].Bytes()))
				} else {
					fmt.Printf("the Tx 0x%s succeed!\n\n\n", hex.EncodeToString(ch.Hash.Bytes()))
				}
			}
		} else {
			delete(contractAbis, ch.Name)
			fmt.Printf("the Tx 0x%s deploying contract can not get receipt!\n\n", hex.EncodeToString(ch.Hash.Bytes()))
		}
	}
}

// GetTransactionReceipt performs a JSON-RPC call with the hash of Tx.
func GetTransactionReceipt(options []string) (interface{}, error) {
	if len(options) != 1 {
		return nil, fmt.Errorf("GetTransactionReceipt arguments, want 1, got %d", len(options))
	}
	if !strings.HasPrefix(options[0], "0x") {
		return nil, fmt.Errorf("tx_hash must begin with 0x")
	}
	hash, err := hex.DecodeString(options[0][2:])
	if err != nil {
		return nil, fmt.Errorf("decode tx_hash err: %s", err.Error())
	}
	txhash := common.BytesToHash(hash)
	var resp = map[string]interface{}{}
	if err := client.Call(&resp, "okchain_getTransactionReceipt", &txhash); err != nil {
		return nil, err
	}
	return resp, nil
}

// ContractCall call a function which contract code defines with arguments abi packed.
func ContractCall(cmd, subcmd string, options []string) (interface{}, error) {
	contractAbi := contractAbis[cmd]
	var passwd string
	if !checkSubstrings(options, "from", "callargs", "passwd") {
		return nil, fmt.Errorf("Error : must contain from, args, passwd when call contract")
	}
	txArg := &server.SendTxArgs{}
	for _, opt := range options {
		keyIndex := strings.Index(opt, ":")
		key, value := strings.TrimSpace(opt[:keyIndex]), strings.TrimSpace(opt[keyIndex+1:])
		switch key {
		case "from":
			if !strings.HasPrefix(value, "0x") {
				return nil, fmt.Errorf("from must begin with 0x")
			}
			from, err := hex.DecodeString(value[2:])
			if err != nil {
				return nil, fmt.Errorf("decode from err: %s", err.Error())
			}
			txArg.From = common.BytesToAddress(from)
		case "gas":
			gas, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("gas must be uint64 err: %s", err.Error())
			}
			txArg.Gas = &gas
		case "gasPrice":
			gasPrice, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("gasPrice must be uing64 err: %s", err.Error())
			}
			txArg.GasPrice = &gasPrice
		case "value":
			val, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("value must be uing64 err: %s", err.Error())
			}
			txArg.Value = &val
		case "nonce":
			nonce, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("nonce must be uing64 err: %s", err.Error())
			}
			txArg.Nonce = &nonce
		case "passwd":
			passwd = value
		case "callargs":
			methodArgs := strings.Split(value, ",")
			if len(methodArgs) == 1 && methodArgs[0] == "" {
				methodArgs = []string{}
			}
			if len(methodArgs) != len(contractAbi.Methods[subcmd].Inputs) {
				return nil, fmt.Errorf("%s need %d arguments, got %d", subcmd, len(contractAbi.Methods[subcmd].Inputs), len(methodArgs))
			}
			args, err := ParseMethodInputs(contractAbi.Methods[subcmd].Inputs, methodArgs)
			if err != nil {
				return nil, err
			}
			callCode, err := contractAbi.Pack(subcmd, args...)
			if err != nil {
				return nil, err
			}
			txArg.Data = callCode
		}
	}

	// construct transaction to call contract method
	var (
		gas             uint64 = 1000000
		gasPrice        uint64 = 1
		value           uint64 = 0
		contractAddress        = contractAddrs[cmd]
	)

	txArg.To = &contractAddress
	if txArg.Gas == nil {
		txArg.Gas = &gas
	}
	if txArg.GasPrice == nil {
		txArg.GasPrice = &gasPrice
	}
	if txArg.Value == nil {
		txArg.Value = &value
	} else if *txArg.Value > 0 && contractAbi.Methods[subcmd].Const {
		return nil, fmt.Errorf("value must be zero when method called is constant")
	}

	if client == nil {
		return nil, fmt.Errorf("client is nill")
	}

	// decide to perform okchain_call or account_sendTransaction JSON-RPC call according to
	// Const attribute of contract method.
	if contractAbi.Methods[subcmd].Const {
		var resp []byte
		callArg := &server.CallArgs{
			From:     txArg.From,
			To:       txArg.To,
			Gas:      txArg.Gas,
			GasPrice: txArg.GasPrice,
			Value:    txArg.Value,
			Data:     txArg.Data,
		}
		err := client.Call(&resp, "okchain_call", callArg)
		if err != nil {
			return nil, err
		}
		return hexutil.Encode(resp), nil
	} else {
		var resp *common.Hash
		err := client.Call(&resp, "account_sendTransaction", txArg, passwd)
		if err != nil {
			return nil, err
		}
		contractInfo := ContractInfo{
			Hash: *resp,
		}
		chanTxReceipt <- contractInfo
		return hexutil.Encode(resp.Bytes()), nil
	}
}

// Instantiate a contract with address, path where contract abi is and contractName identifying
// the contract instance. After that, input contractName to call contract according to prompt.
func ContractAt(options []string) (interface{}, error) {
	addressExist, pathExist, contractNameExist := false, false, false
	var abiPath, contractName, address string
	for _, opt := range options {
		if strings.Contains(opt, "address") {
			keyIndex := strings.Index(opt, ":")
			value := strings.TrimSpace(opt[keyIndex+1:])
			address = value
			addressExist = true
		}
		if strings.Contains(opt, "path") {
			pathExist = true
		}
		if strings.Contains(opt, "contractName") {
			keyIndex := strings.Index(opt, ":")
			value := strings.TrimSpace(opt[keyIndex+1:])
			if _, ok := contractAbis[value]; ok || value == "okc" || value == "okwallet" {
				return nil, fmt.Errorf("contract name can not be set okc/okwallet or %s has been used, please set another name", value)
			}
			contractName = value
			contractNameExist = true
		}
	}

	if (!pathExist) || (!contractNameExist) || (!addressExist) {
		return nil, fmt.Errorf("args must contain address, path, contractName when find deployed contract")
	}

	for _, opt := range options {
		keyIndex := strings.Index(opt, ":")
		key, value := strings.TrimSpace(opt[:keyIndex]), strings.TrimSpace(opt[keyIndex+1:])
		switch key {
		case "address":
			if !strings.HasPrefix(value, "0x") {
				return nil, fmt.Errorf("address must begin with 0x")
			}
			resp, err := GetAccount([]string{value})
			if err != nil {
				return nil, err
			}
			account, _ := resp.(protos.Account)
			codeHash := account.CodeHash
			if hex.EncodeToString(codeHash) == emptyHash {
				return nil, fmt.Errorf("the address is not a contract")
			}
		case "path":
			path := strings.Split(value, "/")
			if strings.HasSuffix(value, "/") {
				contractDirName := path[len(path)-2]
				abiPath = value + contractDirName + ".abi"
			} else {
				contractDirName := path[len(path)-1]
				abiPath = value + "/" + contractDirName + ".abi"
			}
			abiContents, err := ioutil.ReadFile(abiPath)
			if err != nil {
				return nil, fmt.Errorf("read abicontents err: %s", err.Error())
			}
			contractAbi, err := abi.JSON(strings.NewReader(string(abiContents)))
			if err != nil {
				return nil, err
			}
			// contract subcmd i.e. the method of contract
			var contractSub []prompt.Suggest
			contractAddrs[contractName] = common.HexToAddress(address)
			contractSub = append(contractSub, prompt.Suggest{Text: CmdAttrFlag(contractName, "address"), Description: "contract address"})
			for methodName, method := range contractAbi.Methods {
				var methodArgs = []string{"from:0xbcf72fc47d4aeec195dba5e0e26cea2b75416ec4", "gas:1000000", "gasPrice:1", "value:100000", "nonce:0", "passwd:okchain"}
				var args = string("callargs:")
				for i, typ := range method.Inputs {
					methodArg := fmt.Sprintf("%s", typ.Type)
					if i == 0 {
						args = args + methodArg
					}
					args = args + " " + methodArg
				}
				methodArgs = append(methodArgs, args)
				contractSub = append(contractSub, prompt.Suggest{Text: CmdOptionsFlag(contractName, methodName, methodArgs), Description: "contract call: from, args are required"})
			}
			promptSuggests[contractName] = contractSub
			contractAbis[contractName] = contractAbi
			var contractDecp string
			contractDecp = fmt.Sprintf("the contract instance of %s", contractName)
			commands = append(commands, prompt.Suggest{Text: contractName, Description: contractDecp})
			contractAbis[contractName] = contractAbi
		}
	}
	return true, nil
}

// ParseMethodInputs can parse type of uintX, intX, string, bytes, address, bool and those
// multidimensional	array.
// uintX implies uint8, uint16, uint24 ... uint256 and the same to intX.
func ParseMethodInputs(inputs abi.Arguments, argsStr []string) ([]interface{}, error) {
	var args []interface{}
	for i, input := range inputs {
		if strings.HasPrefix(input.Type.String(), "uint") {
			arg, err := HandleUintArgs(input.Type.String()[4:], argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		} else if strings.HasPrefix(input.Type.String(), "int") {
			arg, err := HandleIntArgs(input.Type.String()[3:], argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		} else if strings.HasPrefix(input.Type.String(), "string") {
			arg, err := HandleStringArgs(input.Type.String()[6:], argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		} else if strings.HasPrefix(input.Type.String(), "bytes") {
			arg, err := HandleBytesArgs(input.Type.String()[5:], argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		} else if strings.HasPrefix(input.Type.String(), "address") {
			arg, err := HandleAddressArgs(input.Type.String()[7:], argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		} else if strings.HasPrefix(input.Type.String(), "bool") {
			arg, err := HandleBoolArgs(input.Type.String()[4:], argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		} else if strings.HasPrefix(input.Type.String(), "uint") {
			arg, err := HandleUintArgs("256", argsStr[i])
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}
	}
	return args, nil
}

// HandleUintArgs can handle type of uint bytes and uint array.
// The elements of array is splited by space not comma.
func HandleUintArgs(argType, arg string) (interface{}, error) {
	arrIndex := strings.Index(argType, "[")
	if arrIndex == -1 {
		uintBytes, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("convert uintBytes err : %s", err.Error())
		}
		return ParseUintBytes(arg, int(uintBytes))
	} else if strings.Contains(arg, "[") && strings.Contains(arg, "]") {
		uintBytes, err := strconv.ParseUint(argType[:arrIndex], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("convert uintBytes err : %s", err.Error())
		}
		return ParseArgsArr("uint", arg, int(uintBytes))
	}
	return nil, fmt.Errorf("can not handle arg type : %s", argType)
}

// ParseUintBytes can handle uint8, uint16, uint24, uint32 ... uint256.
func ParseUintBytes(arg string, uintBytes int) (interface{}, error) {
	switch uintBytes {
	case 8:
		val, err := strconv.ParseUint(arg, 10, 8)
		if err != nil {
			return nil, err
		}
		return uint8(val), nil
	case 16:
		val, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			return nil, err
		}
		return uint16(val), nil
	case 32:
		val, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			return nil, err
		}
		return uint32(val), nil
	case 64:
		val, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return nil, err
		}
		return val, nil
	default:
		val := new(big.Int)
		val, ok := val.SetString(arg, 10)
		if !ok {
			return nil, fmt.Errorf("big Int SetString error")
		}
		return val, nil
	}
}

//func ParseUintArr(arg string, uintBytes int) (interface{}, error) {
//	var (
//		argSlice []interface{}
//		strVal string
//		convert = false
//	)
//	for i := len(arg)-1; i >= 0; i-- {
//		val := arg[i]
//		fmt.Println(string(val))
//		if string(val) == "]" {
//			argSlice = append(argSlice, string(val))
//		} else if string(val) == " " || (string(val) == "[" && convert) {
//			if strVal != "" {
//				strVal = reverseString(strVal)
//				value, err := ParseUintBytes(strVal, uintBytes)
//				if err != nil {
//					return nil, err
//				}
//				argSlice = append(argSlice, value)
//				strVal = ""
//			}
//			if string(val) == "[" && convert {
//				i++
//				convert = false
//			}
//		} else if string(val) == "[" {
//			arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(argSlice[len(argSlice)-1])), 0, 0)
//			for  {
//				pop := argSlice[len(argSlice)-1]
//				argSlice = argSlice[:len(argSlice)-1]
//				fmt.Println(pop, argSlice)
//				str1, ok := pop.(string)
//				fmt.Println(str1)
//				if ok && str1 == "]" {
//					break
//				}
//
//				arr = reflect.Append(arr, reflect.ValueOf(pop))
//			}
//			argSlice = append(argSlice, arr.Interface())
//			fmt.Println(argSlice)
//		} else {
//			strVal = strVal + string(val)
//			if i >= 1 && string(arg[i-1]) == "[" {
//				convert = true
//			}
//		}
//	}
//	if len(argSlice) <= 0 {
//		return nil, fmt.Errorf("input array is null")
//	}
//	return argSlice[0], nil
//}

// HandleIntArgs can handle type of int bytes and int array.
// The elements of array is splited by space not comma.
func HandleIntArgs(argType, arg string) (interface{}, error) {
	arrIndex := strings.Index(argType, "[")
	if arrIndex == -1 {
		intBytes, err := strconv.ParseInt(arg, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("convert intBytes err : %s", err.Error())
		}
		return ParseIntBytes(arg, int(intBytes))
	} else if strings.Contains(arg, "[") && strings.Contains(arg, "]") {
		intBytes, err := strconv.ParseInt(argType[:arrIndex], 10, 16)
		if err != nil {
			return nil, fmt.Errorf("convert uintBytes err : %s", err.Error())
		}
		return ParseArgsArr("int", arg, int(intBytes))
	}

	return nil, fmt.Errorf("%s frormat is not true", arg)
}

// ParseIntBytes can handle int8, int16, int24, int32 ... int256.
func ParseIntBytes(arg string, intBytes int) (interface{}, error) {
	switch intBytes {
	case 8:
		val, err := strconv.ParseInt(arg, 10, 8)
		if err != nil {
			return nil, err
		}
		return int8(val), nil
	case 16:
		val, err := strconv.ParseInt(arg, 10, 16)
		if err != nil {
			return nil, err
		}
		return int16(val), nil
	case 32:
		val, err := strconv.ParseInt(arg, 10, 32)
		if err != nil {
			return nil, err
		}
		return int32(val), nil
	case 64:
		val, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return nil, err
		}
		return val, nil
	default:
		val := new(big.Int)
		val, ok := val.SetString(arg, 10)
		if !ok {
			return nil, fmt.Errorf("big Int SetString error")
		}
		return val, nil
	}
}

//func ParseIntArr(arg string, intBytes int) (interface{}, error) {
//	var (
//		argSlice []interface{}
//		strVal string
//		convert = false
//	)
//	for i := len(arg)-1; i >= 0; i-- {
//		val := arg[i]
//		fmt.Println(string(val))
//		if string(val) == "]" {
//			argSlice = append(argSlice, string(val))
//		} else if string(val) == " " || (string(val) == "[" && convert) {
//			if strVal != "" {
//				strVal = reverseString(strVal)
//				value, err := ParseIntBytes(strVal, intBytes)
//				if err != nil {
//					return nil, err
//				}
//				argSlice = append(argSlice, value)
//				strVal = ""
//			}
//			if string(val) == "[" && convert {
//				i++
//				convert = false
//			}
//		} else if string(val) == "[" {
//			arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(argSlice[len(argSlice)-1])), 0, 0)
//			for  {
//				pop := argSlice[len(argSlice)-1]
//				argSlice = argSlice[:len(argSlice)-1]
//				fmt.Println(pop, argSlice)
//				str1, ok := pop.(string)
//				fmt.Println(str1)
//				if ok && str1 == "]" {
//					break
//				}
//
//				arr = reflect.Append(arr, reflect.ValueOf(pop))
//			}
//			argSlice = append(argSlice, arr.Interface())
//			fmt.Println(argSlice)
//		} else {
//			strVal = strVal + string(val)
//			if i >= 1 && string(arg[i-1]) == "[" {
//				convert = true
//			}
//		}
//	}
//	if len(argSlice) <= 0 {
//		return nil, fmt.Errorf("input array is null")
//	}
//	return argSlice[0], nil
//}

func HandleStringArgs(argType, arg string) (interface{}, error) {
	arrIndex := strings.Index(argType, "[")
	if arrIndex == -1 {
		return arg, nil
	} else if strings.Contains(arg, "[") && strings.Contains(arg, "]") {
		return ParseArgsArr("string", arg, 0)
	}
	return nil, fmt.Errorf("%s frormat is not true", arg)
}

//func ParseStringArr(arg string) (interface{}, error) {
//	var (
//		argSlice []interface{}
//		strVal string
//		convert = false
//	)
//	for i := len(arg)-1; i >= 0; i-- {
//		val := arg[i]
//		fmt.Println(string(val))
//		if string(val) == "]" {
//			argSlice = append(argSlice, string(val))
//		} else if string(val) == " " || (string(val) == "[" && convert) {
//			if strVal != "" {
//				strVal = reverseString(strVal)
//				argSlice = append(argSlice, strVal)
//				strVal = ""
//			}
//			if string(val) == "[" && convert {
//				i++
//				convert = false
//			}
//		} else if string(val) == "[" {
//			arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(argSlice[len(argSlice)-1])), 0, 0)
//			for  {
//				pop := argSlice[len(argSlice)-1]
//				argSlice = argSlice[:len(argSlice)-1]
//				fmt.Println(pop, argSlice)
//				str1, ok := pop.(string)
//				fmt.Println(str1)
//				if ok && str1 == "]" {
//					break
//				}
//
//				arr = reflect.Append(arr, reflect.ValueOf(pop))
//			}
//			argSlice = append(argSlice, arr.Interface())
//			fmt.Println(argSlice)
//		} else {
//			strVal = strVal + string(val)
//			if i >= 1 && string(arg[i-1]) == "[" {
//				convert = true
//			}
//		}
//	}
//	if len(argSlice) <= 0 {
//		return nil, fmt.Errorf("input array is null")
//	}
//	return argSlice[0], nil
//}

// HandleIntArgs can handle type of bytes.
// The elements of array is splited by space not comma.
func HandleBytesArgs(argType, arg string) (interface{}, error) {
	arrIndex := strings.Index(argType, "[")
	if arrIndex == -1 {
		return []byte(arg), nil
	} else if strings.Contains(arg, "[") && strings.Contains(arg, "]") {
		return ParseArgsArr("bytes", arg, 0)
	}
	return nil, fmt.Errorf("%s frormat is not true", arg)
}

// HandleIntArgs can handle type of address and address array.
// The elements of array is splited by space not comma.
func HandleAddressArgs(argType, arg string) (interface{}, error) {
	arrIndex := strings.Index(argType, "[")
	if arrIndex == -1 {
		return common.HexToAddress(arg), nil
	} else if strings.Contains(arg, "[") && strings.Contains(arg, "]") {
		return ParseArgsArr("address", arg, 0)
	}
	return nil, fmt.Errorf("%s frormat is not true", arg)
}

// HandleIntArgs can handle type of bool and bool array.
// The elements of array is splited by space not comma.
func HandleBoolArgs(argType, arg string) (interface{}, error) {
	arrIndex := strings.Index(argType, "[")
	if arrIndex == -1 {
		if arg == "true" {
			return true, nil
		} else if arg == "false" {
			return false, nil
		} else {
			return nil, fmt.Errorf("bool only accept ture of false")
		}
	} else if strings.Contains(arg, "[") && strings.Contains(arg, "]") {
		return ParseArgsArr("bool", arg, 0)
	}
	return nil, fmt.Errorf("%s frormat is not true", arg)
}

func reverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

// ParseArg handle arg according to argType.
// info implies X in uintX and intX.
func ParseArg(argType, arg string, info int) (interface{}, error) {
	switch argType {
	case "uint":
		return ParseUintBytes(arg, info)
	case "int":
		return ParseIntBytes(arg, info)
	case "string":
		return arg, nil
	case "bytes":
		return []byte(arg), nil
	case "address":
		return common.HexToAddress(arg), nil
	case "bool":
		if arg == "true" {
			return true, nil
		} else if arg == "false" {
			return false, nil
		} else {
			return nil, fmt.Errorf("bool only accept ture of false")
		}
	}
	return nil, fmt.Errorf("unknown arg type %s", argType)
}

// ParseArgsArr can handle array of argType.
func ParseArgsArr(argType, arg string, info int) (interface{}, error) {
	var (
		argSlice []interface{}
		strVal   string
		convert  = false
	)
	for i := len(arg) - 1; i >= 0; i-- {
		val := arg[i]
		fmt.Println(string(val))
		if string(val) == "]" {
			argSlice = append(argSlice, string(val))
		} else if string(val) == " " || (string(val) == "[" && convert) {
			if strVal != "" {
				strVal = reverseString(strVal)
				value, err := ParseArg(argType, strVal, info)
				if err != nil {
					return nil, err
				}
				argSlice = append(argSlice, value)
				strVal = ""
			}
			if string(val) == "[" && convert {
				i++
				convert = false
			}
		} else if string(val) == "[" {
			arr := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(argSlice[len(argSlice)-1])), 0, 0)
			for {
				pop := argSlice[len(argSlice)-1]
				argSlice = argSlice[:len(argSlice)-1]
				fmt.Println(pop, argSlice)
				str1, ok := pop.(string)
				fmt.Println(str1)
				if ok && str1 == "]" {
					break
				}

				arr = reflect.Append(arr, reflect.ValueOf(pop))
			}
			argSlice = append(argSlice, arr.Interface())
			fmt.Println(argSlice)
		} else {
			strVal = strVal + string(val)
			if i >= 1 && string(arg[i-1]) == "[" {
				convert = true
			}
		}
	}
	if len(argSlice) <= 0 {
		return nil, fmt.Errorf("input array is null")
	}
	return argSlice[0], nil
}

func LockAccount(options []string) (interface{}, error) {
	if len(options) != 1 {
		return nil, fmt.Errorf("LockAccount arguments, want 1, got %d", len(options))
	}
	var resp interface{}
	if err := client.Call(&resp, "account_lockAccount", common.BytesToAddress(common.FromHex(options[0]))); err != nil {
		return nil, err
	}
	return resp, nil
}

func UnlockAccount(options []string) (interface{}, error) {
	num := len(options)
	if num < 2 || num > 3 {
		return nil, fmt.Errorf("UnlockAccount arguments, want 2(addr,pswd) or 3(addr,pswd,unlocksecond), got %d", len(options))
	}
	var resp bool
	if num == 2 {
		if err := client.Call(&resp, "account_unlockAccount", common.BytesToAddress(common.FromHex(options[0])), options[1]); err != nil {
			return nil, err
		}
	} else {
		t, err := strconv.ParseInt(options[2], 10, 64)
		if err != nil {
			return nil, err
		}
		var second = uint64(t)
		if err := client.Call(&resp, "account_unlockAccount", common.BytesToAddress(common.FromHex(options[0])), options[1], &second); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// checkSubstrings check strs contains all subs.
func checkSubstrings(strs []string, subs ...string) bool {
	containCnt := 0
	subsCnt := len(subs)
	for _, str := range strs {
		for i, sub := range subs {
			if strings.Contains(str, sub) {
				subs = append(subs[:i], subs[i+1:]...)
				containCnt++
				break
			}
		}
	}
	return containCnt == subsCnt
}
