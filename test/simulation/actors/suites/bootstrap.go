package suites

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/test/simulation/actors/core"
	"github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/evm"
	"github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/thornode"
	. "github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/types"
)

////////////////////////////////////////////////////////////////////////////////////////
// Bootstrap
////////////////////////////////////////////////////////////////////////////////////////

func Bootstrap() *Actor {
	a := NewActor("Bootstrap")

	pools, err := thornode.GetPools()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get pools")
	}

	// bootstrap pools for all chains
	count := 0
	for _, chain := range common.AllChains {
		if chain == common.THORChain {
			continue
		}
		// BSC not compatible with sim tests
		if chain.Equals(common.BSCChain) {
			continue
		}
		count++

		a.Children[core.NewDualLPActor(chain.GetGasAsset())] = true
	}

	// create token pools
	tokenPools := NewActor("Bootstrap-TokenPools")
	for _, chain := range common.AllChains {
		if !chain.IsEVM() {
			continue
		}
		// BSC not compatible with sim tests
		if chain.Equals(common.BSCChain) {
			continue
		}
		count++

		for asset := range evm.Tokens(chain) {
			tokenPools.Children[core.NewDualLPActor(asset)] = true
		}
	}
	a.Append(tokenPools)

	// verify pools
	verify := NewActor("Bootstrap-Verify")
	verify.Ops = append(verify.Ops, func(config *OpConfig) OpResult {
		pools, err = thornode.GetPools()
		if err != nil {
			return OpResult{Finish: true, Error: err}
		}

		// all pools should be available
		for _, pool := range pools {
			if pool.Status != "Available" {
				return OpResult{
					Finish: true,
					Error:  fmt.Errorf("pool %s not available", pool.Asset),
				}
			}
		}

		// all chains should have pools
		if len(pools) != count {
			return OpResult{
				Finish: true,
				Error:  fmt.Errorf("expected %d pools, got %d", count, len(pools)),
			}
		}

		return OpResult{Finish: true}
	},
	)
	a.Append(verify)

	return a
}
