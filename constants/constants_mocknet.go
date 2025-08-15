//go:build mocknet
// +build mocknet

// For internal testing and mockneting
package constants

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

func camelToSnakeUpper(s string) string {
	re := regexp.MustCompile(`([a-z0-9])([A-Z])|([A-Z]+)([A-Z][a-z])`)
	snake := re.ReplaceAllString(s, `${1}${3}_${2}${4}`)
	return strings.ToUpper(snake)
}

func init() {
	int64Overrides = map[ConstantName]int64{
		// ArtificialRagnarokBlockHeight: 200,
		DesiredValidatorSet:                 4,
		ChurnInterval:                       60,
		ChurnRetryInterval:                  30,
		MinimumBondInSWITCH:                 100_000_000, // 1 rune
		MinimumNodesForBFT:                  3, // Minimum nodes for Byzantine Fault Tolerance
		ValidatorMaxRewardRatio:             3,
		FundMigrationInterval:               15,
		LiquidityLockUpBlocks:               0,
		MaxSWITCHSupply:                     500_000_000_00000000,
		JailTimeKeygen:                      10,
		JailTimeKeysign:                     10,
		AsgardSize:                          4,
		StreamingSwapMinBPFee:               100, // TODO: remove on hard fork
		VirtualMultSynthsBasisPoints:        20_000,
		MinTxOutVolumeThreshold:             2000000_00000000,
		MissingBlockChurnOut:                100,
		MaxMissingBlockChurnOut:             5,
		TxOutDelayRate:                      2000000_00000000,
		MaxSynthPerPoolDepth:                3_500,
		MaxSynthsForSaversYield:             5000,
		PauseLoans:                          0,
		AllowWideBlame:                      1,
		TargetOutboundFeeSurplusSWITCH:      10_000_00000000,
		MaxOutboundFeeMultiplierBasisPoints: 20_000,
		MinOutboundFeeMultiplierBasisPoints: 15_000,
		OperationalVotesMin:                 1, // For regtest single-signer Mimir changes without Admin
		PreferredAssetOutboundFeeMultiplier: 100,
		TradeAccountsEnabled:                1,
		MaxAffiliateFeeBasisPoints:          10_000,
		SWITCHPoolDepositMaturityBlocks:     0,
		SWITCHPoolMaxReserveBackstop:        0,
		SaversEjectInterval:                 60,
		SystemIncomeBurnRateBps:             0,
		DevFundSystemIncomeBps:              0,
		SWCYStakeSystemIncomeBps:            0,
		MultipleAffiliatesMaxCount:          5,
		BankSendEnabled:                     1,
	}
	boolOverrides = map[ConstantName]bool{
		StrictBondLiquidityRatio: false,
	}
	stringOverrides = map[ConstantName]string{
		DefaultPoolStatus: "Available",
		DevFundAddress:    "switch1qhy9zkhtwxrma0epm0a7ln0lz3hq5vc5p6f3dn", // addr_thor_pig in regtest
	}

	v1Values := NewConstantValue()

	// allow overrides from environment variables in mocknet
	for k := range v1Values.int64values {
		env := camelToSnakeUpper(k.String())
		if os.Getenv(env) != "" {
			int64Overrides[k], _ = strconv.ParseInt(os.Getenv(env), 10, 64)
		}
	}
	for k := range v1Values.boolValues {
		env := camelToSnakeUpper(k.String())
		if os.Getenv(env) != "" {
			boolOverrides[k], _ = strconv.ParseBool(os.Getenv(env))
		}
	}
	for k := range v1Values.stringValues {
		env := camelToSnakeUpper(k.String())
		if os.Getenv(env) != "" {
			stringOverrides[k] = os.Getenv(env)
		}
	}
}
