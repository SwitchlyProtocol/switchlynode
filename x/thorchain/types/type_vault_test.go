package types

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

type VaultSuite struct{}

var _ = Suite(&VaultSuite{})

func (s *VaultSuite) TestVault(c *C) {
	pk := GetRandomPubKey()

	vault := Vault{}
	c.Check(vault.IsEmpty(), Equals, true)
	c.Check(vault.Valid(), NotNil)

	vault = NewVault(12, VaultStatus_ActiveVault, VaultType_AsgardVault, pk, common.Chains{common.ETHChain}.Strings(), []ChainContract{})
	c.Check(vault.PubKey.Equals(pk), Equals, true)
	c.Check(vault.HasFunds(), Equals, false)
	c.Check(vault.IsEmpty(), Equals, false)
	c.Check(vault.Valid(), IsNil)

	coins := common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(500*common.One)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(400*common.One)),
	}

	vault.AddFunds(coins)
	c.Check(vault.HasFunds(), Equals, true)
	c.Check(vault.HasFundsForChain(common.ETHChain), Equals, true)
	c.Check(vault.HasFundsForChain(common.GAIAChain), Equals, false)
	c.Check(vault.Coins, HasLen, 2)
	c.Check(vault.GetCoin(common.ETHAsset).Amount.Equal(cosmos.NewUint(500*common.One)), Equals, true)
	c.Check(vault.GetCoin(common.BTCAsset).Amount.Equal(cosmos.NewUint(400*common.One)), Equals, true)
	c.Check(vault.HasAsset(common.ETHAsset), Equals, true)
	vault.AddFunds(coins)
	c.Check(vault.Coins, HasLen, 2)
	c.Check(vault.GetCoin(common.ETHAsset).Amount.Equal(cosmos.NewUint(1000*common.One)), Equals, true)
	c.Check(vault.GetCoin(common.BTCAsset).Amount.Equal(cosmos.NewUint(800*common.One)), Equals, true, Commentf("%+v", vault.GetCoin(common.BTCAsset).Amount))
	vault.SubFunds(coins)
	c.Check(vault.Coins, HasLen, 2)
	c.Check(vault.GetCoin(common.ETHAsset).Amount.Equal(cosmos.NewUint(500*common.One)), Equals, true)
	c.Check(vault.GetCoin(common.BTCAsset).Amount.Equal(cosmos.NewUint(400*common.One)), Equals, true)
	vault.SubFunds(coins)
	c.Check(vault.Coins, HasLen, 2)
	c.Check(vault.GetCoin(common.ETHAsset).Amount.Equal(cosmos.ZeroUint()), Equals, true)
	c.Check(vault.GetCoin(common.BTCAsset).Amount.Equal(cosmos.ZeroUint()), Equals, true)
	c.Check(vault.HasFunds(), Equals, false)
	vault.SubFunds(coins)
	c.Check(vault.GetCoin(common.ETHAsset).Amount.Equal(cosmos.ZeroUint()), Equals, true)
	c.Check(vault.GetCoin(common.BTCAsset).Amount.Equal(cosmos.ZeroUint()), Equals, true)
	c.Check(vault.HasFunds(), Equals, false)
	vault.AddFunds(common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	})
	c.Assert(vault.GetChains().Has(common.ETHChain), Equals, true)
	c.Assert(vault.GetChains().Has(common.ETHChain), Equals, true)
	c.Assert(vault.GetChains().Has(common.BTCChain), Equals, true)

	vault1 := NewVault(1024, VaultStatus_ActiveVault, VaultType_AsgardVault, pk, common.Chains{common.ETHChain, common.BTCChain, common.ETHChain}.Strings(), []ChainContract{})
	vault1.Membership = append(vault.Membership, pk.String()) // nolint
	c.Check(vault1.IsType(VaultType_AsgardVault), Equals, true)
	c.Check(vault1.IsAsgard(), Equals, true)
	c.Check(vault1.Contains(pk), Equals, true)
	vault1.UpdateStatus(VaultStatus_RetiringVault, 10000)
	c.Check(vault1.CoinLength(), Equals, 0)
	c.Check(vault1.HasAsset(common.ETHAsset), Equals, false)
}

