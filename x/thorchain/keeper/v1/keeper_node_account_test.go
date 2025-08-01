package keeperv1

import (
	"github.com/blang/semver"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type KeeperNodeAccountSuite struct{}

var _ = Suite(&KeeperNodeAccountSuite{})

func (s *KeeperNodeAccountSuite) TestNodeAccount(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(10)

	na1 := GetRandomValidatorNode(NodeActive)
	na2 := GetRandomValidatorNode(NodeStandby)
	na3 := GetRandomVaultNode(NodeActive)

	c.Assert(k.SetNodeAccount(ctx, na1), IsNil)
	c.Assert(k.SetNodeAccount(ctx, na2), IsNil)
	c.Assert(k.SetNodeAccount(ctx, na3), IsNil)
	c.Check(na1.ActiveBlockHeight, Equals, int64(10))
	c.Check(na2.ActiveBlockHeight, Equals, int64(0))

	count, err := k.TotalActiveValidators(ctx)
	c.Assert(err, IsNil)
	c.Check(count, Equals, 1)

	na, err := k.GetNodeAccount(ctx, na1.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na.Equals(na1), Equals, true)

	na, err = k.GetNodeAccountByPubKey(ctx, na1.PubKeySet.Secp256k1)
	c.Assert(err, IsNil)
	c.Check(na.Equals(na1), Equals, true)

	valCon := "im unique!"
	pubkeys := GetRandomPubKeySet()
	err = k.EnsureNodeKeysUnique(ctx, na1.ValidatorConsPubKey, common.EmptyPubKeySet)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, "", pubkeys)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, na1.ValidatorConsPubKey, pubkeys)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, valCon, na1.PubKeySet)
	c.Assert(err, NotNil)
	err = k.EnsureNodeKeysUnique(ctx, valCon, pubkeys)
	c.Assert(err, IsNil)
	addr := GetRandomBech32Addr()
	na, err = k.GetNodeAccount(ctx, addr)
	c.Assert(err, IsNil)
	c.Assert(na.Status, Equals, NodeUnknown)
	c.Assert(na.ValidatorConsPubKey, Equals, "")
	nodeAccounts, err := k.ListValidatorsWithBond(ctx)
	c.Check(err, IsNil)
	c.Check(nodeAccounts.Len() > 0 && nodeAccounts.Len() < 3, Equals, true)
}

func (s *KeeperNodeAccountSuite) TestGetMinJoinVersion(c *C) {
	type nodeInfo struct {
		status  NodeStatus
		version semver.Version
	}
	inputs := []struct {
		nodeInfoes            []nodeInfo
		expectedVersion       semver.Version
		expectedActiveVersion semver.Version
	}{
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
			},
			expectedVersion:       semver.MustParse("0.3.0"),
			expectedActiveVersion: semver.MustParse("0.2.0"),
		},
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("1.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeStandby,
					version: semver.MustParse("0.2.0"),
				},
			},
			expectedVersion:       semver.MustParse("0.3.0"),
			expectedActiveVersion: semver.MustParse("0.2.0"),
		},
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("1.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.3.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.2.0"),
				},
			},
			expectedVersion:       semver.MustParse("0.2.0"),
			expectedActiveVersion: semver.MustParse("0.2.0"),
		},
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0+a"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0+b"),
				},
			},
			expectedVersion:       semver.MustParse("0.79.0"),
			expectedActiveVersion: semver.MustParse("0.79.0"),
		},
		{
			nodeInfoes: []nodeInfo{
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0-c"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0-a"),
				},
				{
					status:  NodeActive,
					version: semver.MustParse("0.79.0-b"),
				},
			},
			expectedVersion:       semver.MustParse("0.79.0-b"),
			expectedActiveVersion: semver.MustParse("0.79.0-a"),
		},
	}

	for _, item := range inputs {
		ctx, k := setupKeeperForTest(c)
		for _, ni := range item.nodeInfoes {
			na1 := GetRandomValidatorNode(ni.status)
			na1.Version = ni.version.String()
			c.Assert(k.SetNodeAccount(ctx, na1), IsNil)
		}
		c.Check(k.GetMinJoinVersion(ctx).Equals(item.expectedVersion), Equals, true, Commentf("%+v", k.GetMinJoinVersion(ctx)))
		c.Check(k.GetLowestActiveVersion(ctx).Equals(item.expectedActiveVersion), Equals, true)
	}
}

func (s *KeeperNodeAccountSuite) TestNodeAccountSlashPoints(c *C) {
	ctx, k := setupKeeperForTest(c)
	addr := GetRandomBech32Addr()

	pts, err := k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(0))

	pts = 5
	k.SetNodeAccountSlashPoints(ctx, addr, pts)
	pts, err = k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(5))

	c.Assert(k.IncNodeAccountSlashPoints(ctx, addr, 12), IsNil)
	pts, err = k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(17))

	c.Assert(k.DecNodeAccountSlashPoints(ctx, addr, 7), IsNil)
	pts, err = k.GetNodeAccountSlashPoints(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(pts, Equals, int64(10))
	k.ResetNodeAccountSlashPoints(ctx, GetRandomBech32Addr())
}

