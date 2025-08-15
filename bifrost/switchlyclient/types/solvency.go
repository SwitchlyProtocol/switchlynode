package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
)

// Solvency structure is to hold all the information necessary to report solvency to SWITCHLYNode
type Solvency struct {
	Height int64
	Chain  common.Chain
	PubKey common.PubKey
	Coins  common.Coins
}
