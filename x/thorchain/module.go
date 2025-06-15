package thorchain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cosmossdk.io/core/appmodule"
	"github.com/blang/semver"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkgrpc "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/module"
	gateway "github.com/cosmos/gogogateway"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"

	"gitlab.com/thorchain/thornode/v3/app/params"
	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/constants"

	"gitlab.com/thorchain/thornode/v3/x/thorchain/client/cli"
	"gitlab.com/thorchain/thornode/v3/x/thorchain/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleGenesis    = AppModule{}
	_ module.HasABCIGenesis      = AppModule{}
	_ module.HasServices         = AppModule{}
	_ module.HasABCIEndBlock     = AppModule{}
	_ module.HasConsensusVersion = AppModule{}
	_ appmodule.HasBeginBlocker  = AppModule{}
)

// AppModuleBasic app module Basics object
type AppModuleBasic struct{}

// Name return the module's name
func (AppModuleBasic) Name() string {
	return ModuleName
}

// RegisterLegacyAminoCodec registers the module's types for the given codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	RegisterInterfaces(reg)
}

// DefaultGenesis returns default genesis state as raw bytes for the module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(DefaultGenesis())
}

// ValidateGenesis check of the Genesis
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var data GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return err
	}
	// Once json successfully marshalled, passes along to genesis.go
	return ValidateGenesis(data)
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the mint module.
// thornode current doesn't have grpc endpoint yet
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetQueryCmd get the root query command of this module
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// GetTxCmd get the root tx command of this module
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// ____________________________________________________________________________

// AppModule implements an application module for the thorchain module.
type AppModule struct {
	AppModuleBasic
	mgr              *Mgrs
	telemetryEnabled bool
	msgServer        types.MsgServer
	queryServer      types.QueryServer
}

// NewAppModule creates a new AppModule Object
func NewAppModule(
	mgr *Mgrs,
	telemetryEnabled bool,
	testApp bool,
) AppModule {
	kb := cosmos.KeybaseStore{}
	var err error
	if !testApp {
		kb, err = cosmos.GetKeybase(os.Getenv(cosmos.EnvChainHome))
		if err != nil {
			panic(err)
		}
	}
	txConfig, err := params.TxConfig(mgr.cdc, nil)
	if err != nil {
		panic(fmt.Errorf("failed to create tx config: %w", err))
	}
	return AppModule{
		AppModuleBasic:   AppModuleBasic{},
		mgr:              mgr,
		telemetryEnabled: telemetryEnabled,
		msgServer:        NewMsgServerImpl(mgr),
		queryServer:      NewQueryServerImpl(mgr, txConfig, kb),
	}
}

func (AppModule) IsAppModule() {}

func (AppModule) IsOnePerModuleType() {}

func (AppModule) ConsensusVersion() uint64 {
	return 6
}

func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

func (am AppModule) QuerierRoute() string {
	return types.QuerierRoute
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), am.msgServer)
	types.RegisterQueryServer(cfg.QueryServer(), am.queryServer)
	wasmtypes.RegisterQueryServer(cfg.QueryServer(), wasmkeeper.Querier(&am.mgr.wasmKeeper))

	m := NewMigrator(am.mgr)
	if err := cfg.RegisterMigration(types.ModuleName, 5, m.Migrate5to6); err != nil {
		panic(fmt.Sprintf("failed to migrate x/thorchain from version 5 to 6: %v", err))
	}
}

// BeginBlock called when a block get proposed
func (am AppModule) BeginBlock(goCtx context.Context) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	votes := ctx.CometInfo().GetLastCommit().Votes()
	var existingValidators []string
	for i := range votes.Len() {
		v := votes.Get(i)
		addr := sdk.ValAddress(v.Validator().Address())
		existingValidators = append(existingValidators, addr.String())
	}

	ctx.Logger().Debug("BeginBlock", "height", ctx.BlockHeight())
	// Check/Update the network version before checking the local version
	if err := am.mgr.LoadManagerIfNecessary(ctx); err != nil {
		ctx.Logger().Error("fail to get managers", "error", err)
	}

	version := am.mgr.GetVersion()
	localVer := semver.MustParse(constants.SWVersion.String())
	if version.Major > localVer.Major || version.Minor > localVer.Minor {
		panic(fmt.Sprintf("Unsupported Version: update your binary (your version: %s, network consensus version: %s)", constants.SWVersion.String(), version.String()))
	}

	am.mgr.Keeper().ClearObservingAddresses(ctx)

	am.mgr.GasMgr().BeginBlock()
	if err := am.mgr.NetworkMgr().BeginBlock(ctx, am.mgr); err != nil {
		ctx.Logger().Error("fail to begin network manager", "error", err)
	}
	am.mgr.Slasher().BeginBlock(ctx, am.mgr.GetConstants())
	if err := am.mgr.ValidatorMgr().BeginBlock(ctx, am.mgr, existingValidators); err != nil {
		ctx.Logger().Error("Fail to begin block on validator", "error", err)
	}

	if err := am.mgr.Keeper().RemoveExpiredUpgradeProposals(ctx); err != nil {
		ctx.Logger().Error("Failed to remove expired upgrade proposals", "error", err)
	}
	return nil
}

