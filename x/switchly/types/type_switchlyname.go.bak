package types

import (
	"errors"
	"strings"

	b64 "encoding/base64"

	"github.com/switchlyprotocol/switchlynode/v3/common"
)

// NewSWITCHName create a new instance of SWITCHName
func NewSWITCHName(name string, exp int64, aliases []SWITCHNameAlias) SWITCHName {
	return SWITCHName{
		Name:              name,
		ExpireBlockHeight: exp,
		Aliases:           aliases,
	}
}

// Valid - check whether SWITCHName struct represent valid information
func (m *SWITCHName) Valid() error {
	if len(m.Name) == 0 {
		return errors.New("name can't be empty")
	}
	if len(m.Aliases) == 0 {
		return errors.New("aliases can't be empty")
	}
	for _, a := range m.Aliases {
		if a.Chain.IsEmpty() {
			return errors.New("chain can't be empty")
		}
		if a.Address.IsEmpty() {
			return errors.New("address cannot be empty")
		}
	}
	return nil
}

func (m *SWITCHName) GetAlias(chain common.Chain) common.Address {
	for _, a := range m.Aliases {
		if a.Chain.Equals(chain) {
			return a.Address
		}
	}
	return common.NoAddress
}

func (m *SWITCHName) SetAlias(chain common.Chain, addr common.Address) {
	for i, a := range m.Aliases {
		if a.Chain.Equals(chain) {
			m.Aliases[i].Address = addr
			return
		}
	}
	m.Aliases = append(m.Aliases, SWITCHNameAlias{Chain: chain, Address: addr})
}

func (m *SWITCHName) Key() string {
	// key is Base64 endoded
	return b64.StdEncoding.EncodeToString([]byte(strings.ToLower(m.Name)))
}

// CanReceiveAffiliateFee - returns true if the SWITCHName can receive an affiliate fee.
// Conditions: - Must have an owner
//   - If no preferred asset, must have an alias for SwitchlyProtocol (since fee will be sent in SWITCH)
//   - If preferred asset, can receive affiliate fee (since fee is collected in AC module)
func (m *SWITCHName) CanReceiveAffiliateFee() bool {
	if m.Owner.Empty() {
		return false
	}
	if m.PreferredAsset.IsEmpty() {
		// If no preferred asset set, must have a switch alias to receive switch fees
		return !m.GetAlias(common.SWITCHLYChain).IsEmpty()
	}

	// If preferred asset set, must have an alias for the preferred asset chain
	return !m.GetAlias(m.PreferredAsset.GetChain()).IsEmpty()
}
