package keeperv1

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/config"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper/types"
)

func (k KVStore) setNodeAccount(ctx cosmos.Context, key string, record NodeAccount) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getNodeAccount(ctx cosmos.Context, key string, record *NodeAccount) (bool, error) {
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

// TotalActiveValidators count the number of active node account
func (k KVStore) TotalActiveValidators(ctx cosmos.Context) (int, error) {
	activeNodes, err := k.ListActiveValidators(ctx)
	return len(activeNodes), err
}

// ListValidatorsWithBond - gets a list of all validator node accounts that have bond
// Note: the order of node account in the result is not defined
func (k KVStore) ListValidatorsWithBond(ctx cosmos.Context) (NodeAccounts, error) {
	nodeAccounts := make(NodeAccounts, 0)
	naIterator := k.GetNodeAccountIterator(ctx)
	defer naIterator.Close()
	for ; naIterator.Valid(); naIterator.Next() {
		var na NodeAccount
		if err := k.cdc.Unmarshal(naIterator.Value(), &na); err != nil {
			return nodeAccounts, dbError(ctx, "Unmarshal: node account", err)
		}
		if na.Type == NodeTypeValidator && !na.Bond.IsZero() {
			nodeAccounts = append(nodeAccounts, na)
		}
	}
	return nodeAccounts, nil
}

// ListValidatorsByStatus - get a list of validator node accounts with the given status
func (k KVStore) ListValidatorsByStatus(ctx cosmos.Context, status NodeStatus) (NodeAccounts, error) {
	nodeAccounts := make(NodeAccounts, 0)
	naIterator := k.GetNodeAccountIterator(ctx)
	defer naIterator.Close()
	for ; naIterator.Valid(); naIterator.Next() {
		var na NodeAccount
		if err := k.cdc.Unmarshal(naIterator.Value(), &na); err != nil {
			return nodeAccounts, dbError(ctx, "Unmarshal: node account", err)
		}
		if na.Type == NodeTypeValidator && na.Status == status {
			nodeAccounts = append(nodeAccounts, na)
		}
	}
	return nodeAccounts, nil
}

// ListActiveValidators - get a list of active validator node accounts
func (k KVStore) ListActiveValidators(ctx cosmos.Context) (NodeAccounts, error) {
	return k.ListValidatorsByStatus(ctx, NodeActive)
}

func (k KVStore) RemoveLowBondValidatorAccounts(ctx cosmos.Context) error {
	var events cosmos.Events
	lowBondValidators := make([][]byte, 0)
	naIterator := k.GetNodeAccountIterator(ctx)
	defer naIterator.Close()
	for ; naIterator.Valid(); naIterator.Next() {
		var na NodeAccount
		if err := k.cdc.Unmarshal(naIterator.Value(), &na); err != nil {
			return dbError(ctx, "Unmarshal: node account", err)
		}
		if na.Type == NodeTypeVault || na.Status == NodeActive {
			continue
		}
		if na.Type == NodeTypeValidator && na.Bond.LTE(cosmos.NewUint(common.One)) {
			lowBondValidators = append(lowBondValidators, naIterator.Key())
			if na.Bond.IsZero() {
				continue
			}
			bps, err := k.GetBondProviders(ctx, na.NodeAddress)
			if err != nil {
				return err
			}
			to, err := na.BondAddress.AccAddress()
			if err != nil {
				return dbError(ctx, "", fmt.Errorf("fail to parse bond address(%s)", na.BondAddress))
			}

			// No bond providers
			if len(bps.Providers) == 0 {
				coin := common.NewCoin(common.SwitchNative, na.Bond)
				if err = k.SendFromModuleToAccount(ctx, BondName, to, common.NewCoins(coin)); err != nil {
					ctx.Logger().Error("failed to return bond pool coins", "error", err)
					continue
				}
				bondEvent := NewEventBond(na.Bond, BondReturned, common.Tx{}, &na, to)
				if events, err = bondEvent.Events(); err != nil {
					ctx.Logger().Error("fail to emit bond event", "error", err)
				} else {
					ctx.EventManager().EmitEvents(events)
				}
				continue
			}

			bps.Adjust(na.Bond)
			totalSent := cosmos.ZeroUint()
			for _, provider := range bps.Providers {
				if provider.Bond.IsZero() || provider.BondAddress.Empty() {
					continue
				}
				coin := common.NewCoin(common.SwitchNative, provider.Bond)
				if err = k.SendFromModuleToAccount(ctx, BondName, provider.BondAddress, common.NewCoins(coin)); err != nil {
					ctx.Logger().Error("failed to return bond pool coins", "error", err)
					continue
				}
				bondEvent := NewEventBond(provider.Bond, BondReturned, common.Tx{}, &na, provider.BondAddress)
				if events, err = bondEvent.Events(); err != nil {
					ctx.Logger().Error("fail to emit bond event", "error", err)
				} else {
					ctx.EventManager().EmitEvents(events)
				}
				totalSent = totalSent.Add(provider.Bond)
			}
			// sanity check
			if totalSent.GT(na.Bond) {
				return dbError(ctx, "", fmt.Errorf("total bond returned greater than node bond"))
			}

			// remove bond providers
			k.del(ctx, k.GetKey(prefixBondProviders, na.NodeAddress.String()))
		}
	}
	for _, naKey := range lowBondValidators {
		k.del(ctx, string(naKey))
	}
	return nil
}

// GetMinJoinVersion - get min version to join. Min version is the most popular version
func (k KVStore) GetMinJoinVersion(ctx cosmos.Context) semver.Version {
	type tmpVersionInfo struct {
		version semver.Version
		count   int
	}
	var vCount []tmpVersionInfo
	nodes, err := k.ListActiveValidators(ctx)
	if err != nil {
		_ = dbError(ctx, "Unable to list active node accounts", err)
		return semver.Version{}
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].GetVersion().LT(nodes[j].GetVersion())
	})
	for _, na := range nodes {
		exist := false
		for _, item := range vCount {
			if item.version.String() == na.Version {
				exist = true
				break
			}
		}
		if !exist {
			vCount = append(vCount, tmpVersionInfo{
				version: na.GetVersion(),
				count:   0,
			})
		}

		// assume all versions are backward compatible
		for k, v := range vCount {
			if v.version.LTE(na.GetVersion()) {
				v.count++
				vCount[k] = v
			}
		}
	}
	totalCount := len(nodes)
	version := semver.Version{}
	// sort it by version descending
	sort.SliceStable(vCount, func(i, j int) bool {
		return vCount[i].version.GT(vCount[j].version)
	})

	for _, info := range vCount {
		// skip those version that doesn't have majority
		if !HasSuperMajority(info.count, totalCount) {
			continue
		}
		if info.version.GT(version) {
			version = info.version
		}

	}
	return version
}

