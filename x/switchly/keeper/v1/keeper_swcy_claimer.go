package keeperv1

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/keeper/types"
)

func (k KVStore) setSWCYClaimer(ctx cosmos.Context, key string, record SWCYClaimer) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getSWCYClaimer(ctx cosmos.Context, key string, record *SWCYClaimer) (bool, error) {
	store := ctx.KVStore(k.storeKey)
	if !store.Has([]byte(key)) {
		return false, nil
	}

	bz := store.Get([]byte(key))
	if err := k.cdc.Unmarshal(bz, record); err != nil {
		return true, dbError(ctx, fmt.Sprintf("Unmarshal kvstore: (%T) %s", record, key), err)
	}
	return true, nil
}

// GetSWCYClaimer - gets tcy claimer
func (k KVStore) GetSWCYClaimer(ctx cosmos.Context, l1Address common.Address, asset common.Asset) (SWCYClaimer, error) {
	record := NewSWCYClaimer(l1Address, asset, math.ZeroUint())
	key := fmt.Sprintf("%s/%s", l1Address.String(), asset.String())
	ok, err := k.getSWCYClaimer(ctx, k.GetKey(prefixSWCYClaimer, key), &record)
	if !ok {
		return record, fmt.Errorf("SWCYClaimer doesn't exist: %s", l1Address.String())
	}
	return record, err
}

// SetSWCYClaimer - update the tcy claimer
func (k KVStore) SetSWCYClaimer(ctx cosmos.Context, record SWCYClaimer) error {
	key := fmt.Sprintf("%s/%s", record.L1Address.String(), record.Asset.String())
	k.setSWCYClaimer(ctx, k.GetKey(prefixSWCYClaimer, key), record)
	return nil
}

// GetSWCYClaimerIteratorFromL1Address iterate SWCY claimers
func (k KVStore) GetSWCYClaimerIteratorFromL1Address(ctx cosmos.Context, l1Address common.Address) cosmos.Iterator {
	key := k.GetKey(prefixSWCYClaimer, l1Address.String())
	return k.getIterator(ctx, types.DbPrefix(key))
}

// GetSWCYClaimerIterator iterate SWCY claimers
func (k KVStore) GetSWCYClaimerIterator(ctx cosmos.Context) cosmos.Iterator {
	key := k.GetKey(prefixSWCYClaimer, "")
	return k.getIterator(ctx, types.DbPrefix(key))
}

// DeleteSWCYClaimer - deletes the tcy claimer
func (k KVStore) DeleteSWCYClaimer(ctx cosmos.Context, l1Address common.Address, asset common.Asset) {
	key := fmt.Sprintf("%s/%s", l1Address.String(), asset.String())
	k.del(ctx, k.GetKey(prefixSWCYClaimer, key))
}

// ListSWCYClaimersFromL1Address gets claims from l1 address
func (k KVStore) ListSWCYClaimersFromL1Address(ctx cosmos.Context, l1Address common.Address) ([]SWCYClaimer, error) {
	var claimers []SWCYClaimer
	iterator := k.GetSWCYClaimerIteratorFromL1Address(ctx, l1Address)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var claimer SWCYClaimer
		if err := k.cdc.Unmarshal(iterator.Value(), &claimer); err != nil {
			ctx.Logger().Error("fail to unmarshal tcy claimer", "error", err)
			continue
		}
		claimers = append(claimers, claimer)
	}

	if len(claimers) == 0 {
		return []SWCYClaimer{}, fmt.Errorf("l1 address: (%s) doesn't have any tcy to claim", l1Address.String())
	}
	return claimers, nil
}

// SWCYClaimerExists checks if claimer already exists
func (k KVStore) SWCYClaimerExists(ctx cosmos.Context, l1Address common.Address, asset common.Asset) bool {
	store := ctx.KVStore(k.storeKey)
	key := k.GetKey(prefixSWCYClaimer, fmt.Sprintf("%s/%s", l1Address.String(), asset.String()))
	return store.Has([]byte(key))
}

// UpdateSWCYClaimer will update value if claimer exists, if not it will create a new one
func (k KVStore) UpdateSWCYClaimer(ctx cosmos.Context, l1Address common.Address, asset common.Asset, amount math.Uint) error {
	if k.SWCYClaimerExists(ctx, l1Address, asset) {
		claimer, err := k.GetSWCYClaimer(ctx, l1Address, asset)
		if err != nil {
			return err
		}
		amount = amount.Add(claimer.Amount)
	}

	return k.SetSWCYClaimer(ctx, NewSWCYClaimer(l1Address, asset, amount))
}
