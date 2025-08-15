package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/syndtr/goleveldb/leveldb/util"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/snapshots"
	snapshottypes "cosmossdk.io/store/snapshots/types"
	storetypes "cosmossdk.io/store/types"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	cmtcfg "github.com/cometbft/cometbft/config"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/cosmos/cosmos-sdk/server/types"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmcli "github.com/CosmWasm/wasmd/x/wasm/client/cli"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/switchlyprotocol/switchlynode/v3/app"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	thorlog "github.com/switchlyprotocol/switchlynode/v3/log"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/client/cli"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/ebifrost"

	"github.com/rs/zerolog"
)

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// The following code snippet is just for reference.

	type CustomAppConfig struct {
		serverconfig.Config
		Wasm     wasmtypes.WasmConfig    `mapstructure:"wasm"`
		EBifrost ebifrost.EBifrostConfig `mapstructure:"ebifrost"`
	}

	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In simapp, we set the min gas prices to 0.
	srvCfg.MinGasPrices = "0stake"
	// srvCfg.BaseConfig.IAVLDisableFastNode = true // disable fastnode by default

	customAppConfig := CustomAppConfig{
		Config: *srvCfg,
		Wasm:   wasmtypes.DefaultWasmConfig(),
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate
	customAppTemplate += wasmtypes.DefaultConfigTemplate()
	customAppTemplate += ebifrost.DefaultConfigTemplate()

	return customAppTemplate, customAppConfig
}

func initRootCmd(
	rootCmd *cobra.Command,
	txConfig client.TxConfig,
	interfaceRegistry codectypes.InterfaceRegistry,
	appCodec codec.Codec,
	basicManager module.BasicManager,
) {
	cfg := sdk.GetConfig()
	cfg.Seal()

	rootCmd.AddCommand(
		genutilcli.InitCmd(basicManager, app.DefaultNodeHome),
		//		NewTestnetCmd(basicManager, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(newApp, app.DefaultNodeHome),
		snapshot.Cmd(newApp),
		renderConfigCommand(),
		GetEd25519Keys(),
		GetPubKeyCmd(),
	)

	server.AddCommands(rootCmd, app.DefaultNodeHome, newApp, appExport, addModuleInitFlags)
	wasmcli.ExtendUnsafeResetAllCmd(rootCmd)

	// add keybase, auxiliary RPC, query, genesis, and tx child commands
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager),
		queryCommand(),
		txCommand(),
		cli.GetUtilCmd(),
		compactCommand(),
		keys.Commands(),
	)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	startCmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		serverCtx := server.GetServerContextFromCmd(cmd)

		// Bind flags to the Context's Viper so the app construction can set
		// options accordingly.
		if err := serverCtx.Viper.BindPFlags(cmd.Flags()); err != nil {
			return fmt.Errorf("fail to bind flags,err: %w", err)
		}

		// replace sdk logger with thorlog
		if zl, ok := serverCtx.Logger.Impl().(*zerolog.Logger); ok {
			logger := zl.With().CallerWithSkipFrameCount(3).Logger()
			serverCtx.Logger = thorlog.SdkLogWrapper{
				Logger: &logger,
			}
			return server.SetCmdServerContext(startCmd, serverCtx)
		}
		return nil
	}
	wasm.AddModuleInitFlags(startCmd)
	ebifrost.AddModuleInitFlags(startCmd)
}

func renderConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:                        "render-config",
		Short:                      "renders tendermint and cosmos config from switchlynode base config",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		Run: func(cmd *cobra.Command, args []string) {
			config.Init()
			config.InitSwitchly(cmd.Context())
		},
	}
}

// genesisCommand builds genesis-related `simd genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.Commands(txConfig, basicManager, app.DefaultNodeHome)

	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		server.QueryBlockCmd(),
		authcmd.QueryTxsByEventsCmd(),
		server.QueryBlocksCmd(),
		authcmd.QueryTxCmd(),
		server.QueryBlockResultsCmd(),
	)

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	return cmd
}

func compactCommand() *cobra.Command {
	return &cobra.Command{
		Use:                        "compact",
		Short:                      "force leveldb compaction",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE: func(cmd *cobra.Command, args []string) error {
			data := filepath.Join(app.DefaultNodeHome, "data")
			db, err := dbm.NewGoLevelDB("application", data, nil)
			if err != nil {
				return err
			}
			return db.DB().CompactRange(util.Range{})
		},
	}
}

// newApp creates the application
func newApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	appOpts servertypes.AppOptions,
) servertypes.Application {
	baseappOptions := DefaultBaseappOptions(appOpts)
	return app.NewChainApp(
		logger, db, traceStore, true, appOpts,
		[]wasm.Option{},
		baseappOptions...,
	)
}

func appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var switchlyApp *app.SwitchlyApp
	homePath, ok := appOpts.Get("home").(string)
	if !ok || homePath == "" {
		homePath = app.DefaultNodeHome
	}

	loadLatest := height == -1
	if loadLatest {
		switchlyApp = app.NewChainApp(logger, db, traceStore, loadLatest, appOpts, []wasm.Option{})

		if err := switchlyApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		switchlyApp = app.NewChainApp(logger, db, traceStore, false, appOpts, []wasm.Option{})
	}

	return switchlyApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

func DefaultBaseappOptions(appOpts types.AppOptions) []func(*baseapp.BaseApp) {
	var cache storetypes.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get("inter-block-cache")) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	pruningOpts, err := server.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	snapshotDir := filepath.Join(cast.ToString(appOpts.Get("home")), "data", "snapshots")
	snapshotDB, err := dbm.NewDB("metadata", dbm.GoLevelDBBackend, snapshotDir)
	if err != nil {
		panic(err)
	}
	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get("snapshot-interval")),
		cast.ToUint32(appOpts.Get("snapshot-keep-recent")),
	)

	homeDir := cast.ToString(appOpts.Get(flags.FlagHome))
	chainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if chainID == "" {
		// fallback to genesis chain-id
		reader, err2 := os.Open(filepath.Join(homeDir, "config", "genesis.json"))
		if err2 != nil {
			panic(err2)
		}
		defer reader.Close()

		chainID, err = genutiltypes.ParseChainIDFromGenesis(reader)
		if err != nil {
			panic(fmt.Errorf("failed to parse chain-id from genesis file: %w", err))
		}
	}

	baseappOptions := []func(*baseapp.BaseApp){
		baseapp.SetPruning(pruningOpts),
		baseapp.SetMinGasPrices(cast.ToString(appOpts.Get("minimum-gas-prices"))),
		baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get("halt-height"))),
		baseapp.SetHaltTime(cast.ToUint64(appOpts.Get("halt-time"))),
		baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get("min-retain-blocks"))),
		baseapp.SetInterBlockCache(cache),
		baseapp.SetTrace(cast.ToBool(appOpts.Get("trace"))),
		baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get("index-events"))),
		baseapp.SetSnapshot(snapshotStore, snapshotOptions),
		baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get("iavl-cache-size"))),
		baseapp.SetIAVLDisableFastNode(cast.ToBool(appOpts.Get("iavl-disable-fastnode"))),
	}

	// Add chain-id if we found it in genesis
	if chainID != "" {
		baseappOptions = append(baseappOptions, baseapp.SetChainID(chainID))
	}

	return baseappOptions
}
