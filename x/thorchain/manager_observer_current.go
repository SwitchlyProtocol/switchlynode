package thorchain

import (
	"sort"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	keeper "github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

// ObserverMgrVCUR implement a ObserverManager which will store the
// observers in memory before written to chain
type ObserverMgrVCUR struct {
	chains map[common.Chain][]cosmos.AccAddress
}

// newObserverMgrVCUR create a new instance of ObserverManager
func newObserverMgrVCUR() *ObserverMgrVCUR {
	return &ObserverMgrVCUR{
		chains: make(map[common.Chain][]cosmos.AccAddress),
	}
}

// BeginBlock called when a new block get proposed
func (om *ObserverMgrVCUR) BeginBlock() {
	om.reset()
}

func (om *ObserverMgrVCUR) reset() {
	om.chains = make(map[common.Chain][]cosmos.AccAddress)
}

// AppendObserver add the address
func (om *ObserverMgrVCUR) AppendObserver(chain common.Chain, addrs []cosmos.AccAddress) {
	// combine addresses
	all := append(om.chains[chain], addrs...) // nolint

	// ensure uniqueness
	uniq := make([]cosmos.AccAddress, 0, len(all))
	m := make(map[string]bool)
	for _, val := range all {
		if _, ok := m[val.String()]; !ok {
			m[val.String()] = true
			uniq = append(uniq, val)
		}
	}

	om.chains[chain] = uniq
}

// List - gets a list of addresses that have been observed in all chains
func (om *ObserverMgrVCUR) List() []cosmos.AccAddress {
	result := make([]cosmos.AccAddress, 0)
	tracker := make(map[string]int)

	// analyze-ignore(map-iteration)
	for _, addrs := range om.chains {
		for _, addr := range addrs {
			// check if we need to init this key for the tracker
			if _, ok := tracker[addr.String()]; !ok {
				tracker[addr.String()] = 0
			}
			tracker[addr.String()]++
		}
	}

	// analyze-ignore(map-iteration)
	for key, count := range tracker {
		if count >= len(om.chains) {
			addr, _ := cosmos.AccAddressFromBech32(key)
			result = append(result, addr)
		}
	}

	// Sort our list, ensures we avoid a consensus failure
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].String() < result[j].String()
	})

	return result
}

// EndBlock emit the observers
func (om *ObserverMgrVCUR) EndBlock(ctx cosmos.Context, keeper keeper.Keeper) {
	if err := keeper.AddObservingAddresses(ctx, om.List()); err != nil {
		ctx.Logger().Error("fail to append observers", "error", err)
	}
	om.reset() // do not remove, would cause consensus failure
}