// GetLowestActiveVersion - get version number of lowest active node
func (k KVStore) GetLowestActiveVersion(ctx cosmos.Context) semver.Version {
	nodes, err := k.ListActiveValidators(ctx)
	if err != nil {
		_ = dbError(ctx, "Unable to list active node accounts", err)
		return constants.SWVersion
	}
	if len(nodes) > 0 {
		version := nodes[0].GetVersion()
		for _, na := range nodes {
			if na.GetVersion().LT(version) {
				version = na.GetVersion()
			}
		}
		return version
	}
	return constants.SWVersion
}

// GetNodeAccount try to get node account with the given address from db
func (k KVStore) GetNodeAccount(ctx cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	emptyPubKeySet := common.PubKeySet{
		Secp256k1: common.EmptyPubKey,
		Ed25519:   common.EmptyPubKey,
	}
	record := NewNodeAccount(addr, NodeUnknown, emptyPubKeySet, "", cosmos.ZeroUint(), "", ctx.BlockHeight())
	_, err := k.getNodeAccount(ctx, k.GetKey(prefixNodeAccount, addr.String()), &record)
	return record, err
}

// GetNodeAccountByPubKey try to get node account with the given pubkey from db
func (k KVStore) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	addr, err := pk.GetThorAddress()
	if err != nil {
		return NodeAccount{}, err
	}
	return k.GetNodeAccount(ctx, addr)
}

// SetNodeAccount save the given node account into data store
func (k KVStore) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if na.IsEmpty() {
		return nil
	}
	if na.Status == NodeActive {
		if na.ActiveBlockHeight == 0 {
			// the na is active, and does not have a block height when they
			// became active. This must be the first block they are active, so
			// THORNode will set it now.
			na.ActiveBlockHeight = ctx.BlockHeight()
			k.ResetNodeAccountSlashPoints(ctx, na.NodeAddress) // reset slash points
		}
	}

	k.setNodeAccount(ctx, k.GetKey(prefixNodeAccount, na.NodeAddress.String()), na)
	return nil
}

