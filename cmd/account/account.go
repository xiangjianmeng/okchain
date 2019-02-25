// Copyright The go-okchain Authors 2018,  All rights reserved.

package account

import (
	"fmt"

	"io/ioutil"

	"github.com/ok-chain/okchain/accounts"
	"github.com/ok-chain/okchain/accounts/keystore"
	server "github.com/ok-chain/okchain/api/jsonrpc_server"
	"github.com/ok-chain/okchain/cmd/console"
	"github.com/ok-chain/okchain/cmd/mnemonic"
	"github.com/ok-chain/okchain/common"
	"github.com/ok-chain/okchain/config"

	"io"
	"os"
	"strings"

	"github.com/ok-chain/okchain/common/rlp"
	logging "github.com/ok-chain/okchain/log"
	"github.com/ok-chain/okchain/protos"
	"github.com/ok-chain/okchain/rpc"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/urfave/cli.v1"
)

const accountFuncName = "account"

var (
	ks     *keystore.KeyStore
	signer *protos.P256Signer
)
var accountcmdlogger = logging.MustGetLogger("accountcmd")

func init() {
	config.SetDefaultViperConfig()
	ksDir := viper.GetString("datadir") + "/keystore"
	ks = keystore.NewKeyStore(ksDir, keystore.StandardScryptN, keystore.StandardScryptP)
	// signer = protos.MakeSigner("P256")
	signer = &protos.P256Signer{}
}

// Cmd returns the cobra command for Node
func GetAccountCmd() *cobra.Command {
	accountCmd.AddCommand(buildnewaccountCmd())
	accountCmd.AddCommand(listaccountCmd)
	accountCmd.AddCommand(buildsendTransactionCmd())
	accountCmd.AddCommand(buildgetAccountCmd())
	accountCmd.AddCommand(buildimportaccountCmd())
	accountCmd.AddCommand(buildRemoveAccountCmd())
	//accountCmd.AddCommand(buidlMnemonicCmd())
	return accountCmd
}

var accountCmd = &cobra.Command{
	Use:   accountFuncName,
	Short: "Manage accounts",
	Long:  `Manage accounts`,
}

func buildnewaccountCmd() *cobra.Command {
	// Set the flags on the node start command.
	flags := newaccountCmd.Flags()
	flags.String("password", viper.GetString("keystorepassword"), "required, account password")
	flags.String("url", "http://localhost:16001", "optional, an avaliable okchain node address")
	//flags.Bool("key",false,"")
	flags.Bool("givememoney", false, "optional, way to become rich !")
	flags.MarkHidden("url")
	flags.MarkHidden("givememoney")
	return newaccountCmd
}

var newaccountCmd = &cobra.Command{
	Use:   "create",
	Short: "create a new account",
	Long:  "create a new account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return InvalidInputArgs(args)
		}
		return accountCreate(cmd)
	},
}

//var newMnemonicCmd = &cobra.Command{
//	Use:   "mnemonic",
//	Short: "create a mnemonic randomly",
//	Long:  "a mnemonic for private key",
//	RunE: func(cmd *cobra.Command, args []string) error {
//		flags := cmd.Flags()
//		bits, err := flags.GetInt("bits")
//		if err != nil {
//			return err
//		}
//		monic, err := mnemonic.NewMnemonic(bits)
//		if err != nil {
//			return err
//		}
//		fmt.Println(monic)
//		return nil
//	},
//}
//
//func buidlMnemonicCmd() *cobra.Command {
//	flags := newMnemonicCmd.Flags()
//	flags.Int("bits", 128, "default entropy for mnemonic")
//	return newMnemonicCmd
//}

func buildgetAccountCmd() *cobra.Command {
	flags := getAccountCmd.Flags()
	flags.String("url", "http://localhost:16001", "optional, an avaliable okchain node address")
	flags.String("addr", "", "required, account address")
	flags.Bool("key", false, "optional, display account public key and private key; '--password' flag is required")
	flags.String("password", viper.GetString("keystorepassword"), "required, only if '--key' flag specified")
	getAccountCmd.Example = `  okchain account info --addr <addr> --url <url>
okchain account info --key --addr <addr> --password <password>`

	return getAccountCmd
}

