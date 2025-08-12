package stellar

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stellar/go/xdr"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/config"
)

type SorobanRPCTestSuite struct {
	client *SorobanRPCClient
	server *httptest.Server
}

var _ = Suite(&SorobanRPCTestSuite{})

func (s *SorobanRPCTestSuite) SetUpTest(c *C) {
	// Create mock HTTP server for testing (like other chain clients)
	s.server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Mock Soroban RPC responses
		switch req.URL.Path {
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
		RPCHost: s.server.URL, // Use mock server URL
	}
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
	s.client = NewSorobanRPCClient(cfg, logger, StellarTestnet)
}

func (s *SorobanRPCTestSuite) TearDownTest(c *C) {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *SorobanRPCTestSuite) TestNewSorobanRPCClient(c *C) {
	c.Assert(s.client, NotNil)
	c.Assert(s.client.networkType, Equals, StellarTestnet)

	// Check that the URL is properly configured with the mock server
	c.Assert(s.client.rpcURL, Not(Equals), "")
	c.Assert(strings.Contains(s.client.rpcURL, "/soroban/rpc"), Equals, true)
}

func (s *SorobanRPCTestSuite) TestSorobanRPCClientWithConfiguredHost(c *C) {
	// Test with Docker environment style configuration
	cfg := config.BifrostChainConfiguration{
		RPCHost: "http://stellar:8000", // Docker environment example
	}
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// Test that it appends /soroban/rpc path
	client := NewSorobanRPCClient(cfg, logger, StellarTestnet)
	c.Assert(client.rpcURL, Equals, "http://stellar:8000/soroban/rpc")

	// Test that it doesn't double-append if path already exists
	cfg.RPCHost = "http://stellar:8000/soroban/rpc"
	client = NewSorobanRPCClient(cfg, logger, StellarTestnet)
	c.Assert(client.rpcURL, Equals, "http://stellar:8000/soroban/rpc")

	// Test with localhost configuration
	cfg.RPCHost = "http://localhost:8000"
	client = NewSorobanRPCClient(cfg, logger, StellarTestnet)
	c.Assert(client.rpcURL, Equals, "http://localhost:8000/soroban/rpc")
}

func (s *SorobanRPCTestSuite) TestSorobanRPCClientDefaultURL(c *C) {
	cfg := config.BifrostChainConfiguration{
		RPCHost: "", // Empty to test default URL
	}
	logger := zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()

	// Test testnet default
	client := NewSorobanRPCClient(cfg, logger, StellarTestnet)
	c.Assert(client.rpcURL, Equals, "https://soroban-testnet.stellar.org")

	// Test mainnet default
	client = NewSorobanRPCClient(cfg, logger, StellarMainnet)
	c.Assert(client.rpcURL, Equals, "https://soroban-mainnet.stellar.org")
}

func (s *SorobanRPCTestSuite) TestScValToString(c *C) {
	// Test boolean values
	boolVal := xdr.ScVal{
		Type: xdr.ScValTypeScvBool,
		B:    &[]bool{true}[0],
	}
	result := s.client.scValToString(boolVal)
	c.Assert(result, Equals, "true")

	// Test void value
	voidVal := xdr.ScVal{
		Type: xdr.ScValTypeScvVoid,
	}
	result = s.client.scValToString(voidVal)
	c.Assert(result, Equals, "")

	// Test string value
	str := "test string"
	stringVal := xdr.ScVal{
		Type: xdr.ScValTypeScvString,
		Str:  (*xdr.ScString)(&str),
	}
	result = s.client.scValToString(stringVal)
	c.Assert(result, Equals, "test string")

	// Test symbol value
	sym := "test_symbol"
	symbolVal := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  (*xdr.ScSymbol)(&sym),
	}
	result = s.client.scValToString(symbolVal)
	c.Assert(result, Equals, "test_symbol")

	// Test U32 value
	u32 := xdr.Uint32(42)
	u32Val := xdr.ScVal{
		Type: xdr.ScValTypeScvU32,
		U32:  &u32,
	}
	result = s.client.scValToString(u32Val)
	c.Assert(result, Equals, "42")

	// Test I32 value
	i32 := xdr.Int32(-42)
	i32Val := xdr.ScVal{
		Type: xdr.ScValTypeScvI32,
		I32:  &i32,
	}
	result = s.client.scValToString(i32Val)
	c.Assert(result, Equals, "-42")
}

func (s *SorobanRPCTestSuite) TestParseContractEvent(c *C) {
	// Create valid XDR-encoded data for testing
	// Create a symbol ScVal for the event topic (e.g., "deposit")
	eventSym := "deposit"
	topicScVal := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  (*xdr.ScSymbol)(&eventSym),
	}
	topicXDR, err := xdr.MarshalBase64(topicScVal)
	c.Assert(err, IsNil)

	// Create a map ScVal for the event value (simulating our deposit event structure)
	amountKey := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  (*xdr.ScSymbol)(&[]string{"amount"}[0]),
	}
	amountVal := xdr.ScVal{
		Type: xdr.ScValTypeScvString,
		Str:  (*xdr.ScString)(&[]string{"100"}[0]),
	}

	assetKey := xdr.ScVal{
		Type: xdr.ScValTypeScvSymbol,
		Sym:  (*xdr.ScSymbol)(&[]string{"asset"}[0]),
	}
	assetVal := xdr.ScVal{
		Type: xdr.ScValTypeScvString,
		Str:  (*xdr.ScString)(&[]string{"XLM"}[0]),
	}

	// Create map entries
	mapEntries := []xdr.ScMapEntry{
		{Key: amountKey, Val: amountVal},
		{Key: assetKey, Val: assetVal},
	}
	scMap := xdr.ScMap(mapEntries)
	scMapPtr := &scMap

	valueScVal := xdr.ScVal{
		Type: xdr.ScValTypeScvMap,
		Map:  &scMapPtr,
	}
	valueXDR, err := xdr.MarshalBase64(valueScVal)
	c.Assert(err, IsNil)

	// Test parsing a contract event with valid XDR data
	event := ContractEvent{
		Type:            "contract",
		Ledger:          12345,
		LedgerTime:      "2023-01-01T00:00:00Z",
		ContractID:      "test-contract-id",
		ID:              "test-event-id",
		Topic:           []string{topicXDR},
		Value:           valueXDR,
		TransactionHash: "test-tx-hash",
	}

	routerEvent, err := s.client.ParseContractEvent(event)
	c.Assert(err, IsNil)
	c.Assert(routerEvent, NotNil)
	c.Assert(routerEvent.ContractAddress, Equals, "test-contract-id")
	c.Assert(routerEvent.TransactionHash, Equals, "test-tx-hash")
	c.Assert(routerEvent.Ledger, Equals, uint32(12345))
	c.Assert(routerEvent.Type, Equals, "deposit")
	c.Assert(routerEvent.Amount, Equals, "100")
	c.Assert(routerEvent.Asset, Equals, "XLM")
}

func (s *SorobanRPCTestSuite) TestIsRouterEvent(c *C) {
	event := ContractEvent{
		ContractID: "test-contract-id",
	}

	// Test positive case
	result := s.client.IsRouterEvent(event, []string{"test-contract-id", "other-contract"})
	c.Assert(result, Equals, true)

	// Test negative case
	result = s.client.IsRouterEvent(event, []string{"other-contract", "another-contract"})
	c.Assert(result, Equals, false)

	// Test empty router addresses
	result = s.client.IsRouterEvent(event, []string{})
	c.Assert(result, Equals, false)
}

func TestSorobanRPCClient(t *testing.T) {
	Suite(&SorobanRPCTestSuite{})
}
