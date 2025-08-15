package swcyclaimlist

import (
	"encoding/json"
)

type SWCYClaimsJSON struct {
	Asset     string `json:"asset"`
	Address   string `json:"address"`
	SWCYClaim uint64 `json:"swcy_claim"`
}

var swcyClaims []SWCYClaimsJSON

func init() {
	if err := json.Unmarshal(SWCYClaimsListRaw, &swcyClaims); err != nil {
		panic(err)
	}
}

func GetSWCYClaimsList() []SWCYClaimsJSON {
	return swcyClaims
}
