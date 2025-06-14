package types

import (
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

func NewRUNEProvider(addr cosmos.AccAddress) RUNEProvider {
	return RUNEProvider{
		RuneAddress: addr,
		Units:       cosmos.ZeroUint(),
	}
}

func (rp RUNEProvider) Key() string {
	return rp.RuneAddress.String()
}
