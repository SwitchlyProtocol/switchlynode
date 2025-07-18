package types

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type MsgAddLiquiditySuite struct{}

var _ = Suite(&MsgAddLiquiditySuite{})

func (MsgAddLiquiditySuite) TestMsgAddLiquidity(c *C) {
	addr := GetRandomBech32Addr()
	c.Check(addr.Empty(), Equals, false)
	runeAddress := GetRandomRUNEAddress()
	assetAddress := GetRandomETHAddress()
	txID := GetRandomTxHash()
	c.Check(txID.IsEmpty(), Equals, false)
	tx := common.NewTx(
		txID,
		runeAddress,
		GetRandomRUNEAddress(),
		common.Coins{
			common.NewCoin(common.BTCAsset, cosmos.NewUint(100000000)),
		},
		common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
		"",
	)
	m := NewMsgAddLiquidity(tx, common.ETHAsset, cosmos.NewUint(100000000), cosmos.NewUint(100000000), runeAddress, assetAddress, common.NoAddress, cosmos.ZeroUint(), addr)
	EnsureMsgBasicCorrect(m, c)

	inputs := []struct {
		asset     common.Asset
		r         cosmos.Uint
		amt       cosmos.Uint
		runeAddr  common.Address
		assetAddr common.Address
		txHash    common.TxID
		signer    cosmos.AccAddress
	}{
		{
			asset:     common.Asset{},
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.ETHAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  common.NoAddress,
			assetAddr: common.NoAddress,
			txHash:    txID,
			signer:    addr,
		},
		{
			asset:     common.ETHAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: assetAddress,
			txHash:    common.TxID(""),
			signer:    addr,
		},
		{
			asset:     common.ETHAsset,
			r:         cosmos.NewUint(100000000),
			amt:       cosmos.NewUint(100000000),
			runeAddr:  runeAddress,
			assetAddr: assetAddress,
			txHash:    txID,
			signer:    cosmos.AccAddress{},
		},
	}
	for i, item := range inputs {
		tx = common.NewTx(
			item.txHash,
			GetRandomRUNEAddress(),
			GetRandomETHAddress(),
			common.Coins{
				common.NewCoin(item.asset, item.r),
			},
			common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
			"",
		)
		m = NewMsgAddLiquidity(tx, item.asset, item.r, item.amt, item.runeAddr, item.assetAddr, common.NoAddress, cosmos.ZeroUint(), item.signer)
		c.Assert(m.ValidateBasic(), NotNil, Commentf("%d) %s\n", i, m))
	}
	// If affiliate fee basis point is more than 1000 , the message should be rejected
	m1 := NewMsgAddLiquidity(tx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), GetRandomTHORAddress(), GetRandomETHAddress(), GetRandomTHORAddress(), cosmos.NewUint(1024), GetRandomBech32Addr())
	c.Assert(m1.ValidateBasic(), NotNil)

	// check that we can have zero asset and zero rune amounts
	m1 = NewMsgAddLiquidity(tx, common.ETHAsset, cosmos.ZeroUint(), cosmos.ZeroUint(), GetRandomTHORAddress(), GetRandomETHAddress(), GetRandomTHORAddress(), cosmos.ZeroUint(), GetRandomBech32Addr())
	c.Assert(m1.ValidateBasic(), IsNil)
}
