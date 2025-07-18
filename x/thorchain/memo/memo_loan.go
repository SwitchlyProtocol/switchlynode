package thorchain

import (
	"strings"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

// "LOAN+:BTC.BTC:bc1YYYYYY:minBTC:affAddr:affPts:dexAgg:dexTarAddr:DexTargetLimit"

type LoanOpenMemo struct {
	MemoBase
	TargetAsset          common.Asset
	TargetAddress        common.Address
	MinOut               cosmos.Uint
	AffiliateAddress     common.Address
	AffiliateBasisPoints cosmos.Uint
	DexAggregator        string
	DexTargetAddress     string
	DexTargetLimit       cosmos.Uint
	AffiliateTHORName    *types.THORName
}

func (m LoanOpenMemo) GetTargetAsset() common.Asset          { return m.TargetAsset }
func (m LoanOpenMemo) GetTargetAddress() common.Address      { return m.TargetAddress }
func (m LoanOpenMemo) GetMinOut() cosmos.Uint                { return m.MinOut }
func (m LoanOpenMemo) GetAffiliateAddress() common.Address   { return m.AffiliateAddress }
func (m LoanOpenMemo) GetAffiliateBasisPoints() cosmos.Uint  { return m.AffiliateBasisPoints }
func (m LoanOpenMemo) GetDexAggregator() string              { return m.DexAggregator }
func (m LoanOpenMemo) GetDexTargetAddress() string           { return m.DexTargetAddress }
func (m LoanOpenMemo) GetAffiliateTHORName() *types.THORName { return m.AffiliateTHORName }

func (m LoanOpenMemo) String() string {
	return m.string(false)
}

func (m LoanOpenMemo) ShortString() string {
	return m.string(true)
}

func (m LoanOpenMemo) string(short bool) string {
	var assetString string
	if short && len(m.TargetAsset.ShortCode()) > 0 {
		assetString = m.TargetAsset.ShortCode()
	} else {
		assetString = m.TargetAsset.String()
	}

	args := []string{
		TxLoanOpen.String(),
		assetString,
		m.TargetAddress.String(),
		m.MinOut.String(),
		m.AffiliateAddress.String(),
		m.AffiliateBasisPoints.String(),
		m.DexAggregator,
		m.DexTargetAddress,
		m.DexTargetLimit.String(),
	}
	last := 3

	switch {
	case !m.DexTargetLimit.IsZero():
		last = 9
	case m.DexTargetAddress != "":
		last = 8
	case m.DexAggregator != "":
		last = 7
	case !m.AffiliateBasisPoints.IsZero():
		last = 6
	case !m.AffiliateAddress.IsEmpty():
		last = 5
	case !m.MinOut.IsZero():
		last = 4
	}

	return strings.Join(args[:last], ":")
}

func NewLoanOpenMemo(targetAsset common.Asset, targetAddr common.Address, minOut cosmos.Uint, affAddr common.Address, affPts cosmos.Uint, dexAgg, dexTargetAddr string, dexTargetLimit cosmos.Uint, tn types.THORName) LoanOpenMemo {
	return LoanOpenMemo{
		MemoBase:             MemoBase{TxType: TxLoanOpen},
		TargetAsset:          targetAsset,
		TargetAddress:        targetAddr,
		MinOut:               minOut,
		AffiliateAddress:     affAddr,
		AffiliateBasisPoints: affPts,
		DexAggregator:        dexAgg,
		DexTargetAddress:     dexTargetAddr,
		DexTargetLimit:       dexTargetLimit,
		AffiliateTHORName:    &tn,
	}
}

func (p *parser) ParseLoanOpenMemo() (LoanOpenMemo, error) {
	targetAsset := p.getAsset(1, true, common.EmptyAsset)
	targetAddress := p.getAddressWithKeeper(2, true, common.NoAddress, targetAsset.GetChain())
	minOut := p.getUintWithScientificNotation(3, false, 0)
	affAddr := p.getAddressWithKeeper(4, false, common.NoAddress, common.SWITCHLYChain)
	tn := p.getTHORName(5, false, types.NewTHORName("", 0, nil), -1)
	affPts := p.getUintWithMaxValue(6, false, 0, constants.MaxBasisPts)
	dexAgg := p.get(7)
	dexTargetAddr := p.get(8)
	dexTargetLimit := p.getUint(9, false, 0)
	return NewLoanOpenMemo(targetAsset, targetAddress, minOut, affAddr, affPts, dexAgg, dexTargetAddr, dexTargetLimit, tn), p.Error()
}

// "LOAN-:BTC.BTC:bc1YYYYYY:minOut"

type LoanRepaymentMemo struct {
	MemoBase
	Owner  common.Address
	MinOut cosmos.Uint
}

func (m LoanRepaymentMemo) String() string {
	args := []string{TxLoanRepayment.String(), m.Asset.String(), m.Owner.String()}
	if !m.MinOut.IsZero() {
		args = append(args, m.MinOut.String())
	}
	return strings.Join(args, ":")
}

func NewLoanRepaymentMemo(asset common.Asset, owner common.Address, minOut cosmos.Uint) LoanRepaymentMemo {
	return LoanRepaymentMemo{
		MemoBase: MemoBase{TxType: TxLoanRepayment, Asset: asset},
		Owner:    owner,
		MinOut:   minOut,
	}
}

func (p *parser) ParseLoanRepaymentMemo() (LoanRepaymentMemo, error) {
	asset := p.getAsset(1, true, common.EmptyAsset)
	owner := p.getAddressWithKeeper(2, true, common.NoAddress, asset.Chain)
	minOut := p.getUintWithScientificNotation(3, false, 0)
	return NewLoanRepaymentMemo(asset, owner, minOut), p.Error()
}
