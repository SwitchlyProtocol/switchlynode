package switchlyclient

import (
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	hd "github.com/cosmos/cosmos-sdk/crypto/hd"
	cKeys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly"
)

var m *metrics.Metrics

func SetupSwitchlyForTest(c *C) (config.BifrostClientConfiguration, *cKeys.Record, cKeys.Keyring) {
	switchly.SetupConfigForTest()
	cfg := config.BifrostClientConfiguration{
		ChainID:         "switchly",
		ChainHost:       "localhost",
		ChainRPC:        "localhost",
		SignerName:      "switchly",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kb := cKeys.NewInMemory(cdc)

	params := *hd.NewFundraiserParams(0, sdk.CoinType, 0)
	hdPath := params.String()

	// create a consistent user
	record, err := kb.NewAccount(cfg.SignerName, "industry segment educate height inject hover bargain offer employ select speak outer video tornado story slow chief object junk vapor venue large shove behave", cfg.SignerPasswd, hdPath, hd.Secp256k1)
	c.Assert(err, IsNil)

	return cfg, record, kb
}

func GetMetricForTest(c *C) *metrics.Metrics {
	if m == nil {
		var err error
		m, err = metrics.NewMetrics(config.BifrostMetricsConfiguration{
			Enabled:      false,
			ListenPort:   9000,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			Chains:       common.Chains{common.ETHChain},
		})
		c.Assert(m, NotNil)
		c.Assert(err, IsNil)
	}
	return m
}
