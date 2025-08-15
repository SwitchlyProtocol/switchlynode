//go:build mocknet
// +build mocknet

package utxo

import (
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

func GetConfMulBasisPoint(chain string, bridge switchlyclient.SwitchlyBridge) (cosmos.Uint, error) {
	return cosmos.NewUint(1), nil
}

func MaxConfAdjustment(confirm uint64, chain string, bridge switchlyclient.SwitchlyBridge) (uint64, error) {
	return 1, nil
}
