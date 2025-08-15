//go:build stagenet
// +build stagenet

package swcyclaimlist

import (
	_ "embed"
)

//go:embed swcy_claimers_stagenet.json
var SWCYClaimsListRaw []byte
