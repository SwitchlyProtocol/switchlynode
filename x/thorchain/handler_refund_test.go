package thorchain

import (
	"errors"

	"github.com/blang/semver"
	se "github.com/cosmos/cosmos-sdk/types/errors"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

type HandlerRefundSuite struct{}

var _ = Suite(&HandlerRefundSuite{})

type refundTxHandlerTestHelper struct {
	ctx           cosmos.Context
	pool          Pool
	version       semver.Version
	keeper        *refundTxHandlerKeeperTestHelper
	asgardVault   Vault
	yggVault      Vault
	constAccessor constants.ConstantValues
	nodeAccount   NodeAccount
	inboundTx     ObservedTx
	toi           TxOutItem
	mgr           Manager
}

type refundTxHandlerKeeperTestHelper struct {
	keeper.Keeper
	observeTxVoterErrHash common.TxID
	errGetTxOut           bool
	errGetNodeAccount     bool
	errGetPool            bool
	errSetPool            bool
	errSetNodeAccount     bool
	errGetNetwork         bool
	errSetNetwork         bool
	vault                 Vault
}

func newRefundTxHandlerKeeperTestHelper(keeper keeper.Keeper) *refundTxHandlerKeeperTestHelper {
	return &refundTxHandlerKeeperTestHelper{
		Keeper:                keeper,
		observeTxVoterErrHash: GetRandomTxHash(),
	}
}

func (k *refundTxHandlerKeeperTestHelper) GetObservedTxInVoter(ctx cosmos.Context, hash common.TxID) (ObservedTxVoter, error) {
	if hash.Equals(k.observeTxVoterErrHash) {
		return ObservedTxVoter{}, errKaboom
	}
	return k.Keeper.GetObservedTxOutVoter(ctx, hash)
}

func (k *refundTxHandlerKeeperTestHelper) GetTxOut(ctx cosmos.Context, height int64) (*TxOut, error) {
	if k.errGetTxOut {
		return nil, errKaboom
	}
	return k.Keeper.GetTxOut(ctx, height)
}

func (k *refundTxHandlerKeeperTestHelper) GetNodeAccountByPubKey(ctx cosmos.Context, pk common.PubKey) (NodeAccount, error) {
	if k.errGetNodeAccount {
		return NodeAccount{}, errKaboom
	}
	return k.Keeper.GetNodeAccountByPubKey(ctx, pk)
}

func (k *refundTxHandlerKeeperTestHelper) GetPool(ctx cosmos.Context, asset common.Asset) (Pool, error) {
	if k.errGetPool {
		return NewPool(), errKaboom
	}
	return k.Keeper.GetPool(ctx, asset)
}

func (k *refundTxHandlerKeeperTestHelper) SetPool(ctx cosmos.Context, pool Pool) error {
	if k.errSetPool {
		return errKaboom
	}
	return k.Keeper.SetPool(ctx, pool)
}

func (k *refundTxHandlerKeeperTestHelper) SetNodeAccount(ctx cosmos.Context, na NodeAccount) error {
	if k.errSetNodeAccount {
		return errKaboom
	}
	return k.Keeper.SetNodeAccount(ctx, na)
}

func (k *refundTxHandlerKeeperTestHelper) GetVault(_ cosmos.Context, _ common.PubKey) (Vault, error) {
	return k.vault, nil
}

func (k *refundTxHandlerKeeperTestHelper) SetVault(_ cosmos.Context, v Vault) error {
	k.vault = v
	return nil
}

func (k *refundTxHandlerKeeperTestHelper) GetNetwork(ctx cosmos.Context) (Network, error) {
	if k.errGetNetwork {
		return Network{}, errKaboom
	}
	return k.Keeper.GetNetwork(ctx)
}

func (k *refundTxHandlerKeeperTestHelper) SetNetwork(ctx cosmos.Context, data Network) error {
	if k.errSetNetwork {
		return errKaboom
	}
	return k.Keeper.SetNetwork(ctx, data)
}

// newRefundTxHandlerTestHelper setup all the basic condition to test OutboundTxHandler
func newRefundTxHandlerTestHelper(c *C) refundTxHandlerTestHelper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(1023)
	pool := NewPool()
	pool.Asset = common.ETHAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pool.LPUnits = pool.BalanceRune

	version := GetCurrentVersion()
	asgardVault := GetRandomVault()
	asgardVault.Membership = []string{asgardVault.PubKey.String()}
	addr, err := asgardVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)

	yggVault := GetRandomVault()
	yggVault.Membership = []string{yggVault.PubKey.String()}
	vaultCoins := common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(2*common.One)),
	}
	yggVault.AddFunds(vaultCoins)

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.ETHChain,
		Coins:       vaultCoins,
		Memo:        "swap:RUNE-67C",
		FromAddress: GetRandomETHAddress(),
		ToAddress:   addr,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, 12, GetRandomPubKey(), 12)

	keeperTestHelper := newRefundTxHandlerKeeperTestHelper(k)
	keeperTestHelper.vault = yggVault

	mgr := NewDummyMgrWithKeeper(keeperTestHelper)
	mgr.slasher = newSlasherVCUR(keeperTestHelper, NewDummyEventMgr())

	nodeAccount := GetRandomValidatorNode(NodeActive)
	nodeAccount.NodeAddress, err = yggVault.PubKey.GetThorAddress()
	c.Assert(err, IsNil)
	nodeAccount.Bond = cosmos.NewUint(100 * common.One)
	FundModule(c, ctx, k, BondName, nodeAccount.Bond.Uint64())
	nodeAccount.PubKeySet = common.NewPubKeySet(yggVault.PubKey, yggVault.PubKey)
	c.Assert(keeperTestHelper.SetNodeAccount(ctx, nodeAccount), IsNil)

	c.Assert(keeperTestHelper.SetPool(ctx, pool), IsNil)

	voter := NewObservedTxVoter(tx.Tx.ID, make(ObservedTxs, 0))
	voter.Add(tx, nodeAccount.NodeAddress)
	voter.Tx = *voter.GetTx(NodeAccounts{nodeAccount})
	voter.Height = ctx.BlockHeight()
	voter.FinalisedHeight = ctx.BlockHeight()
	keeperTestHelper.SetObservedTxOutVoter(ctx, voter)

	constAccessor := constants.GetConstantValues(version)
	txOutStorage := newTxOutStorageVCUR(keeperTestHelper, constAccessor, NewDummyEventMgr(), newGasMgrVCUR(constAccessor, keeperTestHelper))
	toi := TxOutItem{
		Chain:       common.ETHChain,
		ToAddress:   tx.Tx.FromAddress,
		VaultPubKey: yggVault.PubKey,
		Coin:        common.NewCoin(common.ETHAsset, cosmos.NewUint(2*common.One)),
		Memo:        NewRefundMemo(tx.Tx.ID).String(),
		InHash:      tx.Tx.ID,
	}
	result, err := txOutStorage.TryAddTxOutItem(ctx, mgr, toi, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Check(result, Equals, true)

	return refundTxHandlerTestHelper{
		ctx:           ctx,
		pool:          pool,
		version:       version,
		keeper:        keeperTestHelper,
		asgardVault:   asgardVault,
		yggVault:      yggVault,
		nodeAccount:   nodeAccount,
		inboundTx:     tx,
		toi:           toi,
		constAccessor: constAccessor,
		mgr:           mgr,
	}
}

