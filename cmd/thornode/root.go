package main

import (
	"os"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/spf13/cobra"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/switchlyprotocol/switchlynode/v1/app"
	"github.com/switchlyprotocol/switchlynode/v1/app/params"
	appparams "github.com/switchlyprotocol/switchlynode/v1/app/params"
	prefix "github.com/switchlyprotocol/switchlynode/v1/cmd"
	thorconfig "github.com/switchlyprotocol/switchlynode/v1/config"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// NewRootCmd creates a new root command for chain app. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(prefix.Bech32PrefixAccAddr, prefix.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(prefix.Bech32PrefixValAddr, prefix.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(prefix.Bech32PrefixConsAddr, prefix.Bech32PrefixConsPub)
	cfg.SetAddressVerifier(wasmtypes.VerifyAddressLen())
	cfg.SetCoinType(prefix.SwitchlyCoinType)
	cfg.SetPurpose(prefix.SwitchlyCoinPurpose)
	cfg.Seal()
	// we "pre"-instantiate the application for getting the injected/configured encoding configuration
	// note, this is not necessary when using app wiring, as depinject can be directly used (see root_v2.go)
	tempApp := app.NewChainApp(
		log.NewNopLogger(), dbm.NewMemDB(), nil, false, app.NewTestAppOptionsWithFlagHome(tempDir()),
		[]wasmkeeper.Option{},
	)
	encodingConfig := params.EncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.TxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}

	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithAccountRetriever(authtypes.AccountRetriever{}).
		WithHomeDir(app.DefaultNodeHome).
		WithViper("")

	rootCmd := &cobra.Command{
		Use:           version.AppName,
		Short:         version.AppName + " Daemon (server)",
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			thorconfig.Init()

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			var err error
			initClientCtx, err = client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL. This sign mode
			// is only available if the client is online.
			if !initClientCtx.Offline {
				txConfig, err2 := appparams.TxConfig(initClientCtx.Codec, txmodule.NewGRPCCoinMetadataQueryFn(initClientCtx))
				if err2 != nil {
					return err2
				}

				initClientCtx = initClientCtx.WithTxConfig(txConfig)
			}

			if err = client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			customAppTemplate, customAppConfig := initAppConfig()
			customCMTConfig := initCometBFTConfig()

			return server.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customCMTConfig)
		},
	}

	initRootCmd(rootCmd, encodingConfig.TxConfig, encodingConfig.InterfaceRegistry, encodingConfig.Codec, tempApp.BasicModuleManager)

	// add keyring to autocli opts
	autoCliOpts := tempApp.AutoCliOpts()
	initClientCtx, _ = config.ReadFromClientConfig(initClientCtx)
	autoCliOpts.Keyring, _ = keyring.NewAutoCLIKeyring(initClientCtx.Keyring)
	autoCliOpts.ClientCtx = initClientCtx

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd
}
