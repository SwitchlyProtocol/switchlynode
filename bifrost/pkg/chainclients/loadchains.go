package chainclients

import (
	"time"

	"github.com/rs/zerolog/log"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/tss"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/ethereum"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/evm"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/gaia"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/shared/types"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/stellar"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/utxo"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/xrp"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pubkeymanager"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
)

// ChainClient exports the shared type.
type ChainClient = types.ChainClient

// LoadChains returns chain clients from chain configuration
func LoadChains(thorKeys *thorclient.Keys,
	cfg map[common.Chain]config.BifrostChainConfiguration,
	server *tss.TssServer,
	thorchainBridge thorclient.ThorchainBridge,
	m *metrics.Metrics,
	pubKeyValidator pubkeymanager.PubKeyValidator,
	poolMgr thorclient.PoolManager,
) (chains map[common.Chain]ChainClient, restart chan struct{}) {
	logger := log.Logger.With().Str("module", "bifrost").Logger()

	chains = make(map[common.Chain]ChainClient)
	restart = make(chan struct{})
	failedChains := []common.Chain{}

	loadChain := func(chain config.BifrostChainConfiguration) (ChainClient, error) {
		switch chain.ChainID {
		case common.ETHChain:
			return ethereum.NewClient(thorKeys, chain, server, thorchainBridge, m, pubKeyValidator, poolMgr)
		case common.AVAXChain, common.BSCChain, common.BASEChain:
			return evm.NewEVMClient(thorKeys, chain, server, thorchainBridge, m, pubKeyValidator, poolMgr)
		case common.GAIAChain:
			return gaia.NewCosmosClient(thorKeys, chain, server, thorchainBridge, m)
		case common.BTCChain, common.BCHChain, common.LTCChain, common.DOGEChain:
			return utxo.NewClient(thorKeys, chain, server, thorchainBridge, m)
		case common.XRPChain:
			return xrp.NewClient(thorKeys, chain, server, thorchainBridge, m)
		case common.StellarChain:
			return stellar.NewClient(thorKeys, chain, server, thorchainBridge, m)
		default:
			log.Fatal().Msgf("chain %s is not supported", chain.ChainID)
			return nil, nil
		}
	}

	for _, chain := range cfg {
		if chain.Disabled {
			logger.Info().Msgf("%s chain is disabled by configure", chain.ChainID)
			continue
		}

		client, err := loadChain(chain)
		if err != nil {
			logger.Debug().Err(err).Stringer("chain", chain.ChainID).Msg("failed to load chain")
			failedChains = append(failedChains, chain.ChainID)
			continue
		}

		// trunk-ignore-all(golangci-lint/forcetypeassert)
		switch chain.ChainID {
		case common.BTCChain, common.BCHChain, common.LTCChain, common.DOGEChain:
			pubKeyValidator.RegisterCallback(client.(*utxo.Client).RegisterPublicKey)
		}
		chains[chain.ChainID] = client
	}

	// watch failed chains minutely and restart bifrost if any succeed init
	if len(failedChains) > 0 {
		go func() {
			tick := time.NewTicker(time.Minute)
			for range tick.C {
				for _, chain := range failedChains {
					ccfg := cfg[chain]
					ccfg.BlockScanner.DBPath = "" // in-memory db

					_, err := loadChain(ccfg)
					if err == nil {
						logger.Info().Stringer("chain", chain).Msg("chain loaded, restarting bifrost")
						close(restart)
						return
					} else {
						logger.Debug().Err(err).Stringer("chain", chain).Msg("failed to load chain")
					}
				}
			}
		}()
	}

	return chains, restart
}