func (s *HandlerRefundSuite) TestRefundTxHandlerShouldUpdateTxOut(c *C) {
	testCases := []struct {
		name           string
		messageCreator func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg
		runner         func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error)
		expectedResult error
	}{
		{
			name: "invalid message should return an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgNoOp(GetRandomObservedTx(), helper.nodeAccount.NodeAddress, "")
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg)
			},
			expectedResult: errInvalidMessage,
		},
		{
			name: "fail to get observed TxVoter should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, helper.keeper.observeTxVoterErrHash, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg)
			},
			expectedResult: errInternal,
		},
		{
			name: "fail to get txout should result in an error",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				helper.keeper.errGetTxOut = true
				return handler.Run(helper.ctx, msg)
			},
			expectedResult: se.ErrUnknownRequest,
		},
		{
			name: "valid outbound message, no event, no txout",
			messageCreator: func(helper refundTxHandlerTestHelper, tx ObservedTx) cosmos.Msg {
				return NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
			},
			runner: func(handler RefundHandler, helper refundTxHandlerTestHelper, msg cosmos.Msg) (*cosmos.Result, error) {
				return handler.Run(helper.ctx, msg)
			},
			expectedResult: nil,
		},
	}

	for _, tc := range testCases {
		helper := newRefundTxHandlerTestHelper(c)
		handler := NewRefundHandler(helper.mgr)
		fromAddr, err := helper.yggVault.PubKey.GetAddress(common.ETHChain)
		c.Assert(err, IsNil)
		tx := NewObservedTx(common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.ETHChain,
			Coins: common.Coins{
				common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One)),
			},
			Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
			FromAddress: fromAddr,
			ToAddress:   helper.inboundTx.Tx.FromAddress,
			Gas: common.Gas{
				common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
			},
		}, helper.ctx.BlockHeight(), helper.yggVault.PubKey, helper.ctx.BlockHeight())
		msg := tc.messageCreator(helper, tx)
		_, err = tc.runner(handler, helper, msg)
		if tc.expectedResult == nil {
			c.Check(err, IsNil)
		} else {
			c.Check(errors.Is(err, tc.expectedResult), Equals, true, Commentf("name:%s", tc.name))
		}
	}
}

