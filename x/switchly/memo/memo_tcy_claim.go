package switchly

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
)

type SWCYClaimMemo struct {
	MemoBase
	Address common.Address
}

func (m SWCYClaimMemo) GetAddress() common.Address {
	return m.Address
}

func NewSWCYClaimMemo(address common.Address) SWCYClaimMemo {
	return SWCYClaimMemo{
		MemoBase: MemoBase{TxType: TxSWCYClaim},
		Address:  address,
	}
}

func (p *parser) ParseSWCYClaimMemo() (SWCYClaimMemo, error) {
	address := p.getThorAddress(1, true, common.NoAddress)
	return NewSWCYClaimMemo(address), p.Error()
}
