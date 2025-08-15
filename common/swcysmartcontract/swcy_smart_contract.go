package swcysmartcontract

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
)

func IsSWCYSmartContractAddress(address common.Address) bool {
	return address.String() == SWCYSmartContractAddress
}

func GetSWCYSmartContractAddress() (common.Address, error) {
	return common.NewAddress(SWCYSmartContractAddress)
}
