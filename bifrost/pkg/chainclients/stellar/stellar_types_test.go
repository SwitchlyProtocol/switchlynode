package stellar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/thorchain/thornode/common"
)

func TestStellarAsset(t *testing.T) {
	assert.Equal(t, common.XLMAsset, stellarAsset)
} 