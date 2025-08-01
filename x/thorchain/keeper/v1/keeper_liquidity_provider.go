package keeperv1

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper/types"
)

func (k KVStore) setLiquidityProvider(ctx cosmos.Context, key string, record LiquidityProvider) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getLiquidityProvider(ctx cosmos.Context, key string, record *LiquidityProvider) (bool, error) {
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

// GetLiquidityProviderIterator iterate liquidity providers
func (k KVStore) GetLiquidityProviderIterator(ctx cosmos.Context, asset common.Asset) cosmos.Iterator {
	key := k.GetKey(prefixLiquidityProvider, (&LiquidityProvider{Asset: asset}).Key())
	return k.getIterator(ctx, types.DbPrefix(key))
}

func (k KVStore) GetTotalSupply(ctx cosmos.Context, asset common.Asset) cosmos.Uint {
	// when pool ragnarok started , synth unit become zero
	lay1Asset := asset.GetLayer1Asset()
	ragStart, _ := k.GetPoolRagnarokStart(ctx, lay1Asset)
	if ragStart > 0 {
		return cosmos.ZeroUint()
	}

	coin := k.coinKeeper.GetSupply(ctx, asset.Native())
	return cosmos.NewUint(coin.Amount.Uint64())
}

// GetLiquidityProvider retrieve liquidity provider from the data store
func (k KVStore) GetLiquidityProvider(ctx cosmos.Context, asset common.Asset, addr common.Address) (LiquidityProvider, error) {
	record := LiquidityProvider{
		Asset:             asset,
		RuneAddress:       addr,
		Units:             cosmos.ZeroUint(),
		PendingRune:       cosmos.ZeroUint(),
		PendingAsset:      cosmos.ZeroUint(),
		RuneDepositValue:  cosmos.ZeroUint(),
		AssetDepositValue: cosmos.ZeroUint(),
	}
	if !addr.IsChain(common.SwitchNative.Chain) {
		record.AssetAddress = addr
		record.RuneAddress = common.NoAddress
	}

	_, err := k.getLiquidityProvider(ctx, k.GetKey(prefixLiquidityProvider, record.Key()), &record)
	if err != nil {
		return record, err
	}

	return record, nil
}

// SetLiquidityProvider save the liquidity provider to kv store
func (k KVStore) SetLiquidityProvider(ctx cosmos.Context, lp LiquidityProvider) {
	k.setLiquidityProvider(ctx, k.GetKey(prefixLiquidityProvider, lp.Key()), lp)
}

// RemoveLiquidityProvider remove the liquidity provider to kv store
func (k KVStore) RemoveLiquidityProvider(ctx cosmos.Context, lp LiquidityProvider) {
	k.del(ctx, k.GetKey(prefixLiquidityProvider, lp.Key()))
}
