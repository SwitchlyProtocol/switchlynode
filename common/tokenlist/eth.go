package tokenlist

import (
	"encoding/json"

	"github.com/switchlyprotocol/switchlynode/v1/common/tokenlist/ethtokens"
)

var ethTokenList EVMTokenList

func init() {
	if err := json.Unmarshal(ethtokens.ETHTokenListRaw, &ethTokenList); err != nil {
		panic(err)
	}
}

func GetETHTokenList() EVMTokenList {
	return ethTokenList
}
