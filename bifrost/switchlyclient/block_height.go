package switchlyclient

import (
	"encoding/json"
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	openapi "github.com/switchlyprotocol/switchlynode/v3/openapi/gen"
)

// GetLastObservedInHeight returns the lastobservedin value for the chain past in
func (b *switchlyBridge) GetLastObservedInHeight(chain common.Chain) (int64, error) {
	lastblock, err := b.getLastBlock(chain)
	if err != nil {
		return 0, fmt.Errorf("failed to GetLastObservedInHeight: %w", err)
	}
	for _, item := range lastblock {
		if item.Chain == chain.String() {
			return item.LastObservedIn, nil
		}
	}
	return 0, fmt.Errorf("fail to GetLastObservedInHeight,chain(%s)", chain)
}

// GetLastSignedOutHeight returns the lastsignedout value for switchly
func (b *switchlyBridge) GetLastSignedOutHeight(chain common.Chain) (int64, error) {
	lastblock, err := b.getLastBlock(chain)
	if err != nil {
		return 0, fmt.Errorf("failed to GetLastSignedOutHeight: %w", err)
	}
	for _, item := range lastblock {
		if item.Chain == chain.String() {
			return item.LastSignedOut, nil
		}
	}
	return 0, fmt.Errorf("fail to GetLastSignedOutHeight,chain(%s)", chain)
}

// GetBlockHeight returns the current height for switchly blocks
func (b *switchlyBridge) GetBlockHeight() (int64, error) {
	latestBlocks, err := b.getLastBlock(common.EmptyChain)
	if err != nil {
		return 0, fmt.Errorf("failed to GetSwitchlyHeight: %w", err)
	}
	for _, item := range latestBlocks {
		return item.Switchly, nil
	}
	return 0, fmt.Errorf("failed to GetSwitchlyHeight")
}

// getLastBlock calls the /lastblock/{chain} endpoint and Unmarshal's into the QueryResLastBlockHeights type
func (b *switchlyBridge) getLastBlock(chain common.Chain) ([]openapi.LastBlock, error) {
	path := LastBlockEndpoint
	if !chain.IsEmpty() {
		path = fmt.Sprintf("%s/%s", path, chain.String())
	}
	buf, _, err := b.getWithPath(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get lastblock: %w", err)
	}
	var lastBlock []openapi.LastBlock
	if err = json.Unmarshal(buf, &lastBlock); err != nil {
		return nil, fmt.Errorf("failed to unmarshal last block: %w", err)
	}
	return lastBlock, nil
}
