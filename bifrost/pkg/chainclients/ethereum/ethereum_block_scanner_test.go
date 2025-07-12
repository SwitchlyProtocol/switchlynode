package ethereum

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cKeys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	etypes "github.com/ethereum/go-ethereum/core/types"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v1/cmd"
	"github.com/switchlyprotocol/switchlynode/v1/config"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain"
)

const Mainnet = 1

type BlockScannerTestSuite struct {
	m      *metrics.Metrics
	bridge thorclient.ThorchainBridge
	keys   *thorclient.Keys
}

var _ = Suite(&BlockScannerTestSuite{})

func (s *BlockScannerTestSuite) SetUpSuite(c *C) {
	thorchain.SetupConfigForTest()
	s.m = GetMetricForTest(c)
	c.Assert(s.m, NotNil)
	cfg := config.BifrostClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       "localhost",
		SignerName:      "bob",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kb := cKeys.NewInMemory(cdc)
	_, _, err := kb.NewMnemonic(cfg.SignerName, cKeys.English, cmd.SwitchlyProtocolHDPath, cfg.SignerPasswd, hd.Secp256k1)
	c.Assert(err, IsNil)
	thorKeys := thorclient.NewKeysWithKeybase(kb, cfg.SignerName, cfg.SignerPasswd)
	c.Assert(err, IsNil)
	s.keys = thorKeys
	s.bridge, err = thorclient.NewThorchainBridge(cfg, s.m, thorKeys)
	c.Assert(err, IsNil)
}

func getConfigForTest() config.BifrostBlockScannerConfiguration {
	return config.BifrostBlockScannerConfiguration{
		StartBlockHeight:           1, // avoids querying thorchain for block height
		BlockScanProcessors:        1,
		HTTPRequestTimeout:         time.Second,
		HTTPRequestReadTimeout:     time.Second * 30,
		HTTPRequestWriteTimeout:    time.Second * 30,
		MaxHTTPRequestRetry:        3,
		BlockHeightDiscoverBackoff: time.Second,
		BlockRetryInterval:         time.Second,
		Concurrency:                1,
		GasCacheBlocks:             40,
		GasPriceResolution:         10_000_000_000,
	}
}

func CreateBlock(height int) (*etypes.Header, error) {
	strHeight := fmt.Sprintf("%x", height)
	blockJson := `{
               "parentHash":"0x8b535592eb3192017a527bbf8e3596da86b3abea51d6257898b2ced9d3a83826",
               "difficulty": "0x31962a3fc82b",
               "extraData": "0x4477617266506f6f6c",
               "gasLimit": "0x47c3d8",
               "gasUsed": "0x0",
               "hash": "0x78bfef68fccd4507f9f4804ba5c65eb2f928ea45b3383ade88aaa720f1209cba",
               "logsBloom": "0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
               "miner": "0x52bc44d5378309ee2abf1539bf71de1b7d7be3b5",
               "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
               "nonce": "0x0000000000000000",
               "number": "0x` + strHeight + `",
               "receiptsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
               "sha3Uncles": "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347",
               "size": "0x218",
               "stateRoot": "0x0000000000000000000000000000000000000000000000000000000000000000",
               "timestamp": "0x55ba4224",
               "totalDifficulty": "0x78ed983323d",
               "transactions": [],
               "transactionsRoot": "0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421",
               "uncles": []
           }`

	var header etypes.Header
	if err := json.Unmarshal([]byte(blockJson), &header); err != nil {
		return nil, err
	}
	return &header, nil
}
