package stellar

import (
	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	stypes "github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/config"
	thorchaintypes "github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

// MockThorchainBridge is a mock implementation for testing
type MockThorchainBridge struct{}

func (m *MockThorchainBridge) EnsureNodeWhitelisted() error {
	return nil
}

func (m *MockThorchainBridge) EnsureNodeWhitelistedWithTimeout() error {
	return nil
}

func (m *MockThorchainBridge) FetchNodeStatus() (thorchaintypes.NodeStatus, error) {
	return thorchaintypes.NodeStatus_Unknown, nil
}

func (m *MockThorchainBridge) FetchActiveNodes() ([]common.PubKey, error) {
	return []common.PubKey{}, nil
}

func (m *MockThorchainBridge) GetAsgards() (thorchaintypes.Vaults, error) {
	return thorchaintypes.Vaults{}, nil
}

func (m *MockThorchainBridge) GetVault(pubkey string) (thorchaintypes.Vault, error) {
	return thorchaintypes.Vault{}, nil
}

func (m *MockThorchainBridge) GetConfig() config.BifrostClientConfiguration {
	return config.BifrostClientConfiguration{}
}

func (m *MockThorchainBridge) GetConstants() (map[string]int64, error) {
	return map[string]int64{}, nil
}

func (m *MockThorchainBridge) GetContext() client.Context {
	return client.Context{}
}

func (m *MockThorchainBridge) GetContractAddress() ([]thorclient.PubKeyContractAddressPair, error) {
	return []thorclient.PubKeyContractAddressPair{}, nil
}

func (m *MockThorchainBridge) GetErrataMsg(txID common.TxID, chain common.Chain) sdk.Msg {
	return nil
}

func (m *MockThorchainBridge) GetKeygenStdTx(poolPubKey common.PubKey, secp256k1Signature, keysharesBackup []byte, blame thorchaintypes.Blame, inputPks common.PubKeys, keygenType thorchaintypes.KeygenType, chains common.Chains, height, keygenTime int64) (sdk.Msg, error) {
	return nil, nil
}

func (m *MockThorchainBridge) GetKeysignParty(vaultPubKey common.PubKey) (common.PubKeys, error) {
	return common.PubKeys{}, nil
}

func (m *MockThorchainBridge) GetMimir(key string) (int64, error) {
	return 0, nil
}

func (m *MockThorchainBridge) GetMimirWithRef(template, ref string) (int64, error) {
	return 0, nil
}

func (m *MockThorchainBridge) GetInboundOutbound(txIns common.ObservedTxs) (common.ObservedTxs, common.ObservedTxs, error) {
	return common.ObservedTxs{}, common.ObservedTxs{}, nil
}

func (m *MockThorchainBridge) GetPools() (thorchaintypes.Pools, error) {
	return thorchaintypes.Pools{}, nil
}

func (m *MockThorchainBridge) GetPubKeys() ([]thorclient.PubKeyContractAddressPair, error) {
	return []thorclient.PubKeyContractAddressPair{}, nil
}

func (m *MockThorchainBridge) GetAsgardPubKeys() ([]thorclient.PubKeyContractAddressPair, error) {
	return []thorclient.PubKeyContractAddressPair{}, nil
}

func (m *MockThorchainBridge) GetSolvencyMsg(height int64, chain common.Chain, pubKey common.PubKey, coins common.Coins) *thorchaintypes.MsgSolvency {
	return &thorchaintypes.MsgSolvency{}
}

func (m *MockThorchainBridge) GetTHORName(name string) (thorchaintypes.THORName, error) {
	return thorchaintypes.THORName{}, nil
}

func (m *MockThorchainBridge) GetThorchainVersion() (semver.Version, error) {
	return semver.Version{}, nil
}

func (m *MockThorchainBridge) IsCatchingUp() (bool, error) {
	return false, nil
}

func (m *MockThorchainBridge) HasNetworkFee(chain common.Chain) (bool, error) {
	return false, nil
}

func (m *MockThorchainBridge) GetNetworkFee(chain common.Chain) (transactionSize, transactionFeeRate uint64, err error) {
	return 1, 100, nil
}

func (m *MockThorchainBridge) PostKeysignFailure(blame thorchaintypes.Blame, height int64, memo string, coins common.Coins, pubkey common.PubKey) (common.TxID, error) {
	return common.TxID(""), nil
}

func (m *MockThorchainBridge) PostNetworkFee(height int64, chain common.Chain, transactionSize, transactionRate uint64) (common.TxID, error) {
	return common.TxID(""), nil
}

func (m *MockThorchainBridge) RagnarokInProgress() (bool, error) {
	return false, nil
}

func (m *MockThorchainBridge) WaitToCatchUp() error {
	return nil
}

func (m *MockThorchainBridge) GetBlockHeight() (int64, error) {
	return 1, nil
}

func (m *MockThorchainBridge) GetLastObservedInHeight(chain common.Chain) (int64, error) {
	return 1, nil
}

func (m *MockThorchainBridge) GetLastSignedOutHeight(chain common.Chain) (int64, error) {
	return 1, nil
}

func (m *MockThorchainBridge) Broadcast(msgs ...sdk.Msg) (common.TxID, error) {
	return common.TxID(""), nil
}

func (m *MockThorchainBridge) GetKeysign(blockHeight int64, pk string) (stypes.TxOut, error) {
	return stypes.TxOut{}, nil
}

func (m *MockThorchainBridge) GetNodeAccount(string) (*thorchaintypes.NodeAccount, error) {
	return &thorchaintypes.NodeAccount{}, nil
}

func (m *MockThorchainBridge) GetNodeAccounts() ([]*thorchaintypes.NodeAccount, error) {
	return []*thorchaintypes.NodeAccount{}, nil
}

func (m *MockThorchainBridge) GetKeygenBlock(int64, string) (thorchaintypes.KeygenBlock, error) {
	return thorchaintypes.KeygenBlock{}, nil
}
