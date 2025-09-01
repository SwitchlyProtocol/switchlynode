//go:build !mocknet
// +build !mocknet

package stellar

import "github.com/switchlyprotocol/switchlynode/v3/common"

const (
	// Production fallback fee: 100 stroops (0.00001 XLM)
	// This ensures we always have a valid fee even when the cache is empty
	initialStellarFee = 100 * common.One
)
