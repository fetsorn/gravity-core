package main

import (
	"context"
	"flag"
	"fmt"
	"gravity-hub/ledger-node/app"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/wavesplatform/gowaves/pkg/client"

	"github.com/dgraph-io/badger"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/spf13/viper"

	cfg "github.com/tendermint/tendermint/config"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	nm "github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
)

var configFile, db string

func init() {
	flag.StringVar(&db, "db", "./db", "Path to config.toml")
	flag.StringVar(&configFile, "config", "./data/config/config.toml", "Path to config.toml")
	flag.Parse()
}

func main() {
	flag.Parse()
	db, err := badger.Open(badger.DefaultOptions(db).WithTruncate(true))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open badger db: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	node, err := newTendermint(db, configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(2)
	}

	node.Start()
	defer func() {
		node.Stop()
		node.Wait()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(0)
}

func newTendermint(db *badger.DB, configFile string) (*nm.Node, error) {

	// read config
	config := cfg.DefaultConfig()
	config.RootDir = filepath.Dir(filepath.Dir(configFile))
	viper.SetConfigFile(configFile)
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("viper failed to read config file: %w", err)
	}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("viper failed to unmarshal config: %w", err)
	}
	if err := config.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	ctx := context.Background()
	ethClient, err := ethclient.DialContext(ctx, viper.GetString("ethNodeUrl"))
	if err != nil {
		return nil, err
	}

	wavesClient, err := client.NewClient(client.Options{ApiKey: "", BaseUrl: viper.GetString("wavesNodeUrl")})
	if err != nil {
		return nil, err
	}
	app := app.NewGHApplication(ethClient, wavesClient, db, ctx)

	// create logger
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	logger, err = tmflags.ParseLogLevel(config.LogLevel, logger, cfg.DefaultLogLevel())
	if err != nil {
		return nil, fmt.Errorf("failed to parse log level: %w", err)
	}

	// read private validator
	pv := privval.LoadFilePV(
		config.PrivValidatorKeyFile(),
		config.PrivValidatorStateFile(),
	)

	// read node key
	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		return nil, fmt.Errorf("failed to load node's key: %w", err)
	}

	// create node
	node, err := nm.NewNode(
		config,
		pv,
		nodeKey,
		proxy.NewLocalClientCreator(app),
		nm.DefaultGenesisDocProviderFunc(config),
		nm.DefaultDBProvider,
		nm.DefaultMetricsProvider(config.Instrumentation),
		logger)

	if err != nil {
		return nil, fmt.Errorf("failed to create new Tendermint node: %w", err)
	}

	return node, nil
}