//go:build !mocknet
// +build !mocknet

package evm

import "github.com/switchlyprotocol/switchlynode/v1/common"

// GetHeight returns the current block height.
func (e *EVMScanner) GetHeight() (int64, error) {
	var (
		height int64
		err    error
	)
	switch e.cfg.ChainID {
	case common.BASEChain:
		height, err = e.ethRpc.GetBlockHeightSafe()
	default:
		height, err = e.ethRpc.GetBlockHeight()
	}

	if err != nil {
		return -1, err
	}
	return height, nil
}