func (s *HandlerRefundSuite) TestRefundTxNormalCase(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.mgr)

	fromAddr, err := helper.yggVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.ETHChain,
		Coins: common.Coins{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(199925000)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, helper.ctx.BlockHeight(), helper.yggVault.PubKey, helper.ctx.BlockHeight())
	// valid outbound message, with event, with txout
	outMsg := NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg)
	c.Assert(err, IsNil)

	// txout should had been complete
	txOut, err := helper.keeper.GetTxOut(helper.ctx, helper.ctx.BlockHeight())
	c.Assert(err, IsNil)
	c.Assert(txOut.TxArray[0].OutHash.IsEmpty(), Equals, false)
}

func (s *HandlerRefundSuite) TestRefundTxHandlerSendExtraFundShouldBeSlashed(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.ETHChain,
		Coins: common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(2*common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, helper.ctx.BlockHeight(), helper.nodeAccount.PubKeySet.Secp256k1, helper.ctx.BlockHeight())
	// expectedBond := helper.nodeAccount.Bond.Sub(ETHGasFeeSingleton[0].Amount).MulUint64(3).QuoUint64(2)
	expectedBond := cosmos.NewUint(9999985000)
	expectedVaultTotalReserve := cosmos.NewUint(1000000079999)
	// valid outbound message, with event, with txout
	outMsg := NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(na.Bond, DeepEquals, expectedBond)
	newReserve := helper.keeper.GetRuneBalanceOfModule(helper.ctx, ReserveName)
	c.Log(newReserve.String())
	c.Assert(newReserve, DeepEquals, expectedVaultTotalReserve)
}

func (s *HandlerRefundSuite) TestOutboundTxHandlerSendAdditionalCoinsShouldBeSlashed(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.ETHChain,
		Coins: common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(1*common.One)),
			common.NewCoin(common.ETHAsset, cosmos.NewUint(1*common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, helper.ctx.BlockHeight(), helper.nodeAccount.PubKeySet.Secp256k1, helper.ctx.BlockHeight())
	expectedBond := cosmos.NewUint(9849987250)
	// slash one ETH and one rune
	outMsg := NewMsgRefundTx(tx, helper.inboundTx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(na.Bond, DeepEquals, expectedBond)
}

func (s *HandlerRefundSuite) TestOutboundTxHandlerInvalidObservedTxVoterShouldSlash(c *C) {
	helper := newRefundTxHandlerTestHelper(c)
	handler := NewRefundHandler(helper.mgr)
	fromAddr, err := helper.asgardVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.ETHChain,
		Coins: common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(1*common.One)),
			common.NewCoin(common.ETHAsset, cosmos.NewUint(1*common.One)),
		},
		Memo:        NewRefundMemo(helper.inboundTx.Tx.ID).String(),
		FromAddress: fromAddr,
		ToAddress:   helper.inboundTx.Tx.FromAddress,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, helper.ctx.BlockHeight(), helper.nodeAccount.PubKeySet.Secp256k1, helper.ctx.BlockHeight())

	expectedBond := cosmos.NewUint(9849987250)
	// expected 0.5 slashed RUNE be added to reserve
	expectedVaultTotalReserve := cosmos.NewUint(1000050079249)
	pool, err := helper.keeper.GetPool(helper.ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	poolETH := common.SafeSub(pool.BalanceAsset, cosmos.NewUint(common.One).AddUint64(10000))

	// given the outbound tx doesn't have relevant OservedTxVoter in system , thus it should be slashed with 1.5 * the full amount of assets
	outMsg := NewMsgRefundTx(tx, tx.Tx.ID, helper.nodeAccount.NodeAddress)
	_, err = handler.Run(helper.ctx, outMsg)
	c.Assert(err, IsNil)
	na, err := helper.keeper.GetNodeAccount(helper.ctx, helper.nodeAccount.NodeAddress)
	c.Assert(err, IsNil)
	c.Assert(na.Bond, DeepEquals, expectedBond)

	newReserve := helper.keeper.GetRuneBalanceOfModule(helper.ctx, ReserveName)
	c.Assert(newReserve, DeepEquals, expectedVaultTotalReserve)
	pool, err = helper.keeper.GetPool(helper.ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	newBalance := cosmos.NewUint(10099933501)
	c.Assert(pool.BalanceRune, DeepEquals, newBalance)
	c.Assert(pool.BalanceAsset, DeepEquals, poolETH)
}
