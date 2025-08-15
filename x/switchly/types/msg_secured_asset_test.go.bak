package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	. "gopkg.in/check.v1"
)

type MsgSecuredAssetSuite struct{}

var _ = Suite(&MsgSecuredAssetSuite{})

func (MsgSecuredAssetSuite) TestDeposit(c *C) {
	asset := common.ETHAsset
	amt := cosmos.NewUint(100)
	signer := GetRandomBech32Addr()
	dummyTx := common.Tx{ID: "test"}

	m := NewMsgSecuredAssetDeposit(asset, amt, signer, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m = NewMsgSecuredAssetDeposit(common.EmptyAsset, amt, signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgSecuredAssetDeposit(common.SwitchNative, amt, signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgSecuredAssetDeposit(asset, cosmos.ZeroUint(), signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)
}

func (MsgSecuredAssetSuite) TestWithdraw(c *C) {
	asset := common.ETHAsset.GetSecuredAsset()
	amt := cosmos.NewUint(100)
	ethAddr := GetRandomETHAddress()
	signer := GetRandomBech32Addr()
	dummyTx := common.Tx{ID: "test"}

	m := NewMsgSecuredAssetWithdraw(asset, amt, ethAddr, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m = NewMsgSecuredAssetWithdraw(common.EmptyAsset, amt, ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgSecuredAssetWithdraw(common.SwitchNative, amt, ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgSecuredAssetWithdraw(asset, cosmos.ZeroUint(), ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgSecuredAssetWithdraw(asset, cosmos.ZeroUint(), GetRandomSWITCHLYAddress(), signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)
}

func (s *MsgSecuredAssetSuite) TestMsgSecuredAssetDeposit(c *C) {
	signer := GetRandomBech32Addr()
	dummyTx := GetRandomTx()
	amt := cosmos.NewUint(100 * common.One)
	m := NewMsgSecuredAssetDeposit(common.ETHAsset, amt, signer, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m1 := NewMsgSecuredAssetDeposit(common.ETHAsset, amt, signer, signer, dummyTx)
	// Basic validation test
	c.Check(m.ValidateBasic(), IsNil)
	c.Check(m1.ValidateBasic(), IsNil)

	// ensure we can set the signer
	m.Signer = GetRandomBech32Addr()
	c.Check(m.Signer.Equals(m1.Signer), Equals, false)
}

func (s *MsgSecuredAssetSuite) TestMsgSecuredAssetWithdraw(c *C) {
	signer := GetRandomBech32Addr()
	ethAddr := GetRandomETHAddress()
	dummyTx := GetRandomTx()
	amt := cosmos.NewUint(100 * common.One)
	securedAsset := common.ETHAsset.GetSecuredAsset()
	m := NewMsgSecuredAssetWithdraw(securedAsset, amt, ethAddr, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m = NewMsgSecuredAssetWithdraw(securedAsset, amt, ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), IsNil)

	m = NewMsgSecuredAssetWithdraw(securedAsset, cosmos.ZeroUint(), ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgSecuredAssetWithdraw(securedAsset, cosmos.ZeroUint(), GetRandomSWITCHLYAddress(), signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)
}
