package keeperv1

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

func (k KVStore) setSWITCHName(ctx cosmos.Context, key string, record SWITCHName) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getSWITCHName(ctx cosmos.Context, key string, record *SWITCHName) (bool, error) {
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

// GetSWITCHNameIterator only iterate SWITCHNames
func (k KVStore) GetSWITCHNameIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixSWITCHName)
}

// SetSWITCHName save the SWITCHName object to store
func (k KVStore) SetSWITCHName(ctx cosmos.Context, name SWITCHName) {
	k.setSWITCHName(ctx, k.GetKey(prefixSWITCHName, name.Key()), name)
}

// SWITCHNameExists check whether the given name exists
func (k KVStore) SWITCHNameExists(ctx cosmos.Context, name string) bool {
	record := SWITCHName{
		Name: name,
	}
	if k.has(ctx, k.GetKey(prefixSWITCHName, record.Key())) {
		record, _ = k.GetSWITCHName(ctx, name)
		return record.ExpireBlockHeight >= ctx.BlockHeight()
	}
	return false
}

// GetSWITCHName get SWITCHName with the given pubkey from data store
func (k KVStore) GetSWITCHName(ctx cosmos.Context, name string) (SWITCHName, error) {
	record := SWITCHName{
		Name: name,
	}
	ok, err := k.getSWITCHName(ctx, k.GetKey(prefixSWITCHName, record.Key()), &record)
	if !ok {
		return record, fmt.Errorf("SWITCHName doesn't exist: %s", name)
	}
	if record.ExpireBlockHeight < ctx.BlockHeight() {
		return SWITCHName{Name: name}, nil
	}
	return record, err
}

// DeleteSWITCHName remove the given SWITCHName from data store
func (k KVStore) DeleteSWITCHName(ctx cosmos.Context, name string) error {
	n := SWITCHName{Name: name}
	k.del(ctx, k.GetKey(prefixSWITCHName, n.Key()))
	return nil
}

// AffiliateFeeCollector

func (k KVStore) setAffiliateCollector(ctx cosmos.Context, key string, record AffiliateFeeCollector) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getAffilateCollector(ctx cosmos.Context, key string, record *AffiliateFeeCollector) (bool, error) {
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

func (k KVStore) SetAffiliateCollector(ctx cosmos.Context, collector AffiliateFeeCollector) {
	k.setAffiliateCollector(ctx, k.GetKey(prefixAffiliateCollector, collector.OwnerAddress.String()), collector)
}

func (k KVStore) GetAffiliateCollector(ctx cosmos.Context, acc cosmos.AccAddress) (AffiliateFeeCollector, error) {
	record := AffiliateFeeCollector{
		OwnerAddress: acc,
		SwitchAmount: cosmos.ZeroUint(),
	}
	_, err := k.getAffilateCollector(ctx, k.GetKey(prefixAffiliateCollector, record.OwnerAddress.String()), &record)
	return record, err
}

func (k KVStore) GetAffiliateCollectorIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixAffiliateCollector)
}

func (k KVStore) GetAffiliateCollectors(ctx cosmos.Context) ([]AffiliateFeeCollector, error) {
	var affCols []AffiliateFeeCollector
	iterator := k.GetAffiliateCollectorIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var ac AffiliateFeeCollector
		err := k.Cdc().Unmarshal(iterator.Value(), &ac)
		if err != nil {
			return nil, dbError(ctx, "Unmarsahl: ac", err)
		}
		affCols = append(affCols, ac)
	}
	return affCols, nil
}
