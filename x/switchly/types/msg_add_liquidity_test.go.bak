package types

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type MsgAddLiquiditySuite struct{}

var _ = Suite(&MsgAddLiquiditySuite{})

func (MsgAddLiquiditySuite) TestMsgAddLiquidity(c *C) {
	addr := GetRandomBech32Addr()
	c.Check(addr.Empty(), Equals, false)
	switchAddress := GetRandomSwitchAddress()
	assetAddress := GetRandomETHAddress()
	txID := GetRandomTxHash()
	c.Check(txID.IsEmpty(), Equals, false)
	tx := common.NewTx(
		txID,
		switchAddress,
		GetRandomSwitchAddress(),
		common.Coins{
			common.NewCoin(common.BTCAsset, cosmos.NewUint(100000000)),
		},
		common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
		"",
	)
	m := NewMsgAddLiquidity(tx, common.ETHAsset, cosmos.NewUint(100000000), cosmos.NewUint(100000000), switchAddress, assetAddress, common.NoAddress, cosmos.ZeroUint(), addr)
	EnsureMsgBasicCorrect(m, c)

	inputs := []struct {
		asset     common.Asset
		r         cosmos.Uint
		amt       cosmos.Uint
		switchAddr  common.Address
		assetAddr common.Address
		txHash    common.TxID
		signer    cosmos.AccAddress
	}{
		{
			asset:     common.Asset{},
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			switchAddr:  switchAddress,
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.ETHAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			switchAddr:  common.NoAddress,
			assetAddr: common.NoAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.ETHAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			switchAddr:  switchAddress,
			assetAddr: assetAddress,
			txHash:    common.TxID(""),
			signer:    addr,
		},
		{
			asset:     common.ETHAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			switchAddr:  switchAddress,
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    cosmos.AccAddress{},
		},
	}
	for i, item := range inputs {
		tx = common.NewTx(
			item.txHash,
			GetRandomSwitchAddress(),
			GetRandomETHAddress(),
			common.Coins{
				common.NewCoin(item.asset, item.r),
			},
			common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
			"",
		)
		m = NewMsgAddLiquidity(tx, item.asset, item.r, item.amt, item.switchAddr, item.assetAddr, common.NoAddress, cosmos.ZeroUint(), item.signer)
		c.Assert(m.ValidateBasic(), NotNil, Commentf("%d) %s\n", i, m))
	}
	// If affiliate fee basis point is more than 1000 , the message should be rejected
	m1 := NewMsgAddLiquidity(tx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), GetRandomSWITCHLYAddress(), GetRandomETHAddress(), GetRandomSWITCHLYAddress(), cosmos.NewUint(1024), GetRandomBech32Addr())
	c.Assert(m1.ValidateBasic(), NotNil)

	// check that we can have zero asset and zero switch amounts
	m1 = NewMsgAddLiquidity(tx, common.ETHAsset, cosmos.ZeroUint(), cosmos.ZeroUint(), GetRandomSWITCHLYAddress(), GetRandomETHAddress(), GetRandomSWITCHLYAddress(), cosmos.ZeroUint(), GetRandomBech32Addr())
	c.Assert(m1.ValidateBasic(), IsNil)
}
