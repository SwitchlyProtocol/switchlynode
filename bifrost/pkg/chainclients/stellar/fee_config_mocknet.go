//go:build mocknet
// +build mocknet

package stellar

import "github.com/switchlyprotocol/switchlynode/v3/common"

const (
	// initialStellarFee overrides the initial stellar fee in mocknet to force a reported fee.
	// This ensures network fees are reported immediately without waiting for 200 transactions.
	initialStellarFee = 100 * common.One // 100 stroops (0.00001 XLM)
) 