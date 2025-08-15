package keeperv1

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/common/swcysmartcontract"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

func (k KVStore) setSWCYStaker(ctx cosmos.Context, key string, record SWCYStaker) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getSWCYStaker(ctx cosmos.Context, key string, record *SWCYStaker) (bool, error) {
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

// GetSWCYStaker - gets swcy staker
func (k KVStore) GetSWCYStaker(ctx cosmos.Context, address common.Address) (SWCYStaker, error) {
	if swcysmartcontract.IsSWCYSmartContractAddress(address) {
		return k.getSWCYSmartContractAddressStaker(ctx)
	}
	record := NewSWCYStaker(address, math.ZeroUint())
	ok, err := k.getSWCYStaker(ctx, k.GetKey(prefixSWCYStaker, address.String()), &record)
	if !ok {
		return record, fmt.Errorf("SWCYStaker doesn't exist: %s", address.String())
	}
	return record, err
}

// SetSWCYStaker - update the swcy staker
func (k KVStore) SetSWCYStaker(ctx cosmos.Context, record SWCYStaker) error {
	// We don't modify smart contract staker
	if swcysmartcontract.IsSWCYSmartContractAddress(record.Address) {
		return nil
	}

	k.setSWCYStaker(ctx, k.GetKey(prefixSWCYStaker, record.Address.String()), record)
	return nil
}

// getSWCYStakerIterator iterate SWCY stakers
func (k KVStore) getSWCYStakerIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixSWCYStaker)
}

// DeleteSWCYStaker - deletes the swcy staker
func (k KVStore) DeleteSWCYStaker(ctx cosmos.Context, address common.Address) {
	// We don't modify smart contract staker
	if swcysmartcontract.IsSWCYSmartContractAddress(address) {
		return
	}
	k.del(ctx, k.GetKey(prefixSWCYStaker, address.String()))
}

// ListSWCYStakers gets stakers
func (k KVStore) ListSWCYStakers(ctx cosmos.Context) ([]SWCYStaker, error) {
	var stakers []SWCYStaker
	iterator := k.getSWCYStakerIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var staker SWCYStaker
		if err := k.cdc.Unmarshal(iterator.Value(), &staker); err != nil {
			ctx.Logger().Error("fail to unmarshal swcy staker", "error", err)
			continue
		}
		stakers = append(stakers, staker)
	}

	// Add SWCY smart contract staker
	swcySCStaker, err := k.getSWCYSmartContractAddressStaker(ctx)
	if err == nil {
		stakers = append(stakers, swcySCStaker)
	}

	return stakers, nil
}

func (k KVStore) SWCYStakerExists(ctx cosmos.Context, address common.Address) bool {
	if swcysmartcontract.IsSWCYSmartContractAddress(address) {
		return true
	}

	store := ctx.KVStore(k.storeKey)
	key := k.GetKey(prefixSWCYStaker, address.String())
	return store.Has([]byte(key))
}

func (k KVStore) UpdateSWCYStaker(ctx cosmos.Context, address common.Address, amount math.Uint) error {
	if k.SWCYStakerExists(ctx, address) {
		staker, err := k.GetSWCYStaker(ctx, address)
		if err != nil {
			return err
		}
		amount = amount.Add(staker.Amount)
	}

	return k.SetSWCYStaker(ctx, types.NewSWCYStaker(address, amount))
}

func (k KVStore) getSWCYSmartContractAddressStaker(ctx cosmos.Context) (SWCYStaker, error) {
	address, err := swcysmartcontract.GetSWCYSmartContractAddress()
	if err != nil {
		return SWCYStaker{}, err
	}
	accAddress, err := address.AccAddress()
	if err != nil {
		return SWCYStaker{}, err
	}
	coin := k.GetBalanceOf(ctx, accAddress, common.SWCY)
	if coin.IsNil() {
		return NewSWCYStaker(address, math.ZeroUint()), nil
	}

	return NewSWCYStaker(address, math.NewUint(coin.Amount.Uint64())), nil
}
