package switchlyclient

import (
	"encoding/json"
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// GetNodeAccount retrieves node account for this address from switchly
func (b *switchlyBridge) GetNodeAccount(thorAddr string) (*types.NodeAccount, error) {
	path := fmt.Sprintf("%s/%s", NodeAccountEndpoint, thorAddr)
	body, _, err := b.getWithPath(path)
	if err != nil {
		return &types.NodeAccount{}, fmt.Errorf("failed to get node account: %w", err)
	}
	var na types.NodeAccount
	if err = json.Unmarshal(body, &na); err != nil {
		return &types.NodeAccount{}, fmt.Errorf("failed to unmarshal node account: %w", err)
	}
	return &na, nil
}

// GetNodeAccounts retrieves all node accounts from switchly
func (b *switchlyBridge) GetNodeAccounts() ([]*types.NodeAccount, error) {
	path := NodeAccountsEndpoint
	body, _, err := b.getWithPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get node account: %w", err)
	}
	var na []*types.NodeAccount
	if err = json.Unmarshal(body, &na); err != nil {
		return nil, fmt.Errorf("failed to unmarshal node accounts: %w", err)
	}
	return na, nil
}
