package thorchain

import (
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

type NetworkMgrDummy struct {
	nas   NodeAccounts
	vault Vault
}

func NewNetworkMgrDummy() *NetworkMgrDummy {
	return &NetworkMgrDummy{}
}

func (vm *NetworkMgrDummy) CalcAnchor(_ cosmos.Context, _ Manager, _ common.Asset) (cosmos.Uint, cosmos.Uint, cosmos.Uint) {
	return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint()
}

func (vm *NetworkMgrDummy) SpawnDerivedAsset(ctx cosmos.Context, asset common.Asset, mgr Manager) {}

func (vm *NetworkMgrDummy) BeginBlock(ctx cosmos.Context, mgr Manager) error {
	return nil
}

func (vm *NetworkMgrDummy) EndBlock(ctx cosmos.Context, mgr Manager) error {
	return nil
}

func (vm *NetworkMgrDummy) TriggerKeygen(_ cosmos.Context, nas NodeAccounts) error {
	vm.nas = nas
	return nil
}

func (vm *NetworkMgrDummy) RotateVault(ctx cosmos.Context, vault Vault) error {
	vm.vault = vault
	return nil
}

func (vm *NetworkMgrDummy) UpdateNetwork(ctx cosmos.Context, constAccessor constants.ConstantValues, gasManager GasManager, eventMgr EventManager) error {
	return nil
}

func (vm *NetworkMgrDummy) RecallChainFunds(ctx cosmos.Context, chain common.Chain, mgr Manager, excludeNodeKeys common.PubKeys) error {
	return nil
}
