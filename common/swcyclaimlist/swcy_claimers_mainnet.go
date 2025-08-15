//go:build !mocknet && !stagenet
// +build !mocknet,!stagenet

package swcyclaimlist

import (
	_ "embed"
)

//go:embed swcy_claimers_mainnet.json
var SWCYClaimsListRaw []byte