var getAccountCmd = &cobra.Command{
	Use:   "info",
	Short: "display specified account information",
	Long:  "display specified account information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return InvalidInputArgs(args)
		}

		return getAccount(cmd)
	},
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(cmd *cobra.Command) error {
	//cfg := gethConfig{Node: defaultNodeConfig()}
	//// Load config file.
	//if file := ctx.GlobalString(configFileFlag.Name); file != "" {
	//	if err := loadConfig(file, &cfg); err != nil {
	//		utils.Fatalf("%v", err)
	//	}
	//}
	//utils.SetNodeConfig(ctx, &cfg.Node)
	//scryptN, scryptP, keydir, err := cfg.Node.AccountConfig()
	//
	//if err != nil {
	//	utils.Fatalf("Failed to read configuration: %v", err)
	//}
	//scryptN, scryptP, keydir, err := accountConfig()
	//password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))
	flags := cmd.Flags()
	password, err := flags.GetString("password")
	if err != nil {
		return err
	}
	//address, err := keystore.StoreKey(keydir, password, scryptN, scryptP)

	//if err != nil {
	//	accountcmdlogger.Errorf("Failed to create account: %v", err)
	//	return err
	//}
	//fmt.Println(scryptN,scryptP,keydir,password)
	mnemonic, err := mnemonic.NewMnemonic(128)
	if err != nil {
		return err
	}
	//fmt.Println(monic)
	priv, addr, err := importBySeed(mnemonic, password)
	if err != nil {
		return err
	}
	type account_output struct {
		Address    string
		Mnemonic   string
		PrivateKey string
	}
	output := account_output{
		Address:    addr,
		Mnemonic:   mnemonic,
		PrivateKey: priv,
	}
	console.PrettyPrint(output)
	//fmt.Printf("Address: {%x}\n", address)
	//flags := cmd.Flags()
	givememoney, err := flags.GetBool("givememoney")
	if err != nil {
		return err
	}
	if givememoney {
		url, err := flags.GetString("url")
		if err != nil {
			return err
		}
		client, err := rpc.DialHTTP(url)
		if err != nil {
			return err
		}
		defer client.Close()

		var resp bool
		if err := client.Call(&resp, "okchain_registerAccount", addr); err != nil {
			return err
		}
	}
	return nil
}

var listaccountCmd = &cobra.Command{
	Use:   "list",
	Short: "list all accounts in local keystore directory",
	Long:  "list all accounts in local keystore directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return InvalidInputArgs(args)
		}
		return accountList(nil)
	},
}

func accountList(ctx *cli.Context) error {
	// ks := keystore.NewKeyStore(viper.GetString("datadir")+"/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	//stack, _ := makeConfigNode(ctx)
	var index int
	for _, wallet := range ks.Wallets() {
		for _, account := range wallet.Accounts() {
			fmt.Printf("Account #%d: {%x} %s\n", index, account.Address, &account.URL)
			index++
		}
	}
	return nil
}

func buildsendTransactionCmd() *cobra.Command {
	flags := sendTransactionCmd.Flags()
	flags.String("from", "", "required, sender address")
	flags.String("to", "", "required, to address")
	flags.Uint64("gas", 1000000, "required, gas")
	flags.Uint64("gasPrice", 1, "required, gasPrice")
	flags.Uint64("amount", 1, "required, amount")
	flags.Uint64("nonce", 0, "required, transaction nonce (default 0)")
	flags.String("password", viper.GetString("keystorepassword"), "required, account password")
	flags.String("url", "http://localhost:16001", "required, an avaliable okchain node address")
	flags.Uint64("sendn", 1, "required, transaction_send_time")
	flags.Bool("offline", false, "optional, sign the transfer only, and the signed transaction hex will be stored in ./$txid.hex")
	flags.MarkHidden("sendn")
	return sendTransactionCmd
}

var sendTransactionCmd = &cobra.Command{
	Use:   "transfer",
	Short: "make an online or an offilne transfer",
	Long:  "make an online or an offilne transfer",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return InvalidInputArgs(args)
		}
		return transfer(cmd)
	},
}

