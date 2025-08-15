package keeperv1

import (
	. "gopkg.in/check.v1"

	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type KeeperSWITCHProviderSuite struct{}

var _ = Suite(&KeeperSWITCHProviderSuite{})

func (mas *KeeperSWITCHProviderSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *KeeperSWITCHProviderSuite) TestSWITCHProvider(c *C) {
	ctx, k := setupKeeperForTest(c)

	addr := GetRandomSwitchAddress()
	accAddr, err := addr.AccAddress()
	c.Check(err, IsNil)
	rp, err := k.GetSWITCHProvider(ctx, accAddr)
	c.Assert(err, IsNil)
	c.Check(rp.SwitchAddress, NotNil)
	c.Check(rp.Units, NotNil)

	addr = GetRandomSwitchAddress()
	accAddr, err = addr.AccAddress()
	c.Assert(err, IsNil)
	rp = SWITCHProvider{
		Units:         cosmos.NewUint(12),
		DepositAmount: cosmos.NewUint(12),
		SwitchAddress: accAddr,
	}
	k.SetSWITCHProvider(ctx, rp)
	rp, err = k.GetSWITCHProvider(ctx, rp.SwitchAddress)
	c.Assert(err, IsNil)
	c.Check(rp.SwitchAddress.Equals(accAddr), Equals, true)
	c.Check(rp.Units.Equal(cosmos.NewUint(12)), Equals, true)
	c.Check(rp.DepositAmount.Equal(cosmos.NewUint(12)), Equals, true)
	c.Check(rp.WithdrawAmount.Equal(cosmos.NewUint(0)), Equals, true)

	var rps []SWITCHProvider
	iterator := k.GetSWITCHProviderIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		k.Cdc().MustUnmarshal(iterator.Value(), &rp)
		if rp.SwitchAddress.Empty() {
			continue
		}
		rps = append(rps, rp)
	}
	c.Check(rps[0].SwitchAddress.Equals(accAddr), Equals, true)

	secondAddr := GetRandomSwitchAddress()
	secondAccAddr, err := secondAddr.AccAddress()
	c.Check(err, IsNil)
	rp2 := SWITCHProvider{
		Units:         cosmos.NewUint(24),
		DepositAmount: cosmos.NewUint(24),
		SwitchAddress: secondAccAddr,
	}
	k.SetSWITCHProvider(ctx, rp2)

	rps = []SWITCHProvider{}
	iterator = k.GetSWITCHProviderIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		k.Cdc().MustUnmarshal(iterator.Value(), &rp)
		if rp.SwitchAddress.Empty() {
			continue
		}
		rps = append(rps, rp)
	}
	c.Check(len(rps), Equals, 2)

	totalUnits := cosmos.ZeroUint()
	iterator = k.GetSWITCHProviderIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		k.Cdc().MustUnmarshal(iterator.Value(), &rp)
		if rp.SwitchAddress.Empty() {
			continue
		}
		totalUnits = totalUnits.Add(rp.Units)
	}
	c.Check(totalUnits.Equal(cosmos.NewUint(36)), Equals, true)

	k.RemoveSWITCHProvider(ctx, rp)
	k.RemoveSWITCHProvider(ctx, rp2)
}
