package stellar

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	stypes "gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/bifrost/tss/go-tss/tss"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/config"
)

type StellarSuite struct {
	suite.Suite
	thorKeys  *thorclient.Keys
	bridge    thorclient.ThorchainBridge
	m         *metrics.Metrics
	server    *httptest.Server
	tssServer *tss.TssServer
}

func TestStellarSuite(t *testing.T) {
	suite.Run(t, new(StellarSuite))
}

func (s *StellarSuite) SetUpSuite() {
	var err error
	s.m, err = metrics.NewMetrics(config.BifrostMetricsConfiguration{
		Enabled:      false,
		ListenPort:   9000,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		Chains:       []common.Chain{common.THORChain},
	})
	s.Require().NoError(err)

	s.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch {
		case req.RequestURI == thorclient.PubKeysEndpoint:
			httpTestHandler(s.T(), rw, "../../../../test/fixtures/endpoints/vaults/pubKeys.json")
		case req.RequestURI == thorclient.InboundAddressesEndpoint:
			httpTestHandler(s.T(), rw, "../../../../test/fixtures/endpoints/inbound_addresses/inbound_addresses.json")
		case strings.HasPrefix(req.RequestURI, "/ledgers"):
			_, err := rw.Write([]byte(`{
				"_embedded": {
					"records": [{
						"sequence": 1234,
						"successful_transaction_count": 1
					}]
				}
			}`))
			s.Require().NoError(err)
		}
	}))

	cfg := config.BifrostClientConfiguration{
		ChainID:         "thorchain",
		ChainHost:       s.server.Listener.Addr().String(),
		SignerName:      "bob",
		SignerPasswd:    "password",
		ChainHomeFolder: "",
	}

	s.bridge, err = thorclient.NewThorchainBridge(cfg, s.m, s.thorKeys)
	s.Require().NoError(err)

	// Add mock TSS server
	s.tssServer = &tss.TssServer{}
}

func (s *StellarSuite) TearDownSuite() {
	s.server.Close()
}

func (s *StellarSuite) TestNewClient() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}

	// Test with invalid network
	cfg.ChainNetwork = "invalid"
	c, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Error(err)
	s.Nil(c)

	// Test with valid network
	cfg.ChainNetwork = "testnet"
	c, err = NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.NoError(err)
	s.NotNil(c)
}

func (s *StellarSuite) TestGetHeight() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	c, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	s.Require().NotNil(c)

	height, err := c.GetHeight()
	s.NoError(err)
	s.Greater(height, int64(0))
}

func (s *StellarSuite) TestGetAccountByAddress() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	c, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	s.Require().NotNil(c)

	acct, err := c.GetAccountByAddress("GCDQCPSWUXC3RD35T2XYCLQVNQWJMO3QJQT7AXQBFD6H73R2EIUKCXZ", nil)
	s.NoError(err)
	s.NotNil(acct)
}

func (s *StellarSuite) TestConfirmationCountReady() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	c, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	s.Require().NotNil(c)

	txIn := stypes.TxIn{
		Chain:   common.STELLARChain,
		TxArray: []stypes.TxInItem{},
	}
	s.True(c.ConfirmationCountReady(txIn))
}

func (s *StellarSuite) TestSignTx() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	c, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	s.Require().NotNil(c)

	// Test invalid chain
	tx := stypes.TxOutItem{
		Chain: common.BTCChain,
	}
	_, _, _, err = c.SignTx(tx, 1)
	s.Error(err)

	// Test empty memo
	tx = stypes.TxOutItem{
		Chain: common.STELLARChain,
	}
	_, _, _, err = c.SignTx(tx, 1)
	s.Error(err)

	// Test successful signing
	tx = stypes.TxOutItem{
		Chain:       common.STELLARChain,
		ToAddress:   common.Address("GBZFRQE42G2ULRFFITXP2UZAXRBYKQM7R7LZ3QS7YHDUUI5QQRHGBZCY"),
		VaultPubKey: GetRandomPubKey(),
		Coins: common.Coins{
			common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000000)),
		},
		Memo: "MEMO",
	}

	signedTx, _, _, err := c.SignTx(tx, 1)
	s.Require().NoError(err)
	s.NotEmpty(signedTx)

	// Test broadcast
	txHash, err := c.BroadcastTx(tx, signedTx)
	s.Require().NoError(err)
	s.NotEmpty(txHash)
}

func (s *StellarSuite) TestBroadcastTx() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	c, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	s.Require().NotNil(c)

	// Test empty signed tx
	_, err = c.BroadcastTx(stypes.TxOutItem{}, nil)
	s.Error(err)
}

func httpTestHandler(t *testing.T, rw http.ResponseWriter, fixture string) {
	content, err := os.ReadFile(fixture)
	require.NoError(t, err)
	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(content)
	require.NoError(t, err)
}

func GetRandomPubKey() common.PubKey {
	// Generate random 32 bytes
	bytes := make([]byte, 32)
	rand.Read(bytes)
	pk, err := common.NewPubKey(hex.EncodeToString(bytes))
	if err != nil {
		panic(err)
	}
	return pk
}