func transfer(cmd *cobra.Command) error {
	flags := cmd.Flags()
	from_str, err := flags.GetString("from")
	if err != nil {
		return err
	}
	if from_str == "" {
		return exceptedFlagError("from")
	}
	from := common.BytesToAddress(common.FromHex(from_str))
	to_str, err := flags.GetString("to")
	if err != nil {
		return err
	}
	if to_str == "" {
		return exceptedFlagError("to")
	}

	to := common.BytesToAddress(common.FromHex(to_str))
	gas, err := flags.GetUint64("gas")
	if err != nil {
		return err
	}
	gasPrice, err := flags.GetUint64("gasPrice")
	if err != nil {
		return err
	}
	value, err := flags.GetUint64("amount")
	if err != nil {
		return err
	}
	nonce, err := flags.GetUint64("nonce")
	if err != nil {
		return err
	}
	passwd, err := flags.GetString("password")
	args := &server.SendTxArgs{
		From:     from,
		To:       &to,
		Gas:      &gas,
		GasPrice: &gasPrice,
		Value:    &value,
		Nonce:    &nonce,
	}

	sendn, err := flags.GetUint64("sendn")
	if err != nil {
		return err
	}
	if sendn > 1000 {
		return errors.New("sendn should be less than 1000")
	}
	if sendn == 0 {
		return errors.New("sendn should be more than 0")
	}
	ks := keystore.NewKeyStore(viper.GetString("datadir")+"/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	backends := []accounts.Backend{
		ks,
	}
	am := accounts.NewManager(backends...)
	account := accounts.Account{Address: args.From}
	w, err := am.Find(account)
	if err != nil {
		return err
	}
	err = ks.Unlock(account, passwd)
	if err != nil {
		return err
	}

	var i = nonce
	sendn += nonce

	offline, err := flags.GetBool("offline")
	if err != nil {
		return err
	}
	if offline {
		return writeRawTransaction(args, i, sendn, w)
	}
	return sendRawTransaction(flags, args, i, sendn, w)
}

func writeRawTransaction(args *server.SendTxArgs, from uint64, to uint64, w accounts.Wallet) error {
	type writeRawTransaction_output struct {
		TransactionId string
		StoreFile     string
	}
	res := []writeRawTransaction_output{}
	for i := from; i < to; i++ {
		t := i
		args.Nonce = &t
		trans := args.ToTransaction()
		account := accounts.Account{Address: args.From}
		trans, err := w.SignTx(account, trans)
		if err != nil {
			return err
		}
		rlptrans, err := rlp.EncodeToBytes(trans)
		if err != nil {
			return err
		}
		txhash := trans.Hash()
		hexrlptrans := common.ToHex(rlptrans)
		ioutil.WriteFile(txhash.String()+".hex", []byte(hexrlptrans), 0644)
		output := writeRawTransaction_output{
			TransactionId: txhash.String(),
			StoreFile:     txhash.String() + ".hex",
		}
		res = append(res, output)
		//onsole.PrettyPrint(output)
		//fmt.Printf("Transaction id: %s\n", txhash.String())
		//fmt.Printf("The signed transaction hex was stored in: %s\n", txhash.String()+".hex")
	}
	console.PrettyPrint(res)
	return nil
}

func sendRawTransaction(flags *pflag.FlagSet, args *server.SendTxArgs, from uint64, to uint64, w accounts.Wallet) error {
	nodeaddr, err := flags.GetString("url")
	if err != nil {
		return err
	}
	client, err := rpc.DialHTTP(nodeaddr)
	if err != nil {
		return err
	}
	defer client.Close()
	type sendRawTransaction_output struct {
		TransactionId string
	}
	res := []sendRawTransaction_output{}
	for i := from; i < to; i++ {
		t := i
		args.Nonce = &t
		trans := args.ToTransaction()
		account := accounts.Account{Address: args.From}
		trans, err := w.SignTx(account, trans)
		if err != nil {
			return err
		}
		//data,err:=json.Marshal(trans)
		//if err!=nil{
		//	return err
		//}
		//fmt.Println(string(data))
		var resp common.Hash
		if err := client.Call(&resp, "okchain_sendRawTransaction", trans); err != nil {
			if i+1 == to {
				return err
			}
			fmt.Println(err)
		}
		output := sendRawTransaction_output{
			TransactionId: resp.String(),
		}
		res = append(res, output)
		//fmt.Println(resp.String())
	}
	console.PrettyPrint(res)
	return nil
}

//func sendRawTransaction(trans *protos.Transaction, client *rpc.Client) (common.Hash, error) {
//	var resp bool
//	if err := client.Call(&resp, "okchain_sendRawTransaction", trans); err != nil {
//		return common.Hash{}, err
//	}
//	return trans.Hash(), nil
//}

