package app

// TODO: add tests below with thorchain's setup (non-typical cosmos chain)

import (
	"testing"

	// abci "github.com/cometbft/cometbft/abci/types"
	// dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	// "cosmossdk.io/log"

	// simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	// sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// func TestAppExport(t *testing.T) {
// 	cfg := sdk.GetConfig()
// 	cfg.SetBech32PrefixForAccount("thor", "thorpub")
// 	cfg.SetBech32PrefixForValidator("thorv", "thorvpub")
// 	cfg.SetBech32PrefixForConsensusNode("thorc", "thorcpub")
// 	cfg.SetCoinType(931)
// 	cfg.SetPurpose(44)
// 	cfg.Seal()
// 	sdk.SetCoinDenomRegex(func() string {
// 		return `[a-zA-Z][a-zA-Z0-9:\\/\\\-\\_\\.]{2,127}`
// 	})

// 	db := dbm.NewMemDB()
// 	logger := log.NewTestLogger(t)
// 	gapp := NewChainAppWithCustomOptions(t, false, SetupOptions{
// 		Logger:  logger.With("instance", "first"),
// 		DB:      db,
// 		AppOpts: simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
// 	})

// 	// finalize block so we have CheckTx state set
// 	_, err := gapp.FinalizeBlock(&abci.RequestFinalizeBlock{
// 		Height: 1,
// 	})
// 	require.NoError(t, err)

// 	_, err = gapp.Commit()
// 	require.NoError(t, err)

// 	// Making a new app object with the db, so that initchain hasn't been called
// 	newGapp := NewChainApp(
// 		logger, db, nil, true, simtestutil.NewAppOptionsWithFlagHome(t.TempDir()),
// 	)
// 	_, err = newGapp.ExportAppStateAndValidators(false, []string{}, nil)
// 	require.NoError(t, err, "ExportAppStateAndValidators should not have an error")
// }

// // ensure that blocked addresses are properly set in bank keeper
// func TestBlockedAddrs(t *testing.T) {
// 	gapp := Setup(t)

// 	for acc := range BlockedAddresses() {
// 		t.Run(acc, func(t *testing.T) {
// 			var addr sdk.AccAddress
// 			if modAddr, err := sdk.AccAddressFromBech32(acc); err == nil {
// 				addr = modAddr
// 			} else {
// 				addr = gapp.AccountKeeper.GetModuleAddress(acc)
// 			}
// 			require.True(t, gapp.BankKeeper.BlockedAddr(addr), "ensure that blocked addresses are properly set in bank keeper")
// 		})
// 	}
// }

func TestGetMaccPerms(t *testing.T) {
	dup := GetMaccPerms()
	require.Equal(t, maccPerms, dup, "duplicated module account permissions differed from actual module account permissions")
}

// TestMergedRegistry tests that fetching the gogo/protov2 merged registry
// doesn't fail after loading all file descriptors.
func TestMergedRegistry(t *testing.T) {
	r, err := proto.MergedRegistry()
	require.NoError(t, err)
	require.Greater(t, r.NumFiles(), 0)
}

func TestProtoAnnotations(t *testing.T) {
	r, err := proto.MergedRegistry()
	require.NoError(t, err)
	err = msgservice.ValidateProtoAnnotations(r)
	require.NoError(t, err)
}
