package switchly

import (
	"strings"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// "pool+"

type SwitchPoolDepositMemo struct {
	MemoBase
}

func (m SwitchPoolDepositMemo) String() string {
	return m.string(false)
}

func (m SwitchPoolDepositMemo) ShortString() string {
	return m.string(true)
}

func (m SwitchPoolDepositMemo) string(short bool) string {
	return "pool+"
}

func NewSwitchPoolDepositMemo() SwitchPoolDepositMemo {
	return SwitchPoolDepositMemo{
		MemoBase: MemoBase{TxType: TxSwitchPoolDeposit},
	}
}

func (p *parser) ParseSwitchPoolDepositMemo() (SwitchPoolDepositMemo, error) {
	return NewSwitchPoolDepositMemo(), nil
}

// "pool-:<basis-points>:<affiliate>:<affiliate-basis-points>"

type SwitchPoolWithdrawMemo struct {
	MemoBase
	BasisPoints          cosmos.Uint
	AffiliateAddress     common.Address
	AffiliateBasisPoints cosmos.Uint
	AffiliateSWITCHName  *types.SWITCHName
}

func (m SwitchPoolWithdrawMemo) GetBasisPts() cosmos.Uint             { return m.BasisPoints }
func (m SwitchPoolWithdrawMemo) GetAffiliateAddress() common.Address  { return m.AffiliateAddress }
func (m SwitchPoolWithdrawMemo) GetAffiliateBasisPoints() cosmos.Uint { return m.AffiliateBasisPoints }
func (m SwitchPoolWithdrawMemo) GetAffiliateSWITCHName() *types.SWITCHName {
	return m.AffiliateSWITCHName
}

func (m SwitchPoolWithdrawMemo) String() string {
	args := []string{TxSwitchPoolWithdraw.String(), m.BasisPoints.String(), m.AffiliateAddress.String(), m.AffiliateBasisPoints.String()}
	return strings.Join(args, ":")
}

func NewSwitchPoolWithdrawMemo(basisPoints cosmos.Uint, affAddr common.Address, affBps cosmos.Uint, tn types.SWITCHName) SwitchPoolWithdrawMemo {
	mem := SwitchPoolWithdrawMemo{
		MemoBase:             MemoBase{TxType: TxSwitchPoolWithdraw},
		BasisPoints:          basisPoints,
		AffiliateAddress:     affAddr,
		AffiliateBasisPoints: affBps,
	}
	if !tn.Owner.Empty() {
		mem.AffiliateSWITCHName = &tn
	}
	return mem
}

func (p *parser) ParseSwitchPoolWithdrawMemo() (SwitchPoolWithdrawMemo, error) {
	basisPoints := p.getUint(1, true, cosmos.ZeroInt().Uint64())
	affiliateAddress := p.getAddressWithKeeper(2, false, common.NoAddress, common.SWITCHLYChain)
	tn := p.getSWITCHName(2, false, types.NewSWITCHName("", 0, nil), -1)
	affiliateBasisPoints := p.getUintWithMaxValue(3, false, 0, constants.MaxBasisPts)
	return NewSwitchPoolWithdrawMemo(basisPoints, affiliateAddress, affiliateBasisPoints, tn), p.Error()
}
