//go:build mocknet
// +build mocknet

package swcyclaimlist

import (
	_ "embed"
)

//go:embed swcy_claimers_mocknet.json
var SWCYClaimsListRaw []byte