// EnsureNodeKeysUnique check the given consensus pubkey and pubkey set against all the the node account
// return an error when it is overlap with any existing account
func (k KVStore) EnsureNodeKeysUnique(ctx cosmos.Context, consensusPubKey string, pubKeys common.PubKeySet) error {
	iter := k.GetNodeAccountIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var na NodeAccount
		if err := k.cdc.Unmarshal(iter.Value(), &na); err != nil {
			return dbError(ctx, "Unmarshal: node account", err)
		}
		if strings.EqualFold("", consensusPubKey) {
			return dbError(ctx, "", errors.New("Validator Consensus Key cannot be empty"))
		}
		if na.ValidatorConsPubKey == consensusPubKey {
			return dbError(ctx, "", fmt.Errorf("%s already exist", na.ValidatorConsPubKey))
		}

		if pubKeys.IsEmpty() {
			return dbError(ctx, "", errors.New("PubKeySet cannot be empty"))
		}
		if na.PubKeySet.Contains(pubKeys.Secp256k1) {
			return dbError(ctx, "", fmt.Errorf("%s already exist", pubKeys))
		}
		if na.PubKeySet.Contains(pubKeys.Ed25519) {
			return dbError(ctx, "", fmt.Errorf("%s already exist", pubKeys))
		}
	}

	return nil
}

// GetNodeAccountIterator iterate node account
func (k KVStore) GetNodeAccountIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixNodeAccount)
}

// GetUpgradeProposalIterator to iterate upgrade proposals.
func (k KVStore) GetUpgradeProposalIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixUpgradeProposals)
}

// GetUpgradeVoteIterator to iterate upgrade votes for a named proposal.
func (k KVStore) GetUpgradeVoteIterator(ctx cosmos.Context, name string) cosmos.Iterator {
	return k.getIterator(ctx, types.DbPrefix(VotePrefix(name)))
}

func VotePrefix(name string) string {
	return fmt.Sprintf("%s%s/", prefixUpgradeVotes, name)
}

// GetNodeAccountSlashPoints - get the slash points associated with the given
// node address
func (k KVStore) GetNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress) (int64, error) {
	record := int64(0)
	_, err := k.getInt64(ctx, k.GetKey(prefixNodeSlashPoints, addr.String()), &record)
	return record, err
}

// SetNodeAccountSlashPoints - set the slash points associated with the given
// node address and uint
func (k KVStore) SetNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) {
	// make sure slash point doesn't go to negative
	if pts < 0 {
		pts = 0
	}
	k.setInt64(ctx, k.GetKey(prefixNodeSlashPoints, addr.String()), pts)
}

// ResetNodeAccountSlashPoints - reset the slash points to zero for associated
// with the given node address
func (k KVStore) ResetNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress) {
	k.del(ctx, k.GetKey(prefixNodeSlashPoints, addr.String()))
}

// IncNodeAccountSlashPoints - increments the slash points associated with the
// given node address and uint
func (k KVStore) IncNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	current, err := k.GetNodeAccountSlashPoints(ctx, addr)
	if err != nil {
		return err
	}
	k.SetNodeAccountSlashPoints(ctx, addr, current+pts)

	metricLabels, _ := ctx.Context().Value(constants.CtxMetricLabels).([]metrics.Label)
	telemetry.IncrCounterWithLabels(
		[]string{"thornode", "point_slash"},
		float32(pts),
		append(
			metricLabels,
			telemetry.NewLabel("address", addr.String()),
		),
	)

	if config.GetSwitchly().Telemetry.SlashPoints {
		slashTelemetry(ctx, pts, addr, "IncSlashPoints")
	}

	return nil
}