func getAccount(cmd *cobra.Command) error {
	flags := cmd.Flags()
	accountaddr, err := flags.GetString("addr")
	if err != nil {
		return err
	}
	if accountaddr == "" {
		return exceptedFlagError("addr")
	}
	key, err := flags.GetBool("key")
	if err != nil {
		return err
	}
	if key {
		password, err := flags.GetString("password")
		if err != nil {
			return err
		}
		if password == "" {
			return exceptedFlagError("password")
		}
		account := accounts.Account{
			Address: common.HexToAddress(accountaddr),
		}
		_, key, err := ks.GetDecryptedKey(account, password)
		if err != nil {
			return err
		}
		type key_output struct {
			Address    string
			PublicKey  string
			PrivateKey string
		}
		output := key_output{
			Address:    common.HexToAddress(accountaddr).String(),
			PrivateKey: common.ToHex(signer.PrvKeyToBytes(key.PrivateKey)),
			PublicKey:  common.ToHex(signer.PubKeyToBytes(&key.PrivateKey.PublicKey)),
		}
		console.PrettyPrint(output)
		return nil
	}
	nodeaddr, err := flags.GetString("url")
	if err != nil {
		return err
	}
	if nodeaddr == "" {
		return exceptedFlagError("url")
	}
	client, err := rpc.DialHTTP(nodeaddr)
	if err != nil {
		return err
	}
	defer client.Close()
	var resp protos.Account
	//var resp map[string]interface{}
	if err := client.Call(&resp, "okchain_getAccount", accountaddr); err != nil {
		return err
	}
	//var nonce = resp["nonce"].(uint64)
	//var balance = resp["balance"].(uint64)
	//var codehash = resp["codehash"].(string)
	//var stateweight = resp["stateweight"].(uint64)
	//fmt.Println("nonce:",nonce,"\nbalance:",balance,"\ncodehash:",codehash,"\nstateweight:",stateweight)
	//fmt.Printf("%s %d\n","nonce:",resp.Nonce)
	//fmt.Printf("%s %d\n","balance:",resp.Balance)
	//fmt.Printf("%s %s\n","codehash:",common.ToHex(resp.CodeHash))
	//fmt.Printf("%s %d\n","stakewight:",resp.StakeWeight)
	type account struct {
		Nonce   uint64
		Balance uint64
		//Root        []byte
		//StakeWeight uint64
	}
	a := account{
		Nonce:   resp.Nonce,
		Balance: resp.Balance,
		//Root:        resp.Root,
		//StakeWeight: resp.StakeWeight,
	}

	console.PrettyPrint(a)
	return nil
}

func buildimportaccountCmd() *cobra.Command {
	flags := importaccountCmd.Flags()
	flags.String("mnemonic", "", "optional, mnemonic seed")
	flags.String("privatekey", "", "optional, private key")
	flags.String("password", viper.GetString("keystorepassword"), "required")
	flags.String("keystorefile", "", "optional, keystore file path")
	return importaccountCmd
}

var importaccountCmd = &cobra.Command{
	Use:   "import",
	Short: "import an account by a mnemonic seed, a private key or a keystore file.",
	Long:  "import an account by a mnemonic seed, a private key or a keystore file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return InvalidInputArgs(args)
		}
		return importaccount(cmd)
	},
}

func importaccount(cmd *cobra.Command) error {
	flags := cmd.Flags()
	importFlagNum := 0
	privatekey, err := flags.GetString("privatekey")
	if err != nil {
		return err
	}
	if privatekey != "" {
		importFlagNum++
	}
	keystorefile, err := flags.GetString("keystorefile")
	if err != nil {
		return err
	}
	if keystorefile != "" {
		importFlagNum++
	}
	seed, err := flags.GetString("mnemonic")
	if err != nil {
		return err
	}
	if seed != "" {
		importFlagNum++
	}

	if importFlagNum > 1 {
		return errors.New("only one flag is required: (keystorefile | privatekey | seed)")
	}

	password, err := flags.GetString("password")
	if err != nil {
		return err
	}
	if password == "" {
		return exceptedFlagError("password")
	}
	type import_output struct {
		Address    string
		PrivateKey string `json:"PrivateKey,omitempty"`
	}
	print := import_output{}
	if len(privatekey) > 0 {
		addr, err := importByRawKey(privatekey, password)
		if err != nil {
			return err
		}
		print.Address = addr
		console.PrettyPrint(print)
		return nil

	}
	seed, err = flags.GetString("mnemonic")
	if err != nil {
		return err
	}

	if len(seed) > 0 {
		prikey, addr, err := importBySeed(seed, password)
		if err != nil {
			return err
		}
		print.Address = addr
		print.PrivateKey = prikey
		console.PrettyPrint(print)
		return nil
	}
	if len(keystorefile) > 0 {
		addr, err := importByKeyStoreFile(keystorefile, password)
		if err != nil {
			return err
		}
		print.Address = addr
		console.PrettyPrint(print)
		return nil
	}
	return exceptedFlagError("keystorefile | privatekey | mnemonic")
}

