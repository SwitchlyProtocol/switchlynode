package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

var (
	_ sdk.Msg              = &MsgSWCYUnstake{}
	_ sdk.HasValidateBasic = &MsgSWCYUnstake{}
	_ sdk.LegacyMsg        = &MsgSWCYUnstake{}
)

// NewMsgSWCYUnstake create new MsgSWCYUnstake message
func NewMsgSWCYUnstake(tx common.Tx, basisPoints math.Uint, signer sdk.AccAddress) *MsgSWCYUnstake {
	return &MsgSWCYUnstake{
		Signer:      signer,
		Tx:          tx,
		BasisPoints: basisPoints,
	}
}

// ValidateBasic runs stateless checks on the message
func (m *MsgSWCYUnstake) ValidateBasic() error {
	if m.Signer.Empty() {
		return cosmos.ErrInvalidAddress("signer must not be empty")
	}
	if m.BasisPoints.IsZero() || m.BasisPoints.GT(cosmos.NewUint(constants.MaxBasisPts)) {
		return cosmos.ErrUnknownRequest("invalid basis points")
	}
	if !m.Tx.FromAddress.IsChain(common.SWITCHLYChain) {
		return cosmos.ErrInvalidAddress("address should be switch address")
	}
	if !m.Tx.Coins.IsEmpty() {
		return cosmos.ErrInvalidCoins("coins must be empty (zero amount)")
	}

	return nil
}

// GetSigners defines whose signature is required
func (m *MsgSWCYUnstake) GetSigners() []cosmos.AccAddress {
	return []cosmos.AccAddress{m.Signer}
}
