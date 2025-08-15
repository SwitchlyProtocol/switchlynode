package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

var (
	_ sdk.Msg              = &MsgManageSWITCHName{}
	_ sdk.HasValidateBasic = &MsgManageSWITCHName{}
	_ sdk.LegacyMsg        = &MsgManageSWITCHName{}
)

// NewMsgManageSWITCHName create a new instance of MsgManageSWITCHName
func NewMsgManageSWITCHName(name string, chain common.Chain, addr common.Address, coin common.Coin, exp int64, asset common.Asset, owner, signer cosmos.AccAddress) *MsgManageSWITCHName {
	return &MsgManageSWITCHName{
		Name:              name,
		Chain:             chain,
		Address:           addr,
		Coin:              coin,
		ExpireBlockHeight: exp,
		PreferredAsset:    asset,
		Owner:             owner,
		Signer:            signer,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgManageSWITCHName) ValidateBasic() error {
	// validate n
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress(m.Signer.String())
	}
	if m.Chain.IsEmpty() {
		return cosmos.ErrUnknownRequest("chain can't be empty")
	}
	if m.Address.IsEmpty() {
		return cosmos.ErrUnknownRequest("address can't be empty")
	}
	if !m.Address.IsChain(m.Chain) {
		return cosmos.ErrUnknownRequest("address and chain must match")
	}
	if !m.Coin.IsSwitch() {
		return cosmos.ErrUnknownRequest("coin must be native switch")
	}
	return nil
}

// GetSigners defines whose signature is required
func (m *MsgManageSWITCHName) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
