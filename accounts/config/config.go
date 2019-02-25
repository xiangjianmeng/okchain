package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ok-chain/okchain/accounts"
	logging "github.com/ok-chain/okchain/log"

	"github.com/ok-chain/okchain/accounts/keystore"
	"github.com/spf13/viper"
)

const (
	StandardScryptN = 1 << 18
	StandardScryptP = 1

	LightScryptN = 1 << 12
	LightScryptP = 6
)

const (
	datadirDefaultKeyStore = "keystore" // Path within the datadir to the keystore
	pkgLogId               = "accounts/config"
	Prefix                 = "CONFIG"
)

var (
	logger     *logging.Logger
	configName string
)

func init() {
	logger = logging.MustGetLogger(pkgLogId)
	configName = strings.ToLower(Prefix)
}

type Config struct {
	DataDir           string
	KeyStoreDir       string
	UseLightweightKDF bool
}

//default accounts configuration if not set
var defaultConfig = []byte(`
	dataDir: datadir
	keyStoreDir: keystore
	useLightWeightKDF: false
	`)

//use viper to manage config
func Load() (conf *Config, err error) {
	v := viper.New()
	InitViper(v, configName)
	//v.SetConfigType("yaml")

	// for environment variables
	v.SetEnvPrefix(Prefix)
	v.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	v.SetEnvKeyReplacer(replacer)

	if err := v.ReadInConfig(); err == nil {
		logger.Infof("Using config file: %s", v.ConfigFileUsed())
	} else if err := v.ReadConfig(bytes.NewBuffer(defaultConfig)); err == nil {
		logger.Info("Using default configuration.")
	} else {
		logger.Fatal("loading config failed!")
	}

	conf = &Config{}
	err = v.Unmarshal(conf)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling config into struct: %s", err)
	}
	return
}
func InitViper(v *viper.Viper, configName string) error {
	v.AddConfigPath("/etc/okchain/")
	v.AddConfigPath("$HOME/.okchain")
	v.AddConfigPath(".")
	// DevConfigPath
	AddDevConfigPath(v)

	// Now set the configuration file.
	v.SetConfigName(configName)

	return nil
}
func AddDevConfigPath(v *viper.Viper) error {
	devPath, err := GetDevConfigDir()
	if err != nil {
		return err
	}
	v.AddConfigPath(devPath)
	return nil
}
func GetDevConfigDir() (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		return "", fmt.Errorf("GOPATH not set")
	}

	for _, p := range filepath.SplitList(gopath) {
		devPath := filepath.Join(p, "src/github.com/ok-chain/okchain/accounts/config")
		if _, err := os.Stat(devPath); err != nil {
			continue
		}

		return devPath, nil
	}

	return "", fmt.Errorf("DevConfigDir not found in %s", gopath)
}

// AccountConfig determines the settings for scrypt and keydirectory
func (conf *Config) AccountConfig() (int, int, string, error) {
	scryptN := StandardScryptN
	scryptP := StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = LightScryptN
		scryptP = LightScryptP
	}

	var (
		keydir string
		err    error
	)
	switch {
	case filepath.IsAbs(conf.KeyStoreDir):
		keydir = conf.KeyStoreDir
	case conf.DataDir != "":
		if conf.KeyStoreDir == "" {
			keydir = filepath.Join(conf.DataDir, datadirDefaultKeyStore)
		} else {
			keydir, err = filepath.Abs(conf.KeyStoreDir)
		}
	case conf.KeyStoreDir != "":
		keydir, err = filepath.Abs(conf.KeyStoreDir)
	}
	return scryptN, scryptP, keydir, err
}

/*
	by hxy, 2018/04/19
	创建节点的账户管理器AccountManager,并设置支持的后端钱包, 目前仅包含KeyStore软钱包
*/
func MakeAccountManager(conf *Config) (*accounts.Manager, string, error) {
	scryptN, scryptP, keydir, err := conf.AccountConfig()
	var ephemeral string
	if keydir == "" {
		// There is no datadir.
		keydir, err = ioutil.TempDir("", "go-ethereum-keystore")
		ephemeral = keydir
	}

	if err != nil {
		return nil, "", err
	}
	if err := os.MkdirAll(keydir, 0700); err != nil {
		return nil, "", err
	}
	// Assemble the account manager and supported backends
	//软钱包
	backends := []accounts.Backend{
		keystore.NewKeyStore(keydir, scryptN, scryptP),
	}
	logger.Info("Setting accounts backends: keystore wallet")
	return accounts.NewManager(backends...), ephemeral, nil
}

func NewManager(keydir string) (am *accounts.Manager, err error) {
	if keydir == "" {
		// There is no datadir.
		keydir, err = ioutil.TempDir("", "go-okchain-keystore")
		logger.Infof("Using temp keystore dir[%s]\n", keydir)
	}

	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(keydir, 0700); err != nil {
		return nil, err
	}
	// Assemble the account manager and supported backends
	//软钱包
	backends := []accounts.Backend{
		keystore.NewKeyStore(keydir, keystore.StandardScryptN, keystore.StandardScryptP),
	}
	logger.Info("Setting accounts backends: keystore wallet")
	return accounts.NewManager(backends...), nil
}

func NewKeyStore(keydir string) (ks *keystore.KeyStore, err error) {
	if keydir == "" {
		// There is no datadir.
		keydir, err = ioutil.TempDir("", "go-okchain-keystore")
		logger.Infof("Using temp keystore dir[%s]\n", keydir)
	}
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(keydir, 0700); err != nil {
		return nil, err
	}
	// Assemble the account manager and supported backends
	//软钱包
	return keystore.NewKeyStore(keydir, keystore.StandardScryptN, keystore.StandardScryptP), nil
}
