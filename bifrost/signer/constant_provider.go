package signer

import (
	"fmt"
	"sync"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

// ConstantsProvider which will query switchly to get the constants value per request
// it will also cache the constant values internally
type ConstantsProvider struct {
	requestHeight int64 // the block height last request to switchly to retrieve constant values
	bridge        switchlyclient.SwitchlyBridge
	constantsLock *sync.Mutex
	constants     map[string]int64 // the constant values get from switchly and cached in memory
}

// NewConstantsProvider create a new instance of ConstantsProvider
func NewConstantsProvider(bridge switchlyclient.SwitchlyBridge) *ConstantsProvider {
	return &ConstantsProvider{
		constants:     make(map[string]int64),
		requestHeight: 0,
		bridge:        bridge,
		constantsLock: &sync.Mutex{},
	}
}

// GetInt64Value get the constant value that match the given key
func (cp *ConstantsProvider) GetInt64Value(switchlyBlockHeight int64, key constants.ConstantName) (int64, error) {
	if err := cp.EnsureConstants(switchlyBlockHeight); err != nil {
		return 0, fmt.Errorf("fail to get constants from switchly: %w", err)
	}
	cp.constantsLock.Lock()
	defer cp.constantsLock.Unlock()
	return cp.constants[key.String()], nil
}

func (cp *ConstantsProvider) EnsureConstants(switchlyBlockHeight int64) error {
	if cp.requestHeight == 0 {
		return cp.getConstantsFromSwitchly(switchlyBlockHeight)
	}
	cp.constantsLock.Lock()
	churnInterval := cp.constants[constants.ChurnInterval.String()]
	cp.constantsLock.Unlock()
	// Switchly will have new version and constants only when new node get rotated in , and the new version get consensus
	if switchlyBlockHeight-cp.requestHeight < churnInterval {
		return nil
	}
	return cp.getConstantsFromSwitchly(switchlyBlockHeight)
}

func (cp *ConstantsProvider) getConstantsFromSwitchly(height int64) error {
	constants, err := cp.bridge.GetConstants()
	if err != nil {
		return fmt.Errorf("fail to get constants: %w", err)
	}
	cp.constantsLock.Lock()
	defer cp.constantsLock.Unlock()
	cp.constants = constants
	cp.requestHeight = height
	return nil
}
