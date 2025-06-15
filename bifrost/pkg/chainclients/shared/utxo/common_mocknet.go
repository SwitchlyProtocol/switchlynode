//go:build mocknet
// +build mocknet

package utxo

import (
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

func GetConfMulBasisPoint(chain string, bridge thorclient.ThorchainBridge) (cosmos.Uint, error) {
	return cosmos.NewUint(1), nil
}

func MaxConfAdjustment(confirm uint64, chain string, bridge thorclient.ThorchainBridge) (uint64, error) {
	return 1, nil
}
