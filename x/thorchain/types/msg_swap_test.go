package types

import (
	. "gopkg.in/check.v1"

	"fmt"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type MsgSwapSuite struct{}

var _ = Suite(&MsgSwapSuite{})

func (MsgSwapSuite) TestMsgSwap(c *C) {
	addr := GetRandomBech32Addr()
	c.Check(addr.Empty(), Equals, false)
	ethAddress := GetRandomETHAddress()
	txID := GetRandomTxHash()
	c.Check(txID.IsEmpty(), Equals, false)

	tx := common.NewTx(
		txID,
		GetRandomETHAddress(),
		GetRandomETHAddress(),
		common.Coins{
			common.NewCoin(common.BTCAsset, cosmos.NewUint(1)),
		},
		common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
		"SWAP:BTC.BTC",
	)

	m := NewMsgSwap(tx, common.ETHAsset, ethAddress, cosmos.NewUint(200000000), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, addr)
	EnsureMsgBasicCorrect(m, c)

	inputs := []struct {
		requestTxHash         common.TxID
		source                common.Asset
		target                common.Asset
		amount                cosmos.Uint
		requester             common.Address
		destination           common.Address
		targetPrice           cosmos.Uint
		signer                cosmos.AccAddress
		aggregator            common.Address
		aggregatorTarget      common.Address
		aggregatorTargetLimit cosmos.Uint
	}{
		{
			requestTxHash: common.TxID(""),
			source:        common.SWTCNative,
			target:        common.ETHAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     ethAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.Asset{},
			target:        common.ETHAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     ethAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.ETHAsset,
			target:        common.ETHAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     ethAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.SWTCNative,
			target:        common.Asset{},
			amount:        cosmos.NewUint(100000000),
			requester:     ethAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.SWTCNative,
			target:        common.ETHAsset,
			amount:        cosmos.ZeroUint(),
			requester:     ethAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.SWTCNative,
			target:        common.ETHAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     common.NoAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.SWTCNative,
			target:        common.ETHAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     ethAddress,
			destination:   common.NoAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        addr,
		},
		{
			requestTxHash: txID,
			source:        common.SWTCNative,
			target:        common.ETHAsset,
			amount:        cosmos.NewUint(100000000),
			requester:     ethAddress,
			destination:   ethAddress,
			targetPrice:   cosmos.NewUint(200000000),
			signer:        cosmos.AccAddress{},
		},
	}
	for _, item := range inputs {
		tx = common.NewTx(
			item.requestTxHash,
			item.requester,
			GetRandomETHAddress(),
			common.Coins{
				common.NewCoin(item.source, item.amount),
			},
			common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
			"SWAP:BTC.BTC",
		)

		m = NewMsgSwap(tx, item.target, item.destination, item.targetPrice, common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, item.signer)
		c.Assert(m.ValidateBasic(), NotNil)
	}

	// happy path
	m = NewMsgSwap(tx, common.ETHAsset, GetRandomETHAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "123", "0x123456", nil, 0, 10, 20, addr)
	c.Assert(m.ValidateBasic(), IsNil)
	c.Check(m.Aggregator, Equals, "123")
	c.Check(m.AggregatorTargetAddress, Equals, "0x123456")
	c.Check(m.AggregatorTargetLimit, IsNil)

	// test address and synth swapping fails when appropriate
	m = NewMsgSwap(tx, common.ETHAsset, GetRandomTHORAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, addr)
	c.Assert(m.ValidateBasic(), NotNil)
	m = NewMsgSwap(tx, common.ETHAsset.GetSyntheticAsset(), GetRandomTHORAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, addr)
	c.Assert(m.ValidateBasic(), IsNil)
	m = NewMsgSwap(tx, common.ETHAsset.GetSyntheticAsset(), GetRandomETHAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, addr)
	c.Assert(m.ValidateBasic(), NotNil)

	// affiliate fee basis point larger than 1000 should be rejected
	m = NewMsgSwap(tx, common.ETHAsset, GetRandomETHAddress(), cosmos.ZeroUint(), GetRandomTHORAddress(), cosmos.NewUint(1024), "", "", nil, 0, 0, 0, addr)
	c.Assert(m.ValidateBasic(), NotNil)

	// Define test addresses
	bnbAddr := GetRandomBNBAddress()
	thorAddr := GetRandomTHORAddress()

	for name, tc := range map[string]struct {
		source        common.Asset
		dest          common.Asset
		destAddr      common.Address
		tradeTarget   cosmos.Uint
		expectedError error
	}{
		"swap RUNE for BNB": {
			source:        common.SWTCNative,
			dest:          common.BNBBEP20Asset,
			destAddr:      bnbAddr,
			tradeTarget:   cosmos.ZeroUint(),
			expectedError: nil,
		},
		"swap BNB for RUNE": {
			source:        common.BNBBEP20Asset,
			dest:          common.SWTCNative,
			destAddr:      thorAddr,
			tradeTarget:   cosmos.ZeroUint(),
			expectedError: nil,
		},
		"swap RUNE for RUNE": {
			source:        common.SWTCNative,
			dest:          common.SWTCNative,
			destAddr:      thorAddr,
			tradeTarget:   cosmos.ZeroUint(),
			expectedError: fmt.Errorf("invalid message"),
		},
		"swap RUNE for BNB with trade target": {
			source:        common.SWTCNative,
			dest:          common.BNBBEP20Asset,
			destAddr:      bnbAddr,
			tradeTarget:   cosmos.NewUint(100),
			expectedError: nil,
		},
		"swap BNB for RUNE with trade target": {
			source:        common.BNBBEP20Asset,
			dest:          common.SWTCNative,
			destAddr:      thorAddr,
			tradeTarget:   cosmos.NewUint(100),
			expectedError: nil,
		},
		"swap RUNE for RUNE with trade target": {
			source:        common.SWTCNative,
			dest:          common.SWTCNative,
			destAddr:      thorAddr,
			tradeTarget:   cosmos.NewUint(100),
			expectedError: fmt.Errorf("invalid message"),
		},
	} {
		c.Logf("test case: %s", name)
		tx := common.NewTx(
			GetRandomTxHash(),
			GetRandomTHORAddress(),
			GetRandomTHORAddress(),
			common.Coins{
				common.NewCoin(tc.source, cosmos.NewUint(common.One)),
			},
			common.Gas{common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One))},
			fmt.Sprintf("SWAP:%s:%s", tc.dest.String(), tc.destAddr.String()),
		)
		m := NewMsgSwap(tx, tc.dest, tc.destAddr, tc.tradeTarget, common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, addr)
		if tc.expectedError != nil {
			c.Check(m.ValidateBasic(), NotNil)
		} else {
			c.Check(m.ValidateBasic(), IsNil)
		}
	}
}
