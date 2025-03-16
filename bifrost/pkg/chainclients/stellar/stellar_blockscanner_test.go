package stellar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
	"gitlab.com/thorchain/thornode/bifrost/thorclient"
	"gitlab.com/thorchain/thornode/bifrost/tss/go-tss/tss"
	"gitlab.com/thorchain/thornode/config"
)

type BlockScannerTestSuite struct {
	suite.Suite
	thorKeys  *thorclient.Keys
	bridge    thorclient.ThorchainBridge
	m         *metrics.Metrics
	tssServer *tss.TssServer
}

func (s *BlockScannerTestSuite) SetUpSuite() {
	// Copy setup from StellarSuite
}

func TestBlockScannerSuite(t *testing.T) {
	suite.Run(t, new(BlockScannerTestSuite))
}

func (s *BlockScannerTestSuite) TestNewStellarBlockScanner() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	client, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	s.Require().NotNil(client)

	scanner, err := NewStellarBlockScanner(client)
	s.NoError(err)
	s.NotNil(scanner)
}

func (s *BlockScannerTestSuite) TestStellarBlockScannerHealth() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	client, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	require.NotNil(s.T(), client)

	scanner, err := NewStellarBlockScanner(client)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), scanner)

	assert.True(s.T(), scanner.IsHealthy())
}

func (s *BlockScannerTestSuite) TestStellarBlockScannerGetHeight() {
	cfg := config.BifrostChainConfiguration{
		ChainNetwork: "testnet",
	}
	client, err := NewStellarClient(s.thorKeys, cfg, s.tssServer, s.bridge, s.m)
	s.Require().NoError(err)
	require.NotNil(s.T(), client)

	scanner, err := NewStellarBlockScanner(client)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), scanner)

	height, err := scanner.GetHeight()
	assert.NoError(s.T(), err)
	assert.Greater(s.T(), height, int64(0))
}