// EndBlock called when a block get committed
func (am AppModule) EndBlock(goCtx context.Context) ([]abci.ValidatorUpdate, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.Logger().Debug("End Block", "height", ctx.BlockHeight())

	if err := am.mgr.SwapQ().EndBlock(ctx, am.mgr); err != nil {
		ctx.Logger().Error("fail to process swap queue", "error", err)
	}

	if err := am.mgr.AdvSwapQueueMgr().EndBlock(ctx, am.mgr); err != nil {
		ctx.Logger().Error("fail to process adv swap queue", "error", err)
	}

	if err := am.mgr.Slasher().LackSigning(ctx, am.mgr); err != nil {
		ctx.Logger().Error("Unable to slash for lack of signing:", "error", err)
	}

	if err := am.mgr.PoolMgr().EndBlock(ctx, am.mgr); err != nil {
		ctx.Logger().Error("fail to process pools", "error", err)
	}

	am.mgr.ObMgr().EndBlock(ctx, am.mgr.Keeper())

	// update network data to account for block rewards and reward units
	if err := am.mgr.NetworkMgr().UpdateNetwork(ctx, am.mgr.GetConstants(), am.mgr.GasMgr(), am.mgr.EventMgr()); err != nil {
		ctx.Logger().Error("fail to update network data", "error", err)
	}

	if err := am.mgr.NetworkMgr().EndBlock(ctx, am.mgr); err != nil {
		ctx.Logger().Error("fail to end block for vault manager", "error", err)
	}

	validators := am.mgr.ValidatorMgr().EndBlock(ctx, am.mgr)

	if err := am.mgr.TxOutStore().EndBlock(ctx, am.mgr); err != nil {
		ctx.Logger().Error("fail to process txout endblock", "error", err)
	}

	am.mgr.GasMgr().EndBlock(ctx, am.mgr.Keeper(), am.mgr.EventMgr())

	// telemetry
	if am.telemetryEnabled {
		if err := emitEndBlockTelemetry(ctx, am.mgr); err != nil {
			ctx.Logger().Error("unable to emit end block telemetry", "error", err)
		}
	}

	return validators, nil
}

// InitGenesis initialise genesis
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genState GenesisState
	ModuleCdc.MustUnmarshalJSON(data, &genState)
	return InitGenesis(ctx, am.mgr.Keeper(), genState)
}

// ExportGenesis export genesis
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.mgr.Keeper())
	return ModuleCdc.MustMarshalJSON(&gs)
}

// CustomGRPCGatewayRouter sets thorchain's custom GRPC gateway router
// Must be called before any GRPC gateway routes are registered
// GRPC gateway router settings are the same as cosmos sdk except for the additional
// serve mux option, WithMetadata().
func CustomGRPCGatewayRouter(apiSvr *api.Server) {
	clientCtx := apiSvr.ClientCtx

	// The default JSON marshaller used by the gRPC-Gateway is unable to marshal non-nullable non-scalar fields.
	// Using the gogo/gateway package with the gRPC-Gateway WithMarshaler option fixes the scalar field marshaling issue.
	marshalerOption := &gateway.JSONPb{
		EmitDefaults: true,
		Indent:       "",
		OrigName:     true,
		AnyResolver:  clientCtx.InterfaceRegistry,
	}

	apiSvr.GRPCGatewayRouter = runtime.NewServeMux(
		// Custom marshaler option is required for gogo proto
		runtime.WithMarshalerOption(runtime.MIMEWildcard, marshalerOption),

		// This is necessary to get error details properly
		// marshaled in unary requests.
		runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),

		// Custom header matcher for mapping request headers to
		// GRPC metadata
		runtime.WithIncomingHeaderMatcher(api.CustomGRPCHeaderMatcher),

		// This is necessary to be able to use the height query param for setting the correct state.
		// Cosmos sdk expect the GRPCBlockHeightHeader to be set if the latest height is not used.
		// This function will extract the height query param and set it in the metadata for the sdk to consume.
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			md := make(metadata.MD, 1)
			for key := range req.Header {
				// if the GRPCBlockHeightHeader is set, use that and ignore the height query parameter
				if key == sdkgrpc.GRPCBlockHeightHeader {
					return md
				}
			}
			// The following checked endpoint prefixes have the height query parameter extracted.
			if strings.HasPrefix(req.URL.Path, "/thorchain/") ||
				strings.HasPrefix(req.URL.Path, "/cosmos/") ||
				strings.HasPrefix(req.URL.Path, "/bank/balances/") ||
				strings.HasPrefix(req.URL.Path, "/auth/accounts/") {
				heightStr, ok := req.URL.Query()["height"]
				if ok && len(heightStr) > 0 {
					_, err := strconv.ParseInt(heightStr[0], 10, 64)
					// if a valid int, set the GRPCBlockHeightHeader, the query server will error later on invalid height params
					if err == nil {
						md.Set(sdkgrpc.GRPCBlockHeightHeader, heightStr...)
					}
				}
			}
			return md
		}),
	)
}