func (s *VaultSuite) TestGetTssSigners(c *C) {
	vault := NewVault(12, VaultStatus_ActiveVault, VaultType_AsgardVault, GetRandomPubKey(), common.Chains{common.ETHChain}.Strings(), []ChainContract{})
	nodeAccounts := NodeAccounts{}
	memberShip := make([]string, 0)
	for i := 0; i < 10; i++ {
		na := GetRandomValidatorNode(NodeStatus_Active)
		nodeAccounts = append(nodeAccounts, na)
		memberShip = append(memberShip, na.PubKeySet.Secp256k1.String())
	}
	vault.Membership = memberShip
	addrs := []cosmos.AccAddress{
		nodeAccounts[0].NodeAddress,
		nodeAccounts[1].NodeAddress,
	}
	keys, err := vault.GetMembers(addrs)
	c.Assert(err, IsNil)
	c.Assert(keys, HasLen, 2)
	c.Assert(keys[0].Equals(nodeAccounts[0].PubKeySet.Secp256k1), Equals, true)
	c.Assert(keys[1].Equals(nodeAccounts[1].PubKeySet.Secp256k1), Equals, true)
}

func (s *VaultSuite) TestPendingTxBlockHeights(c *C) {
	vault := NewVault(12, VaultStatus_ActiveVault, VaultType_AsgardVault, GetRandomPubKey(), common.Chains{common.ETHChain}.Strings(), []ChainContract{})

	version := GetCurrentVersion()
	constAccessor := constants.GetConstantValues(version)
	vault.AppendPendingTxBlockHeights(1, constAccessor)
	c.Assert(vault.LenPendingTxBlockHeights(2, 300), Equals, 1)
	c.Assert(vault.LenPendingTxBlockHeights(302, 300), Equals, 0)
	for i := 0; i < 100; i++ {
		vault.AppendPendingTxBlockHeights(int64(i), constAccessor)
	}
	c.Assert(vault.LenPendingTxBlockHeights(100, 300), Equals, 101)
	vault.AppendPendingTxBlockHeights(1000, constAccessor)
	c.Assert(vault.LenPendingTxBlockHeights(1001, 300), Equals, 1)
	vault.RemovePendingTxBlockHeights(1000)
	c.Assert(vault.LenPendingTxBlockHeights(1002, 300), Equals, 0)
	vault.RemovePendingTxBlockHeights(1001)
	c.Assert(vault.LenPendingTxBlockHeights(1002, 300), Equals, 0)
}

func (s *VaultSuite) TestVaultSort(c *C) {
	vault := NewVault(1024, VaultStatus_ActiveVault, VaultType_AsgardVault, GetRandomPubKey(),
		common.Chains{
			common.BTCChain,
			common.ETHChain,
			common.GAIAChain,
		}.Strings(), []ChainContract{})
	vault.AddFunds(common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One*100)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(common.One*50)),
		common.NewCoin(common.ATOMAsset, cosmos.NewUint(common.One*10)),
	})
	vault1 := NewVault(1024, VaultStatus_ActiveVault, VaultType_AsgardVault, GetRandomPubKey(),
		common.Chains{
			common.BTCChain,
			common.ETHChain,
			common.GAIAChain,
		}.Strings(), []ChainContract{})
	vault1.AddFunds(common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(common.One*90)),
		common.NewCoin(common.BTCAsset, cosmos.NewUint(common.One*90)),
		common.NewCoin(common.ATOMAsset, cosmos.NewUint(common.One*90)),
	})
	vaults := Vaults{
		vault, vault1,
	}
	vaults1 := vaults.SortBy(common.BTCAsset)
	c.Check(vaults1[0].PubKey.Equals(vault1.PubKey), Equals, true)
	atomVault := vaults.SelectByMaxCoin(common.ATOMAsset)
	c.Check(atomVault.PubKey.Equals(vault1.PubKey), Equals, true)

	ethVault := vaults.SelectByMinCoin(common.ETHAsset)
	c.Check(ethVault.PubKey.Equals(vault1.PubKey), Equals, true)
	addr, err := vault.PubKey.GetAddress(common.ETHChain)
	c.Check(err, IsNil)
	result, err := vaults.HasAddress(common.ETHChain, addr)
	c.Check(err, IsNil)
	c.Check(result, Equals, true)
	result1, err1 := vaults.HasAddress(common.BTCChain, GetRandomBTCAddress())
	c.Check(err1, IsNil)
	c.Check(result1, Equals, false)
}
