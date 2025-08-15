package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

var (
	_ sdk.Msg              = &MsgSWCYClaim{}
	_ sdk.HasValidateBasic = &MsgSWCYClaim{}
	_ sdk.LegacyMsg        = &MsgSWCYClaim{}
)

// NewMsgSWCYClaim create new MsgSWCYClaim message
func NewMsgSWCYClaim(address, l1Address common.Address, signer sdk.AccAddress) *MsgSWCYClaim {
	return &MsgSWCYClaim{
		SwitchAddress: address,
		L1Address:     l1Address,
		Signer:        signer,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgSWCYClaim) ValidateBasic() error {
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress(m.Signer.String())
	}
	if m.SwitchAddress.IsEmpty() {
		return cosmos.ErrInvalidAddress("switch addresses cannot be empty")
	}
	if m.L1Address.IsEmpty() {
		return cosmos.ErrInvalidAddress("l1 addresses cannot be empty")
	}
	return nil
}

// GetSigners defines whose signature is required
func (m *MsgSWCYClaim) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
