package keeperv1

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

////////////////////////////////////////////////////////////////////////////////////////
// SwitchPool
////////////////////////////////////////////////////////////////////////////////////////

func (k KVStore) GetSwitchPool(ctx cosmos.Context) (SwitchPool, error) {
	record := NewSwitchPool()
	key := k.GetKey(prefixSwitchPool, "")

	store := ctx.KVStore(k.storeKey)
	if !store.Has([]byte(key)) {
		return record, nil
	}

	bz := store.Get([]byte(key))
	if err := k.cdc.Unmarshal(bz, &record); err != nil {
		return record, dbError(ctx, fmt.Sprintf("Unmarshal kvstore: (%T) %s", record, key), err)
	}
	return record, nil
}

func (k KVStore) SetSwitchPool(ctx cosmos.Context, pool SwitchPool) {
	store := ctx.KVStore(k.storeKey)
	key := k.GetKey(prefixSwitchPool, "")
	buf := k.cdc.MustMarshal(&pool)
	store.Set([]byte(key), buf)
}

////////////////////////////////////////////////////////////////////////////////////////
// SwitchProviders
////////////////////////////////////////////////////////////////////////////////////////

func (k KVStore) setSWITCHProvider(ctx cosmos.Context, key string, record SWITCHProvider) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getSWITCHProvider(ctx cosmos.Context, key string, record *SWITCHProvider) (bool, error) {
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

// GetSWITCHProviderIterator iterate SWITCH providers
func (k KVStore) GetSWITCHProviderIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixSWITCHProvider)
}

// GetSWITCHProvider retrieve SWITCH provider from the data store
func (k KVStore) GetSWITCHProvider(ctx cosmos.Context, addr cosmos.AccAddress) (SWITCHProvider, error) {
	record := SWITCHProvider{
		SwitchAddress:  addr,
		DepositAmount:  cosmos.ZeroUint(),
		WithdrawAmount: cosmos.ZeroUint(),
		Units:          cosmos.ZeroUint(),
	}

	_, err := k.getSWITCHProvider(ctx, k.GetKey(prefixSWITCHProvider, record.Key()), &record)
	return record, err
}

// SetSWITCHProvider save the SWITCH provider to kv store
func (k KVStore) SetSWITCHProvider(ctx cosmos.Context, rp SWITCHProvider) {
	k.setSWITCHProvider(ctx, k.GetKey(prefixSWITCHProvider, rp.Key()), rp)
}

// RemoveSWITCHProvider remove the SWITCH provider from the kv store
func (k KVStore) RemoveSWITCHProvider(ctx cosmos.Context, rp SWITCHProvider) {
	k.del(ctx, k.GetKey(prefixSWITCHProvider, rp.Key()))
}
