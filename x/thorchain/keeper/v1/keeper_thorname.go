package keeperv1

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

func (k KVStore) setTHORName(ctx cosmos.Context, key string, record THORName) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getTHORName(ctx cosmos.Context, key string, record *THORName) (bool, error) {
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

// GetTHORNameIterator only iterate THORNames
func (k KVStore) GetTHORNameIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixTHORName)
}

// SetTHORName save the THORName object to store
func (k KVStore) SetTHORName(ctx cosmos.Context, name THORName) {
	k.setTHORName(ctx, k.GetKey(prefixTHORName, name.Key()), name)
}

// THORNameExists check whether the given name exists
func (k KVStore) THORNameExists(ctx cosmos.Context, name string) bool {
	record := THORName{
		Name: name,
	}
	if k.has(ctx, k.GetKey(prefixTHORName, record.Key())) {
		record, _ = k.GetTHORName(ctx, name)
		return record.ExpireBlockHeight >= ctx.BlockHeight()
	}
	return false
}

// GetTHORName get THORName with the given pubkey from data store
func (k KVStore) GetTHORName(ctx cosmos.Context, name string) (THORName, error) {
	record := THORName{
		Name: name,
	}
	ok, err := k.getTHORName(ctx, k.GetKey(prefixTHORName, record.Key()), &record)
	if !ok {
		return record, fmt.Errorf("THORName doesn't exist: %s", name)
	}
	if record.ExpireBlockHeight < ctx.BlockHeight() {
		return THORName{Name: name}, nil
	}
	return record, err
}

// DeleteTHORName remove the given THORName from data store
func (k KVStore) DeleteTHORName(ctx cosmos.Context, name string) error {
	n := THORName{Name: name}
	k.del(ctx, k.GetKey(prefixTHORName, n.Key()))
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
		RuneAmount:   cosmos.ZeroUint(),
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
