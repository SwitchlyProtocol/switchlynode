package thorchain

import (
	"gitlab.com/thorchain/thornode/v3/common"
	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	. "gopkg.in/check.v1"
)

type HandlerSecuredAssetWithdraw struct{}

var _ = Suite(&HandlerSecuredAssetWithdraw{})

func (HandlerSecuredAssetWithdraw) TestSecuredAssetWithdraw(c *C) {
	ctx, mgr := setupManagerForTest(c)
	asset := common.BTCAsset
	addr := GetRandomBech32Addr()
	bc1Addr := GetRandomBTCAddress()
	dummyTx := common.Tx{ID: "test"}

	{
		msg := NewMsgSecuredAssetDeposit(asset, cosmos.NewUint(500), addr, addr, dummyTx)

		h := NewSecuredAssetDepositHandler(mgr)

		pool, err := mgr.K.GetPool(ctx, asset)
		c.Assert(err, IsNil)
		pool.Asset = asset
		err = mgr.K.SetPool(ctx, pool)
		c.Assert(err, IsNil)

		_, err = h.Run(ctx, msg)
		c.Assert(err, IsNil)

		bal := mgr.SecuredAssetManager().BalanceOf(ctx, asset, addr)
		c.Check(bal.String(), Equals, "500")

		vault := GetRandomVault()
		vault.Status = ActiveVault
		vault.Coins = common.Coins{
			common.NewCoin(asset, cosmos.NewUint(500*common.One)),
		}
		c.Assert(mgr.Keeper().SetVault(ctx, vault), IsNil)
	}

	c.Assert(mgr.Keeper().SaveNetworkFee(ctx, common.BTCChain, NetworkFee{
		Chain: common.BTCChain, TransactionSize: 80000, TransactionFeeRate: 30,
	}), IsNil)

	msg := NewMsgSecuredAssetWithdraw(asset.GetSecuredAsset(), cosmos.NewUint(350), bc1Addr, addr, dummyTx)

	h := NewSecuredAssetWithdrawHandler(mgr)
	_, err := h.Run(ctx, msg)
	c.Assert(err, IsNil)

	bal := mgr.SecuredAssetManager().BalanceOf(ctx, asset, addr)
	c.Check(bal.String(), Equals, "150")

	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
	c.Check(items[0].Coin.String(), Equals, "350 BTC.BTC")
	c.Check(items[0].ToAddress.String(), Equals, bc1Addr.String())

	bankBals := mgr.coinKeeper.GetAllBalances(ctx, addr)
	c.Check(bankBals.String(), Equals, "150btc-btc")
}
