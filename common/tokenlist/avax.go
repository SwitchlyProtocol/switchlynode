package tokenlist

import (
	"encoding/json"

	"github.com/switchlyprotocol/switchlynode/v1/common/tokenlist/avaxtokens"
)

var avaxTokenList EVMTokenList

func init() {
	if err := json.Unmarshal(avaxtokens.AVAXTokenListRaw, &avaxTokenList); err != nil {
		panic(err)
	}
}

func GetAVAXTokenList() EVMTokenList {
	return avaxTokenList
}
