package thorchain

import (
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

type RouterUpgradeControllerTestSuite struct{}

var _ = Suite(&RouterUpgradeControllerTestSuite{})

func (s *RouterUpgradeControllerTestSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

func (s *RouterUpgradeControllerTestSuite) TestUpgradeProcess(c *C) {
	// create vault
	// create pool
	// create
	ctx, mgr := setupManagerForTest(c)
	ctx = ctx.WithBlockHeight(1024)
	c.Assert(mgr.Keeper().SaveNetworkFee(ctx, common.ETHChain, NetworkFee{
		Chain:              common.ETHChain,
		TransactionSize:    80000,
		TransactionFeeRate: 10,
	}), IsNil)
	c.Assert(mgr.Keeper().SaveNetworkFee(ctx, common.AVAXChain, NetworkFee{
		Chain:              common.AVAXChain,
		TransactionSize:    80000,
		TransactionFeeRate: 10,
	}), IsNil)
	activeNodes := make(NodeAccounts, 4)
	for i := 0; i < 4; i++ {
		activeNodes[i] = GetRandomValidatorNode(NodeActive)
		c.Assert(mgr.Keeper().SetNodeAccount(ctx, activeNodes[i]), IsNil)
	}
	oldContractAddr, err := common.NewAddress(ethOldRouter)
	c.Assert(err, IsNil)
	oldChainContract := ChainContract{
		Chain:  common.ETHChain,
		Router: oldContractAddr,
	}
	mgr.Keeper().SetChainContract(ctx, oldChainContract)
	usdtAsset, err := common.NewAsset("ETH.USDT-0XA3910454BF2CB59B8B3A401589A3BACC5CA42306")
	c.Assert(err, IsNil)

	funds := common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(usdtAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.BCHAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.LTCAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	}

	activeVault := NewVault(ctx.BlockHeight(), types.VaultStatus_ActiveVault, AsgardVault, GetRandomPubKey(), []string{
		common.ETHChain.String(), common.ETHChain.String(), common.BTCChain.String(),
		common.BCHChain.String(), common.LTCChain.String(), common.AVAXChain.String(),
	}, []ChainContract{oldChainContract})
	activeVault.AddFunds(funds)
	c.Assert(mgr.Keeper().SetVault(ctx, activeVault), IsNil)
	controller := NewRouterUpgradeController(mgr)

	// nothing should happen
	controller.Process(ctx)
	txOut, err := mgr.TxOutStore().GetBlockOut(ctx)
	c.Assert(err, IsNil)
	// make sure it is empty, means it didn't recall funds , didn't make any outbound
	c.Assert(txOut.IsEmpty(), Equals, true)

	// make sure contract has not changed
	asgards, err := mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	c.Assert(asgards[0].GetContract(common.ETHChain), Equals, oldChainContract)

	asgards, err = mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	c.Assert(asgards[0].GetContract(common.ETHChain), Equals, oldChainContract)

	// update contract
	ctx = ctx.WithBlockHeight(3048)
	mgr.Keeper().SetMimir(ctx, fmt.Sprintf(MimirUpgradeContractTemplate, "ETH"), 1)
	controller.Process(ctx)
	asgards, err = mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	// contract on asgard should have not been changed
	// contract will be update for the next asgard, when churn
	c.Assert(asgards[0].GetContract(common.ETHChain), Equals, oldChainContract)

	upgradeContractMimir, err := mgr.Keeper().GetMimir(ctx, fmt.Sprintf(MimirUpgradeContractTemplate, "ETH"))
	c.Assert(err, IsNil)
	c.Assert(upgradeContractMimir, Equals, int64(-1))

	// Test second chain

	// There is no router for AVAX yet, test adding a new one
	emptyChainContract := ChainContract{
		Chain:  "",
		Router: "",
	}
	avaxOldRouter = ""
	avaxNewRouter = "0xcbEAF3BDe82155F56486Fb5a1072cb8baAf547cc"

	// update contract
	ctx = ctx.WithBlockHeight(3048)
	mgr.Keeper().SetMimir(ctx, fmt.Sprintf(MimirUpgradeContractTemplate, "AVAX"), 1)
	controller.Process(ctx)
	asgards, err = mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	// contract on asgard should have not been changed
	// contract will be update for the next asgard, when churn
	c.Assert(asgards[0].GetContract(common.AVAXChain), Equals, emptyChainContract)

	upgradeContractMimir, err = mgr.Keeper().GetMimir(ctx, fmt.Sprintf(MimirUpgradeContractTemplate, "AVAX"))
	c.Assert(err, IsNil)
	c.Assert(upgradeContractMimir, Equals, int64(-1))

	// add avax funds
	avaxUSDT, _ := common.NewAsset("AVAX.USDT-0XC7198437980C041C805A1EDCBA50C1CE5DB95118")
	avaxFunds := common.Coins{
		common.NewCoin(common.AVAXAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(avaxUSDT, cosmos.NewUint(100*common.One)),
	}
	asgards, _ = mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	asgards[0].AddFunds(avaxFunds)
	c.Assert(mgr.Keeper().SetVault(ctx, asgards[0]), IsNil)

	// test updating router
	avaxOldRouter = "0xcbEAF3BDe82155F56486Fb5a1072cb8baAf547cc"
	avaxNewRouter = "0x17aB05351fC94a1a67Bf3f56DdbB941aE6c63E25"

	ctx = ctx.WithBlockHeight(6048)
	mgr.Keeper().SetMimir(ctx, fmt.Sprintf(MimirUpgradeContractTemplate, "AVAX"), 1)
	controller.Process(ctx)
	asgards, err = mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	c.Assert(err, IsNil)
	// contract on asgard should have not been changed
	// contract will be update for the next asgard, when churn
	c.Assert(asgards[0].GetContract(common.AVAXChain), Equals, emptyChainContract)
}
