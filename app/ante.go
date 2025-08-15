package app

import (
	"errors"

	corestoretypes "cosmossdk.io/core/store"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	switchly "github.com/switchlyprotocol/switchlynode/v3/x/switchly"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/ebifrost"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/keeper"
)

// HandlerOptions extend the SDK's AnteHandler options
type HandlerOptions struct {
	ante.HandlerOptions

	WasmConfig *wasmtypes.WasmConfig
	WasmKeeper *wasmkeeper.Keeper

	TXCounterStoreService corestoretypes.KVStoreService

	BypassMinFeeMsgTypes []string

	SWITCHLYChainKeeper keeper.Keeper
}

// NewAnteHandler constructor
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errors.New("account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return nil, errors.New("bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return nil, errors.New("sign mode handler is required for ante builder")
	}
	if options.WasmConfig == nil {
		return nil, errors.New("wasm config is required for ante builder")
	}
	if options.WasmKeeper == nil {
		return nil, errors.New("wasm keeper is required for ante builder")
	}
	if options.SWITCHLYChainKeeper == nil {
		return nil, errors.New("switchly keeper is required for ante builder")
	}
	if options.TXCounterStoreService == nil {
		return nil, errors.New("wasm store service is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		// must be first to ensure that injected txs bypass the remaining ante handlers, as they do not have gas.
		ebifrost.NewInjectedTxDecorator(),

		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first

		// replace gas meter immediately after setting up ctx
		switchly.NewGasDecorator(options.SWITCHLYChainKeeper),

		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),

		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		switchly.NewWasmExecuteAnteDecorator(options.SWITCHLYChainKeeper, options.AccountKeeper, options.BankKeeper),

		// run switchly-specific msg antes
		switchly.NewAnteDecorator(options.SWITCHLYChainKeeper),

		ante.NewSetPubKeyDecorator(options.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
