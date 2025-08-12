package suites

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/test/simulation/actors/core"
	"github.com/switchlyprotocol/switchlynode/v3/test/simulation/pkg/thornode"
	. "github.com/switchlyprotocol/switchlynode/v3/test/simulation/pkg/types"
)

////////////////////////////////////////////////////////////////////////////////////////
// Ragnarok
////////////////////////////////////////////////////////////////////////////////////////

func Ragnarok() *Actor {
	a := NewActor("Ragnarok")

	// ragnarok all gas asset pools (should apply to tokens implicitly)
	for _, chain := range common.AllChains {
		if chain == common.THORChain {
			continue
		}
		// BSC not compatible with sim tests
		if chain.Equals(common.BSCChain) {
			continue
		}
		a.Children[core.NewRagnarokPoolActor(chain.GetGasAsset())] = true
	}

	// verify pool removals
	verify := NewActor("Ragnarok-Verify")
	verify.Ops = append(verify.Ops, func(config *OpConfig) OpResult {
		pools, err := thornode.GetPools()
		if err != nil {
			return OpResult{Finish: true, Error: err}
		}

		// no chains should have pools
		if len(pools) != 0 {
			return OpResult{
				Finish: true,
				Error:  fmt.Errorf("found %d pools after ragnarok", len(pools)),
			}
		}

		return OpResult{Finish: true}
	})
	a.Append(verify)

	return a
}
