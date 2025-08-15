package switchly

import (
	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type SwitchMemo struct {
	MemoBase
	Address cosmos.AccAddress
}

func (m SwitchMemo) GetAccAddress() cosmos.AccAddress { return m.Address }

func NewSwitchMemo(addr cosmos.AccAddress) SwitchMemo {
	return SwitchMemo{
		MemoBase: MemoBase{TxType: TxSwitch},
		Address:  addr,
	}
}

func (p *parser) ParseSwitch() (SwitchMemo, error) {
	addr := p.getAccAddress(1, true, nil)
	return NewSwitchMemo(addr), p.Error()
}