func (s *KeeperNodeAccountSuite) TestJail(c *C) {
	ctx, k := setupKeeperForTest(c)
	addr := GetRandomBech32Addr()

	jail, err := k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(0))
	c.Check(jail.Reason, Equals, "")

	// ensure setting it works
	err = k.SetNodeAccountJail(ctx, addr, 50, "foo")
	c.Assert(err, IsNil)
	jail, err = k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(50))
	c.Check(jail.Reason, Equals, "foo")

	// ensure we won't reduce sentence
	err = k.SetNodeAccountJail(ctx, addr, 20, "bar")
	c.Assert(err, IsNil)
	jail, err = k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(50))
	c.Check(jail.Reason, Equals, "foo")

	// ensure we can update
	err = k.SetNodeAccountJail(ctx, addr, 70, "bar")
	c.Assert(err, IsNil)
	jail, err = k.GetNodeAccountJail(ctx, addr)
	c.Assert(err, IsNil)
	c.Check(jail.NodeAddress.Equals(addr), Equals, true)
	c.Check(jail.ReleaseHeight, Equals, int64(70))
	c.Check(jail.Reason, Equals, "bar")
}

func (s *KeeperNodeAccountSuite) TestBondProviders(c *C) {
	acc := GetRandomBech32Addr()
	bp := NewBondProviders(acc)
	bp.NodeOperatorFee = cosmos.NewUint(2000)
	p := NewBondProvider(acc)
	p.Bond = cosmos.NewUint(100)
	bp.Providers = append(bp.Providers, p)
	c.Assert(bp.Providers, HasLen, 1)

	ctx, k := setupKeeperForTest(c)
	c.Assert(k.SetBondProviders(ctx, bp), IsNil)

	providers, err := k.GetBondProviders(ctx, acc)
	c.Assert(err, IsNil)
	c.Assert(providers.Providers, HasLen, 1)
}

func (s *KeeperNodeAccountSuite) TestRemoveLowBondValidatorAccounts(c *C) {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(10)

	na1 := GetRandomValidatorNode(NodeActive)
	na1.Bond = cosmos.ZeroUint()

	na2 := GetRandomValidatorNode(NodeStandby)
	na2.Bond = cosmos.NewUint(common.One)
	na2Coin := common.NewCoin(common.SwitchNative, na2.Bond)

	// sending tokens to bond
	c.Assert(k.MintToModule(ctx, ModuleName, na2Coin), IsNil)
	c.Assert(k.SendFromModuleToModule(ctx, ModuleName, BondName, common.NewCoins(na2Coin)), IsNil)

	// adding bond providers
	bAddress, err := na2.BondAddress.AccAddress()
	c.Assert(err, IsNil)
	bond := cosmos.NewUint(common.One / 2)
	na2BondProviders := BondProviders{
		NodeAddress:     na2.NodeAddress,
		NodeOperatorFee: cosmos.ZeroUint(),
		Providers: []BondProvider{
			{
				BondAddress: bAddress,
				Bond:        bond,
			},
			{
				BondAddress: GetRandomBech32Addr(),
				Bond:        bond,
			},
		},
	}
	c.Assert(k.SetBondProviders(ctx, na2BondProviders), IsNil)

	na3 := GetRandomVaultNode(NodeStandby)
	na3.Bond = cosmos.NewUint(10 * common.One)

	c.Assert(k.SetNodeAccount(ctx, na1), IsNil)
	c.Assert(k.SetNodeAccount(ctx, na2), IsNil)
	c.Assert(k.SetNodeAccount(ctx, na3), IsNil)
	c.Assert(k.RemoveLowBondValidatorAccounts(ctx), IsNil)

	// 1st and 3rd na should be skipped
	na1Store, err := k.GetNodeAccount(ctx, na1.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na1Store.IsEmpty(), Equals, false)
	c.Check(na1Store.String(), Equals, na1.String())

	na2Store, err := k.GetNodeAccount(ctx, na2.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na2Store.IsEmpty(), Equals, true)

	// check bond providers
	baStore, err := k.GetBondProviders(ctx, bAddress)
	c.Assert(err, IsNil)
	c.Check(len(baStore.Providers), Equals, 0)
	// check bal
	for _, bps := range na2BondProviders.Providers {
		bal := k.GetBalance(ctx, bps.BondAddress)
		c.Check(bal[0].Amount.Int64(), Equals, int64(bond.Uint64()))
	}

	na3Store, err := k.GetNodeAccount(ctx, na3.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na3Store.IsEmpty(), Equals, false)
	c.Check(na3Store.String(), Equals, na3.String())
}
