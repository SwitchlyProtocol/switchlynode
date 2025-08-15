package stellar

import (
	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	stypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	switchlytypes "github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// MockSwitchlyBridge is a mock implementation for testing
type MockSwitchlyBridge struct{}

func (m *MockSwitchlyBridge) EnsureNodeWhitelisted() error {
	return nil
}

func (m *MockSwitchlyBridge) EnsureNodeWhitelistedWithTimeout() error {
	return nil
}

func (m *MockSwitchlyBridge) FetchNodeStatus() (switchlytypes.NodeStatus, error) {
	return switchlytypes.NodeStatus_Unknown, nil
}

func (m *MockSwitchlyBridge) FetchActiveNodes() ([]common.PubKey, error) {
	return []common.PubKey{}, nil
}

func (m *MockSwitchlyBridge) GetAsgards() (switchlytypes.Vaults, error) {
	return switchlytypes.Vaults{}, nil
}

func (m *MockSwitchlyBridge) GetVault(pubkey string) (switchlytypes.Vault, error) {
	return switchlytypes.Vault{}, nil
}

func (m *MockSwitchlyBridge) GetConfig() config.BifrostClientConfiguration {
	return config.BifrostClientConfiguration{}
}

func (m *MockSwitchlyBridge) GetConstants() (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *MockSwitchlyBridge) GetContext() client.Context {
	return client.Context{}
}

func (m *MockSwitchlyBridge) GetContractAddress() ([]switchlyclient.PubKeyContractAddressPair, error) {
	return []switchlyclient.PubKeyContractAddressPair{}, nil
}

func (m *MockSwitchlyBridge) GetErrataMsg(txID common.TxID, chain common.Chain) sdk.Msg {
	return nil
}

func (m *MockSwitchlyBridge) GetKeygenStdTx(poolPubKey common.PubKey, secp256k1Signature, keysharesBackup []byte, blame switchlytypes.Blame, inputPks common.PubKeys, keygenType switchlytypes.KeygenType, chains common.Chains, height, keygenTime int64) (sdk.Msg, error) {
	return nil, nil
}

func (m *MockSwitchlyBridge) GetKeysignParty(vaultPubKey common.PubKey) (common.PubKeys, error) {
	return common.PubKeys{}, nil
}

func (m *MockSwitchlyBridge) GetMimir(key string) (int64, error) {
	return 0, nil
}

func (m *MockSwitchlyBridge) GetMimirWithRef(template, ref string) (int64, error) {
	return 0, nil
}

func (m *MockSwitchlyBridge) GetInboundOutbound(txIns common.ObservedTxs) (common.ObservedTxs, common.ObservedTxs, error) {
	return common.ObservedTxs{}, common.ObservedTxs{}, nil
}

func (m *MockSwitchlyBridge) GetPools() (switchlytypes.Pools, error) {
	return switchlytypes.Pools{}, nil
}

func (m *MockSwitchlyBridge) GetPubKeys() ([]switchlyclient.PubKeyContractAddressPair, error) {
	return []switchlyclient.PubKeyContractAddressPair{}, nil
}

func (m *MockSwitchlyBridge) GetAsgardPubKeys() ([]switchlyclient.PubKeyContractAddressPair, error) {
	return []switchlyclient.PubKeyContractAddressPair{}, nil
}

func (m *MockSwitchlyBridge) GetSolvencyMsg(height int64, chain common.Chain, pubKey common.PubKey, coins common.Coins) *switchlytypes.MsgSolvency {
	return &switchlytypes.MsgSolvency{}
}

func (m *MockSwitchlyBridge) GetSWITCHName(name string) (switchlytypes.SWITCHName, error) {
	return switchlytypes.SWITCHName{}, nil
}

func (m *MockSwitchlyBridge) GetSwitchlyVersion() (semver.Version, error) {
	return semver.Version{}, nil
}

func (m *MockSwitchlyBridge) IsCatchingUp() (bool, error) {
	return false, nil
}

func (m *MockSwitchlyBridge) HasNetworkFee(chain common.Chain) (bool, error) {
	return false, nil
}

func (m *MockSwitchlyBridge) GetNetworkFee(chain common.Chain) (transactionSize, transactionFeeRate uint64, err error) {
	return 1, 100, nil
}

func (m *MockSwitchlyBridge) PostKeysignFailure(blame switchlytypes.Blame, height int64, memo string, coins common.Coins, pubkey common.PubKey) (common.TxID, error) {
	return common.TxID(""), nil
}

func (m *MockSwitchlyBridge) PostNetworkFee(height int64, chain common.Chain, transactionSize, transactionRate uint64) (common.TxID, error) {
	return common.TxID(""), nil
}

func (m *MockSwitchlyBridge) RagnarokInProgress() (bool, error) {
	return false, nil
}

func (m *MockSwitchlyBridge) WaitToCatchUp() error {
	return nil
}

func (m *MockSwitchlyBridge) GetBlockHeight() (int64, error) {
	return 1, nil
}

func (m *MockSwitchlyBridge) GetLastObservedInHeight(chain common.Chain) (int64, error) {
	return 1, nil
}

func (m *MockSwitchlyBridge) GetLastSignedOutHeight(chain common.Chain) (int64, error) {
	return 1, nil
}

func (m *MockSwitchlyBridge) Broadcast(msgs ...sdk.Msg) (common.TxID, error) {
	return common.TxID(""), nil
}

func (m *MockSwitchlyBridge) GetKeysign(blockHeight int64, pk string) (stypes.TxOut, error) {
	return stypes.TxOut{}, nil
}

func (m *MockSwitchlyBridge) GetNodeAccount(string) (*switchlytypes.NodeAccount, error) {
	return &switchlytypes.NodeAccount{}, nil
}

func (m *MockSwitchlyBridge) GetNodeAccounts() ([]*switchlytypes.NodeAccount, error) {
	return []*switchlytypes.NodeAccount{}, nil
}

func (m *MockSwitchlyBridge) GetKeygenBlock(int64, string) (switchlytypes.KeygenBlock, error) {
	return switchlytypes.KeygenBlock{}, nil
}
