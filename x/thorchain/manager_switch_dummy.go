package thorchain

import (
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type SwitchMgrDummy struct{}

func NewDummySwitchManager() *SwitchMgrDummy {
	return &SwitchMgrDummy{}
}

func (s *SwitchMgrDummy) IsSwitch(
	ctx cosmos.Context,
	asset common.Asset,
) bool {
	return false
}

func (s *SwitchMgrDummy) Switch(
	ctx cosmos.Context,
	asset common.Asset,
	amount cosmos.Uint,
	owner cosmos.AccAddress,
	assetAddr common.Address,
	txID common.TxID,
) (common.Address, error) {
	return common.NoAddress, nil
}
