package stellar

import (
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/rs/zerolog"
	"github.com/stellar/go/clients/horizonclient"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/config"
)

type RouterIntegrationTestSuite struct {
	client        *Client
	routerScanner *RouterEventScanner
	bridge        *MockThorchainBridge
	server        *httptest.Server
}

var _ = Suite(&RouterIntegrationTestSuite{})

func (s *RouterIntegrationTestSuite) SetUpTest(c *C) {
	s.bridge = &MockThorchainBridge{}

	// Create mock HTTP server for testing (like other chain clients)
	s.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Mock Stellar Horizon API responses
		switch req.URL.Path {
		case "/":
			// Basic health check
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"horizon_version": "2.0.0"}`))
		case "/soroban/rpc":
			// Mock Soroban RPC responses
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{"jsonrpc": "2.0", "id": 1, "result": {}}`))
		default:
			// Default mock response
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`{}`))
		}
	}))

	cfg := config.BifrostChainConfiguration{
		ChainID:      common.StellarChain,
		ChainNetwork: "testnet",
		RPCHost:      s.server.URL, // Use mock server URL
		BlockScanner: config.BifrostBlockScannerConfiguration{
			ChainID:                    common.StellarChain,
			HTTPRequestTimeout:         30 * time.Second,
			HTTPRequestReadTimeout:     30 * time.Second,
			HTTPRequestWriteTimeout:    30 * time.Second,
			MaxHTTPRequestRetry:        3,
			BlockHeightDiscoverBackoff: 5 * time.Second,
			BlockRetryInterval:         10 * time.Second,
			DBPath:                     "", // Use in-memory storage for tests
		},
		ScannerLevelDB: config.LevelDBOptions{},
	}

	// Create a simple client without full initialization for testing
	horizonClient := &horizonclient.Client{
		HorizonURL: cfg.RPCHost,
	}

	// Create Soroban RPC client for testing
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	sorobanRPCClient := NewSorobanRPCClient(cfg, logger, StellarTestnet)

	client := &Client{
		cfg:             cfg,
		thorchainBridge: s.bridge,
		horizonClient:   horizonClient,
		routerAddress:   "",
	}
	s.client = client

	// Create router scanner with Soroban RPC client
	s.routerScanner = NewRouterEventScanner(
		cfg.BlockScanner,
		horizonClient,
		sorobanRPCClient,
		"test-router-address",
	)
}

func (s *RouterIntegrationTestSuite) TearDownTest(c *C) {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *RouterIntegrationTestSuite) TestRouterEventScanner(c *C) {
	c.Assert(s.routerScanner, NotNil)
	c.Assert(s.routerScanner.routerAddress, Equals, "test-router-address")

	// Test scanning for router events (should now work with mock server)
	events, err := s.routerScanner.ScanRouterEvents(1)
	c.Assert(err, IsNil)
	// Should return empty events from mock server (can be nil or empty slice)
	if events != nil {
		c.Assert(len(events), Equals, 0)
	}
}

func (s *RouterIntegrationTestSuite) TestRouterAwareScanner(c *C) {
	// Create router-aware scanner
	routerAwareScanner := NewRouterAwareStellarScanner(s.client.stellarScanner, s.routerScanner)
	c.Assert(routerAwareScanner, NotNil)
	c.Assert(routerAwareScanner.routerScanner, Equals, s.routerScanner)
}

func (s *RouterIntegrationTestSuite) TestRouterConfiguration(c *C) {
	// Test router configuration
	config, err := s.client.LoadRouterConfig()
	c.Assert(err, IsNil)
	c.Assert(config, NotNil)
	c.Assert(config.Version, Equals, "1.0.0")

	// Test saving router config
	config.Address = "new-router-address"
	err = s.client.SaveRouterConfig(config)
	c.Assert(err, IsNil)
	c.Assert(s.client.routerAddress, Equals, "new-router-address")
}

func (s *RouterIntegrationTestSuite) TestRouterDeployment(c *C) {
	pubKey := common.EmptyPubKey

	// Test router deployment (should fail as not implemented)
	_, err := s.client.DeployRouter(pubKey)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "router deployment not yet implemented for Stellar")

	// Test router deployment check
	isDeployed := s.client.IsRouterDeployed(pubKey)
	c.Assert(isDeployed, Equals, false)
}

func (s *RouterIntegrationTestSuite) TestRouterHealthMonitoring(c *C) {
	// Test router health monitoring
	err := s.client.MonitorRouterHealth()
	c.Assert(err, NotNil) // Should fail as no router configured

	// Set router address and test again
	s.client.routerAddress = "test-router-address"
	err = s.client.MonitorRouterHealth()
	c.Assert(err, IsNil)

	// Test router version
	version, err := s.client.GetRouterVersion()
	c.Assert(err, IsNil)
	c.Assert(version, Equals, "1.0.0")
}

func (s *RouterIntegrationTestSuite) TestRouterAddressManagement(c *C) {
	pubKey := common.EmptyPubKey

	// Test getting router address (should fail as not configured)
	_, err := s.client.GetRouterAddress(pubKey)
	c.Assert(err, NotNil)

	// Test updating router address
	newAddr := common.Address("new-router-address")
	err = s.client.UpdateRouterAddress(pubKey, newAddr)
	c.Assert(err, IsNil)
	c.Assert(s.client.routerAddress, Equals, newAddr.String())
}
