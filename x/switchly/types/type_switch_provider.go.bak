package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

func NewSWITCHProvider(addr cosmos.AccAddress) SWITCHProvider {
	return SWITCHProvider{
		SwitchAddress: addr,
		Units:         cosmos.ZeroUint(),
	}
}

func (rp SWITCHProvider) Key() string {
	return rp.SwitchAddress.String()
}
