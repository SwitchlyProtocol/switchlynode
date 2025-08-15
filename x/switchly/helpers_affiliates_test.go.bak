package switchly

import (
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

type HelperAffiliateSuite struct{}

var _ = Suite(&HelperAffiliateSuite{})

func (HelperAffiliateSuite) TestSkimAffiliateFees(c *C) {
	ctx, mgr := setupManagerForTest(c)
	affAddr1 := GetRandomSWITCHLYAddress()
	affAddr2 := GetRandomSWITCHLYAddress()
	tx := GetRandomTx()
	signer, _ := GetRandomSWITCHLYAddress().AccAddress()

	// Check affiliate balances before skimming fee
	affAcctAddr1, err := affAddr1.AccAddress()
	c.Assert(err, IsNil)
	acct := mgr.Keeper().GetBalance(ctx, affAcctAddr1)
	c.Assert(acct.AmountOf(common.SwitchNative.Native()).String(), Equals, "0")

	affAcctAddr2, err := affAddr2.AccAddress()
	c.Assert(err, IsNil)
	acct2 := mgr.Keeper().GetBalance(ctx, affAcctAddr2)
	c.Assert(acct2.AmountOf(common.SwitchNative.Native()).String(), Equals, "0")

	memo := fmt.Sprintf("=:SWITCHLY.SWITCH:%s::%s/%s:100/50", GetRandomSWITCHLYAddress(), affAddr1, affAddr2)
	coin := common.NewCoin(common.SwitchNative, cosmos.NewUint(10*common.One))

	feeSkimmed, err := skimAffiliateFees(ctx, mgr, tx, signer, coin, memo)
	c.Assert(err, IsNil)
	c.Assert(feeSkimmed.String(), Equals, "15000000") // 150 basis points of 10 SWITCH

	// Check affiliate balances after skimming fee
	acct = mgr.Keeper().GetBalance(ctx, affAcctAddr1)
	c.Assert(acct.AmountOf(common.SwitchNative.Native()).String(), Equals, "10000000")
	acct2 = mgr.Keeper().GetBalance(ctx, affAcctAddr2)
	c.Assert(acct2.AmountOf(common.SwitchNative.Native()).String(), Equals, "5000000")

	// Use one switchlyname and one switch address
	tn := types.SWITCHName{Name: "t", Owner: affAcctAddr1, ExpireBlockHeight: 10000000, Aliases: []types.SWITCHNameAlias{{Chain: common.SWITCHLYChain, Address: affAddr1}}}
	mgr.Keeper().SetSWITCHName(ctx, tn)
	memo = fmt.Sprintf("=:SWITCHLY.SWITCH:%s::t/%s:100/50", GetRandomSWITCHLYAddress(), affAddr2)

	feeSkimmed, err = skimAffiliateFees(ctx, mgr, tx, signer, coin, memo)
	c.Assert(err, IsNil)
	c.Assert(feeSkimmed.String(), Equals, "15000000")

	// Check affiliate balances after skimming fee
	acct = mgr.Keeper().GetBalance(ctx, affAcctAddr1)
	c.Assert(acct.AmountOf(common.SwitchNative.Native()).String(), Equals, "20000000")
	acct2 = mgr.Keeper().GetBalance(ctx, affAcctAddr2)
	c.Assert(acct2.AmountOf(common.SwitchNative.Native()).String(), Equals, "10000000")

	// Set a preferred asset, make sure affiliate collector is updated
	tn.PreferredAsset = common.BTCAsset
	mgr.Keeper().SetSWITCHName(ctx, tn)
	tn, err = mgr.Keeper().GetSWITCHName(ctx, "t")
	c.Assert(err, IsNil)
	c.Assert(tn.PreferredAsset.String(), Equals, "BTC.BTC")
	c.Assert(mgr.Keeper().SWITCHNameExists(ctx, "t"), Equals, true)
	// Must have BTC alias
	c.Assert(tn.CanReceiveAffiliateFee(), Equals, false)
	tn.Aliases = append(tn.Aliases, types.SWITCHNameAlias{Chain: common.BTCChain, Address: GetRandomBTCAddress()})
	mgr.Keeper().SetSWITCHName(ctx, tn)
	c.Assert(mgr.Keeper().SWITCHNameExists(ctx, "t"), Equals, true)
	c.Assert(tn.CanReceiveAffiliateFee(), Equals, true)

	feeSkimmed, err = skimAffiliateFees(ctx, mgr, tx, signer, coin, memo)
	c.Assert(err, IsNil)
	c.Assert(feeSkimmed.String(), Equals, "15000000")

	// Check affiliate balances after skimming fee, affAcctAddr1's balance should be same
	// as before + affiliate collector module updated
	acct = mgr.Keeper().GetBalance(ctx, affAcctAddr1)
	c.Assert(acct.AmountOf(common.SwitchNative.Native()).String(), Equals, "20000000")

	// ac, err := mgr.Keeper().GetAffiliateCollector(ctx, affAcctAddr1)
	// c.Assert(err, IsNil)
	// c.Assert(ac.SwitchAmount.String(), Equals, "10000000")

	// affAcctAddr2's balance should be updated as normal
	acct2 = mgr.Keeper().GetBalance(ctx, affAcctAddr2)
	c.Assert(acct2.AmountOf(common.SwitchNative.Native()).String(), Equals, "15000000")
}
