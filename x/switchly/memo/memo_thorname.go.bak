package switchly

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type ManageSWITCHNameMemo struct {
	MemoBase
	Name           string
	Chain          common.Chain
	Address        common.Address
	PreferredAsset common.Asset
	Expire         int64
	Owner          cosmos.AccAddress
}

func (m ManageSWITCHNameMemo) GetName() string            { return m.Name }
func (m ManageSWITCHNameMemo) GetChain() common.Chain     { return m.Chain }
func (m ManageSWITCHNameMemo) GetAddress() common.Address { return m.Address }
func (m ManageSWITCHNameMemo) GetBlockExpire() int64      { return m.Expire }

func NewManageSWITCHNameMemo(name string, chain common.Chain, addr common.Address, expire int64, asset common.Asset, owner cosmos.AccAddress) ManageSWITCHNameMemo {
	return ManageSWITCHNameMemo{
		MemoBase:       MemoBase{TxType: TxSWITCHName},
		Name:           name,
		Chain:          chain,
		Address:        addr,
		PreferredAsset: asset,
		Expire:         expire,
		Owner:          owner,
	}
}

func (p *parser) ParseManageSWITCHNameMemo() (ManageSWITCHNameMemo, error) {
	chain := p.getChain(2, true, common.EmptyChain)
	addr := p.getAddress(3, true, common.NoAddress)
	owner := p.getAccAddress(4, false, nil)
	preferredAsset := p.getAsset(5, false, common.EmptyAsset)
	expire := p.getInt64(6, false, 0)
	return NewManageSWITCHNameMemo(p.get(1), chain, addr, expire, preferredAsset, owner), p.Error()
}