//
//func buildsignTransactionCmd() *cobra.Command {
//	flags := signtransCmd.Flags()
//	flags.String("from", "", "sender address")
//	flags.String("to", "", "to address")
//	flags.Uint64("gas", 1000000, "gas")
//	flags.Uint64("gasPrice", 1, "gasPrice")
//	flags.Uint64("value", 1, "value")
//	flags.Uint64("nonce", 0, "transaction nonce (default 0)")
//	flags.String("password", viper.GetString("keystorepassword"), "account password")
//	flags.String("outputfile","./transaction","output filepath")
//	return signtransCmd
//}
//
//var signtransCmd = &cobra.Command{
//	Use:   "sign",
//	Short: "sign transaction",
//	Long:  "sign transaction",
//	RunE: func(cmd *cobra.Command, args []string) error {
//		return signtrans(nil)
//	},
//}
//
//func signtrans(ctx *cli.Context) error {
//
//	return nil
//}

func importByRawKey(prvStr, password string) (string, error) {
	prvBytes := common.FromHex(prvStr)
	prv, err := signer.BytesToPrvKey(prvBytes)
	if err != nil {
		return "", err
	}
	acc, err := ks.ImportECDSA(prv, password)
	if err != nil {
		return "", err
	}
	//fmt.Printf("Address: {%x}\n", acc.Address)
	return acc.Address.String(), nil
}

func importBySeed(monic, password string) (string, string, error) {
	seed, err := mnemonic.ToSeed(monic)
	if err != nil {
		return "", "", err
	}
	prvBytes, err := mnemonic.NewMaster(seed)
	if err != nil {
		return "", "", err
	}
	prv, err := signer.BytesToPrvKey(prvBytes)
	if err != nil {
		return "", "", err
	}
	acc, err := ks.ImportECDSA(prv, password)
	if err != nil {
		return "", "", err
	}
	//fmt.Printf("Address: {%x}\n", acc.Address)
	return common.ToHex(prvBytes), acc.Address.String(), nil
}

func importByKeyStoreFile(filepath string, password string) (string, error) {
	tmp := strings.Split(filepath, "/")
	filename := tmp[len(tmp)-1]
	if filename[:3] != "UTC" {
		return "", errors.New("Invalid keystore filename")
	}
	tmp = strings.Split(filename, "-")
	addr := tmp[len(tmp)-1]
	if len(addr) != 40 {
		return "", errors.New("Invalid keystore filename")
	}
	dst := viper.GetString("datadir") + "/keystore/" + filename
	_, err := CopyFile(dst, filepath)
	if err != nil {
		return "", err
	}
	ks := keystore.NewKeyStore(viper.GetString("datadir")+"/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	backends := []accounts.Backend{
		ks,
	}
	am := accounts.NewManager(backends...)
	account := accounts.Account{Address: common.HexToAddress(addr)}
	_, err = am.Find(account)
	if err != nil {
		return "", err
	}
	err = ks.Unlock(account, password)
	if err != nil {
		return "", err
	}
	return account.Address.String(), nil
}

func CopyFile(dstName, srcName string) (written int64, err error) {
	src, err := os.Open(srcName)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstName, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}

func buildRemoveAccountCmd() *cobra.Command {
	flags := removeAccountCmd.Flags()
	flags.String("addr", "", "required, account address")
	flags.String("password", viper.GetString("keystorepassword"), "required, account password")

	return removeAccountCmd
}

var removeAccountCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove specified account information from local keystore",
	Long:  "remove specified account information from local keystore",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return InvalidInputArgs(args)
		}
		return removeAccount(cmd)
	},
}

func removeAccount(cmd *cobra.Command) error {
	flags := cmd.Flags()
	password, err := flags.GetString("password")
	if err != nil {
		return err
	}
	addr, err := flags.GetString("addr")
	if err != nil {
		return err
	}
	if addr == "" {
		return exceptedFlagError("addr")
	}
	//ks := keystore.NewKeyStore(viper.GetString("datadir")+"/keystore", keystore.StandardScryptN, keystore.StandardScryptP)
	account := accounts.Account{Address: common.HexToAddress(addr)}
	return ks.Delete(account, password)
}

func exceptedFlagError(flagname string) error {
	return errors.New("expected flag: " + flagname)
}

func InvalidInputArgs(args interface{}) error {
	return errors.New(fmt.Sprintln("invalid input args:", args))
}
