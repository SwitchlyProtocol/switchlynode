package stellar

import (
	"gitlab.com/thorchain/thornode/common"
)

const (
	maxGasAmount       = 100000
	maxMemoLength      = 28
	maxRetries         = 3
	defaultTimeoutSecs = 300
	minTxValue         = 1000000 // 0.1 XLM in stroops
)

var (
	stellarAsset     = common.XLMAsset
)
