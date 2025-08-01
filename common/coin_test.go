package common

import (
	. "gopkg.in/check.v1"

	cosmos "github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type CoinSuite struct{}

var _ = Suite(&CoinSuite{})

func (s CoinSuite) TestCoin(c *C) {
	coin := NewCoin(DOGEAsset, cosmos.NewUint(230000000))
	c.Check(coin.Asset.Equals(DOGEAsset), Equals, true)
	c.Check(coin.Amount.Uint64(), Equals, uint64(230000000))
	c.Check(coin.Valid(), IsNil)
	c.Check(coin.IsEmpty(), Equals, false)
	c.Check(NoCoin.IsEmpty(), Equals, true)

	c.Check(coin.IsNative(), Equals, false)
	_, err := coin.Native()
	c.Assert(err, NotNil)
	coin = NewCoin(SwitchNative, cosmos.NewUint(230))
	c.Check(coin.IsNative(), Equals, true)
	sdkCoin, err := coin.Native()
	c.Assert(err, IsNil)
	c.Check(sdkCoin.Denom, Equals, "switch")
	c.Check(sdkCoin.Amount.Equal(cosmos.NewInt(230)), Equals, true)
}

func (s CoinSuite) TestDistinct(c *C) {
	coins := Coins{
		NewCoin(DOGEAsset, cosmos.NewUint(1000)),
		NewCoin(DOGEAsset, cosmos.NewUint(1000)),
		NewCoin(BTCAsset, cosmos.NewUint(1000)),
		NewCoin(BTCAsset, cosmos.NewUint(1000)),
	}
	newCoins := coins.Distinct()
	c.Assert(len(newCoins), Equals, 2)
}

func (s CoinSuite) TestAdds(c *C) {
	oldCoins := Coins{
		NewCoin(DOGEAsset, cosmos.NewUint(1000)),
		NewCoin(BCHAsset, cosmos.NewUint(1000)),
	}
	newCoins := oldCoins.Add(NewCoins(
		NewCoin(DOGEAsset, cosmos.NewUint(1000)),
		NewCoin(BTCAsset, cosmos.NewUint(1000)),
	)...)

	c.Assert(len(newCoins), Equals, 3)
	c.Assert(len(oldCoins), Equals, 2)
	// oldCoins asset types are unchanged, while newCoins has all types.

	c.Check(newCoins.GetCoin(DOGEAsset).Amount.Uint64(), Equals, uint64(2000))
	c.Check(newCoins.GetCoin(BCHAsset).Amount.Uint64(), Equals, uint64(1000))
	c.Check(newCoins.GetCoin(BTCAsset).Amount.Uint64(), Equals, uint64(1000))
	// For newCoins, the amount adding works as expected.

	c.Check(oldCoins.GetCoin(DOGEAsset).Amount.Uint64(), Equals, uint64(1000))

	newerCoins := make(Coins, len(oldCoins))
	copy(newerCoins, oldCoins)
	newerCoins = newerCoins.Add(NewCoins(
		NewCoin(DOGEAsset, cosmos.NewUint(4000)),
	)...)
	c.Check(newerCoins.GetCoin(DOGEAsset).Amount.Uint64(), Equals, uint64(5000))
	c.Check(oldCoins.GetCoin(DOGEAsset).Amount.Uint64(), Equals, uint64(1000))
	// Having used make(Coins, len()) and copy(), oldCoins is unchanged.

	newAmount := oldCoins.GetCoin(DOGEAsset).Amount.Add(NewCoin(DOGEAsset, cosmos.NewUint(7000)).Amount)
	c.Check(newAmount.Uint64(), Equals, uint64(8000))
	c.Check(oldCoins.GetCoin(DOGEAsset).Amount.Uint64(), Equals, uint64(1000))
	// When Add alone is used with .Amount and no sanitisation,
	// the newAmount is as expected while the oldCoins amount is unaffected.
}

func (s CoinSuite) TestNoneEmpty(c *C) {
	coins := Coins{
		NewCoin(DOGEAsset, cosmos.NewUint(1000)),
		NewCoin(ETHAsset, cosmos.ZeroUint()),
	}
	newCoins := coins.NoneEmpty()
	c.Assert(newCoins, HasLen, 1)
	ethCoin := newCoins.GetCoin(ETHAsset)
	c.Assert(ethCoin.IsEmpty(), Equals, true)
}

func (s CoinSuite) TestHasSynthetic(c *C) {
	dogeSynthAsset, _ := NewAsset("DOGE/DOGE")
	coins := Coins{
		NewCoin(dogeSynthAsset, cosmos.NewUint(1000)),
		NewCoin(ETHAsset, cosmos.ZeroUint()),
	}
	c.Assert(coins.HasSynthetic(), Equals, true)
	coins = Coins{
		NewCoin(DOGEAsset, cosmos.NewUint(1000)),
		NewCoin(ETHAsset, cosmos.ZeroUint()),
	}
	c.Assert(coins.HasSynthetic(), Equals, false)
}
