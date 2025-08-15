package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

func NewSecuredAsset(asset common.Asset) SecuredAsset {
	return SecuredAsset{
		Asset: asset,
		Depth: cosmos.ZeroUint(),
	}
}

func (tu SecuredAsset) Key() string {
	return tu.Asset.String()
}
