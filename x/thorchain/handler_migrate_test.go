package thorchain

import (
	"errors"
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

type HandlerMigrateSuite struct{}

var _ = Suite(&HandlerMigrateSuite{})

type TestMigrateKeeper struct {
	keeper.KVStoreDummy
	activeNodeAccount NodeAccount
	vault             Vault
}

// GetNodeAccount
func (k *TestMigrateKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, nil
}

func (HandlerMigrateSuite) TestMigrate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestMigrateKeeper{
		activeNodeAccount: GetRandomValidatorNode(NodeActive),
		vault:             GetRandomVault(),
	}

	handler := NewMigrateHandler(NewDummyMgrWithKeeper(keeper))

	addr, err := keeper.vault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)

	tx := NewObservedTx(common.Tx{
		ID:          GetRandomTxHash(),
		Chain:       common.ETHChain,
		Coins:       common.Coins{common.NewCoin(common.ETHAsset, cosmos.NewUint(1*common.One))},
		Memo:        "",
		FromAddress: GetRandomETHAddress(),
		ToAddress:   addr,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, 12, GetRandomPubKey(), 12)

	msgMigrate := NewMsgMigrate(tx, 1, keeper.activeNodeAccount.NodeAddress)
	err = handler.validate(ctx, *msgMigrate)
	c.Assert(err, IsNil)

	// invalid msg
	msgMigrate = &MsgMigrate{}
	err = handler.validate(ctx, *msgMigrate)
	c.Assert(err, NotNil)
}

type TestMigrateKeeperHappyPath struct {
	keeper.Keeper
	activeNodeAccount NodeAccount
	newVault          Vault
	retireVault       Vault
	txout             *TxOut
	pool              Pool
}

func (k *TestMigrateKeeperHappyPath) GetVault(_ cosmos.Context, pk common.PubKey) (Vault, error) {
	if pk.Equals(k.retireVault.PubKey) {
		return k.retireVault, nil
	}
	if pk.Equals(k.newVault.PubKey) {
		return k.newVault, nil
	}
	return Vault{}, fmt.Errorf("vault not found")
}

func (k *TestMigrateKeeperHappyPath) GetTxOut(ctx cosmos.Context, blockHeight int64) (*TxOut, error) {
	if k.txout != nil && k.txout.Height == blockHeight {
		return k.txout, nil
	}
	return nil, errKaboom
}

func (k *TestMigrateKeeperHappyPath) SetTxOut(ctx cosmos.Context, blockOut *TxOut) error {
	if k.txout.Height == blockOut.Height {
		k.txout = blockOut
		return nil
	}
	return errKaboom
}

func (k *TestMigrateKeeperHappyPath) GetNodeAccountByPubKey(_ cosmos.Context, _ common.PubKey) (NodeAccount, error) {
	return k.activeNodeAccount, nil
}

func (k *TestMigrateKeeperHappyPath) SetNodeAccount(_ cosmos.Context, na NodeAccount) error {
	k.activeNodeAccount = na
	return nil
}

func (k *TestMigrateKeeperHappyPath) GetPool(_ cosmos.Context, _ common.Asset) (Pool, error) {
	return k.pool, nil
}

func (k *TestMigrateKeeperHappyPath) SetPool(_ cosmos.Context, p Pool) error {
	k.pool = p
	return nil
}

func (HandlerMigrateSuite) TestMigrateHappyPath(c *C) {
	ctx, k := setupKeeperForTest(c)
	retireVault := GetRandomVault()

	newVault := GetRandomVault()
	txout := NewTxOut(1)
	newVaultAddr, err := newVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	txout.TxArray = append(txout.TxArray, TxOutItem{
		Chain:       common.ETHChain,
		InHash:      common.BlankTxID,
		ToAddress:   newVaultAddr,
		VaultPubKey: retireVault.PubKey,
		Coin:        common.NewCoin(common.ETHAsset, cosmos.NewUint(1024)),
		Memo:        NewMigrateMemo(1).String(),
	})
	keeper := &TestMigrateKeeperHappyPath{
		Keeper:            k,
		activeNodeAccount: GetRandomValidatorNode(NodeActive),
		newVault:          newVault,
		retireVault:       retireVault,
		txout:             txout,
	}
	addr, err := keeper.retireVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	handler := NewMigrateHandler(NewDummyMgrWithKeeper(keeper))
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.ETHChain,
		Coins: common.Coins{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewMigrateMemo(1).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, 1, retireVault.PubKey, 1)

	msgMigrate := NewMsgMigrate(tx, 1, keeper.activeNodeAccount.NodeAddress)
	_, err = handler.Run(ctx, msgMigrate)
	c.Assert(err, IsNil)
	c.Assert(keeper.txout.TxArray[0].OutHash.Equals(tx.Tx.ID), Equals, true)
}

func (HandlerMigrateSuite) TestSlash(c *C) {
	ctx, k := setupKeeperForTest(c)
	retireVault := GetRandomVault()
	newVault := GetRandomVault()
	vaultCoins := common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(2*common.One)),
	}
	retireVault.AddFunds(vaultCoins)
	txout := NewTxOut(1)
	newVaultAddr, err := newVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)

	pool := NewPool()
	pool.Asset = common.ETHAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	na := GetRandomValidatorNode(NodeActive)
	na.Bond = cosmos.NewUint(100 * common.One)
	retireVault.Membership = []string{
		na.PubKeySet.Secp256k1.String(),
	}
	keeper := &TestMigrateKeeperHappyPath{
		Keeper:            k,
		activeNodeAccount: na,
		newVault:          newVault,
		retireVault:       retireVault,
		txout:             txout,
		pool:              pool,
	}
	addr, err := keeper.retireVault.PubKey.GetAddress(common.ETHChain)
	c.Assert(err, IsNil)
	mgr := NewDummyMgrWithKeeper(keeper)
	mgr.slasher = newSlasherVCUR(keeper, NewDummyEventMgr())
	handler := NewMigrateHandler(mgr)
	tx := NewObservedTx(common.Tx{
		ID:    GetRandomTxHash(),
		Chain: common.ETHChain,
		Coins: common.Coins{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(1024)),
		},
		Memo:        NewMigrateMemo(1).String(),
		FromAddress: addr,
		ToAddress:   newVaultAddr,
		Gas: common.Gas{
			common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
		},
	}, 1, retireVault.PubKey, 1)

	msgMigrate := NewMsgMigrate(tx, 1, keeper.activeNodeAccount.NodeAddress)
	_, err = handler.handle(ctx, *msgMigrate)
	c.Assert(err, IsNil)
	c.Assert(keeper.activeNodeAccount.Bond, DeepEquals, cosmos.NewUint(9999983464))
}

func (HandlerMigrateSuite) TestHandlerMigrateValidation(c *C) {
	// invalid message should return an error
	ctx, mgr := setupManagerForTest(c)
	h := NewMigrateHandler(mgr)
	result, err := h.Run(ctx, NewMsgNetworkFee(ctx.BlockHeight(), common.ETHChain, 1, 10000, GetRandomBech32Addr()))
	c.Check(err, NotNil)
	c.Check(result, IsNil)
	c.Check(errors.Is(err, errInvalidMessage), Equals, true)
}