// DecNodeAccountSlashPoints - decrements the slash points associated with the
// given node address and uint
func (k KVStore) DecNodeAccountSlashPoints(ctx cosmos.Context, addr cosmos.AccAddress, pts int64) error {
	current, err := k.GetNodeAccountSlashPoints(ctx, addr)
	if err != nil {
		return err
	}
	k.SetNodeAccountSlashPoints(ctx, addr, current-pts)

	dec := pts
	if dec > current {
		dec = current
	}

	metricLabels, _ := ctx.Context().Value(constants.CtxMetricLabels).([]metrics.Label)
	telemetry.IncrCounterWithLabels(
		[]string{"thornode", "point_slash_refund"},
		float32(dec),
		append(
			metricLabels,
			telemetry.NewLabel("address", addr.String()),
		),
	)

	if config.GetSwitchly().Telemetry.SlashPoints {
		slashTelemetry(ctx, -pts, addr, "DecSlashPoints")
	}

	return nil
}

func (k KVStore) setJail(ctx cosmos.Context, key string, record Jail) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getJail(ctx cosmos.Context, key string, record *Jail) (bool, error) {
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

// GetNodeAccountJail - gets jail details for a given node address
func (k KVStore) GetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress) (Jail, error) {
	record := NewJail(addr)
	_, err := k.getJail(ctx, k.GetKey(prefixNodeJail, addr.String()), &record)
	return record, err
}

// SetNodeAccountJail - update the jail details of a node account
func (k KVStore) SetNodeAccountJail(ctx cosmos.Context, addr cosmos.AccAddress, height int64, reason string) error {
	jail, err := k.GetNodeAccountJail(ctx, addr)
	if err != nil {
		return err
	}
	// never reduce sentence
	if jail.ReleaseHeight > height {
		return nil
	}
	jail.ReleaseHeight = height
	jail.Reason = reason

	k.setJail(ctx, k.GetKey(prefixNodeJail, addr.String()), jail)
	return nil
}

// ReleaseNodeAccountFromJail - update the jail details of a node account
func (k KVStore) ReleaseNodeAccountFromJail(ctx cosmos.Context, addr cosmos.AccAddress) error {
	jail, err := k.GetNodeAccountJail(ctx, addr)
	if err != nil {
		return err
	}
	jail.ReleaseHeight = ctx.BlockHeight()
	jail.Reason = ""
	k.setJail(ctx, k.GetKey(prefixNodeJail, addr.String()), jail)
	return nil
}

func (k KVStore) setBondProviders(ctx cosmos.Context, key string, record BondProviders) {
	store := ctx.KVStore(k.storeKey)
	buf := k.cdc.MustMarshal(&record)
	if buf == nil {
		store.Delete([]byte(key))
	} else {
		store.Set([]byte(key), buf)
	}
}

func (k KVStore) getBondProviders(ctx cosmos.Context, key string, record *BondProviders) (bool, error) {
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

// GetBondProviders - gets bond providers for a node account
func (k KVStore) GetBondProviders(ctx cosmos.Context, addr cosmos.AccAddress) (BondProviders, error) {
	record := NewBondProviders(addr)
	_, err := k.getBondProviders(ctx, k.GetKey(prefixBondProviders, addr.String()), &record)
	return record, err
}

// SetBondProviders - update the bond providers of a node account
func (k KVStore) SetBondProviders(ctx cosmos.Context, record BondProviders) error {
	k.setBondProviders(ctx, k.GetKey(prefixBondProviders, record.NodeAddress.String()), record)
	return nil
}

func (k KVStore) DeductNativeTxFeeFromBond(ctx cosmos.Context, nodeAddr cosmos.AccAddress) error {
	fee := k.GetNativeTxFee(ctx)
	if fee.IsZero() {
		return nil // no fee
	}

	// deduct fee from node account's bond
	na, err := k.GetNodeAccount(ctx, nodeAddr)
	if err != nil {
		return err
	}
	if na.Bond.LT(fee) {
		return fmt.Errorf("not enound bond for fee")
	}
	na.Bond = common.SafeSub(na.Bond, fee)
	if err = k.SetNodeAccount(ctx, na); err != nil {
		return fmt.Errorf("fail to save node account: %w", err)
	}

	// transfer fee from bond module to reserve
	coins := common.NewCoins(common.NewCoin(common.SwitchNative, fee))
	if err = k.SendFromModuleToModule(ctx, BondName, ReserveName, coins); err != nil {
		return err
	}

	// emit bond cost event
	bondEvent := NewEventBond(fee, BondCost, common.Tx{}, &na, nil)
	events, err := bondEvent.Events()
	if err != nil {
		return fmt.Errorf("fail to get events: %w", err)
	}
	ctx.EventManager().EmitEvents(events)

	return nil
}
