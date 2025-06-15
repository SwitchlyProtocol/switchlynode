package common

import (
	"github.com/btcsuite/btcd/chaincfg"
	dogchaincfg "github.com/eager7/dogd/chaincfg"
	ltcchaincfg "github.com/ltcsuite/ltcd/chaincfg"
	. "gopkg.in/check.v1"
)

type ChainSuite struct{}

var _ = Suite(&ChainSuite{})

func (s ChainSuite) TestChain(c *C) {
	ethChain, err := NewChain("eth")
	c.Assert(err, IsNil)
	c.Check(ethChain.Equals(ETHChain), Equals, true)
	c.Check(ethChain.IsEmpty(), Equals, false)
	c.Check(ethChain.String(), Equals, "ETH")

	// Test StellarChain
	stellarChain, err := NewChain("xlm")
	c.Assert(err, IsNil)
	c.Check(stellarChain.Equals(StellarChain), Equals, true)
	c.Check(stellarChain.IsEmpty(), Equals, false)
	c.Check(stellarChain.String(), Equals, "XLM")

	_, err = NewChain("B") // too short
	c.Assert(err, NotNil)

	chains := Chains{"DOGE", "DOGE", "BTC"}
	c.Check(chains.Has("BTC"), Equals, true)
	c.Check(chains.Has("ETH"), Equals, false)
	uniq := chains.Distinct()
	c.Assert(uniq, HasLen, 2)

	algo := ETHChain.GetSigningAlgo()
	c.Assert(algo, Equals, SigningAlgoSecp256k1)

	c.Assert(BTCChain.GetGasAsset(), Equals, BTCAsset)
	c.Assert(ETHChain.GetGasAsset(), Equals, ETHAsset)
	c.Assert(LTCChain.GetGasAsset(), Equals, LTCAsset)
	c.Assert(BCHChain.GetGasAsset(), Equals, BCHAsset)
	c.Assert(DOGEChain.GetGasAsset(), Equals, DOGEAsset)
	c.Assert(StellarChain.GetGasAsset(), Equals, XLMAsset)
	c.Assert(EmptyChain.GetGasAsset(), Equals, EmptyAsset)

	// Test Stellar chain properties
	c.Assert(StellarChain.GetGasUnits(), Equals, "stroop")
	c.Assert(StellarChain.GetGasAssetDecimal(), Equals, int64(7))
	c.Assert(StellarChain.DustThreshold().String(), Equals, "100000000") // 1 XLM in stroops
	c.Assert(StellarChain.ApproximateBlockMilliseconds(), Equals, int64(5000))
	c.Assert(StellarChain.InboundNotes(), Equals, "Transfer the inbound_address the asset with the memo. Use MemoText for the memo field. Do not use multi-in, multi-out transactions.")

	c.Assert(BTCChain.AddressPrefix(MockNet), Equals, chaincfg.RegressionNetParams.Bech32HRPSegwit)
	c.Assert(BTCChain.AddressPrefix(MainNet), Equals, chaincfg.MainNetParams.Bech32HRPSegwit)
	c.Assert(BTCAsset.Chain.AddressPrefix(StageNet), Equals, chaincfg.MainNetParams.Bech32HRPSegwit)

	c.Assert(LTCChain.AddressPrefix(MockNet), Equals, ltcchaincfg.RegressionNetParams.Bech32HRPSegwit)
	c.Assert(LTCChain.AddressPrefix(MainNet), Equals, ltcchaincfg.MainNetParams.Bech32HRPSegwit)
	c.Assert(LTCChain.AddressPrefix(StageNet), Equals, ltcchaincfg.MainNetParams.Bech32HRPSegwit)

	c.Assert(DOGEChain.AddressPrefix(MockNet), Equals, dogchaincfg.RegressionNetParams.Bech32HRPSegwit)
	c.Assert(DOGEChain.AddressPrefix(MainNet), Equals, dogchaincfg.MainNetParams.Bech32HRPSegwit)
	c.Assert(DOGEChain.AddressPrefix(StageNet), Equals, dogchaincfg.MainNetParams.Bech32HRPSegwit)
}
