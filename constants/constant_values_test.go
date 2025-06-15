package constants

import (
	"regexp"
	"testing"

	"github.com/blang/semver"
	. "gopkg.in/check.v1"
)

func TestPackage(t *testing.T) { TestingT(t) }

type ConstantsTestSuite struct{}

var _ = Suite(&ConstantsTestSuite{})

func (ConstantsTestSuite) TestConstantName_String(c *C) {
	constantNames := []ConstantName{
		EmissionCurve,
		BlocksPerYear,
		OutboundTransactionFee,
		PoolCycle,
		MinimumNodesForBFT,
		DesiredValidatorSet,
		ChurnInterval,
		LackOfObservationPenalty,
		SigningTransactionPeriod,
		DoubleSignMaxAge,
		MinimumBondInRune,
		ValidatorMaxRewardRatio,
	}
	for _, item := range constantNames {
		c.Assert(item.String(), Not(Equals), "NA")
	}
}

func (ConstantsTestSuite) TestGetConstantValues(c *C) {
	ver := semver.MustParse("0.0.9")
	c.Assert(GetConstantValues(ver), NotNil)
	c.Assert(GetConstantValues(SWVersion), NotNil)
}

func (ConstantsTestSuite) TestAllConstantName(c *C) {
	keyRegex := regexp.MustCompile(MimirKeyRegex).MatchString
	for i := 0; i < len(_ConstantName_index)-1; i++ {
		key := ConstantName(i)
		if !keyRegex(key.String()) {
			c.Errorf("key:%s can't be used to set mimir", key)
		}
	}
}

func (ConstantsTestSuite) TestStellarConstants(c *C) {
	// Test that Stellar constants are properly defined
	constantValues := NewConstantValue()

	// Test StellarMinAccountBalance
	minBalance := constantValues.GetInt64Value(StellarMinAccountBalance)
	c.Assert(minBalance, Equals, int64(10000000)) // 1 XLM in stroops

	// Test StellarBaseFee
	baseFee := constantValues.GetInt64Value(StellarBaseFee)
	c.Assert(baseFee, Equals, int64(100)) // 100 stroops

	// Test StellarMaxMemoLength
	maxMemoLength := constantValues.GetInt64Value(StellarMaxMemoLength)
	c.Assert(maxMemoLength, Equals, int64(28)) // 28 characters
}

func (ConstantsTestSuite) TestStellarConstantNames(c *C) {
	// Test that constant names are properly defined
	c.Assert(StellarMinAccountBalance.String(), Equals, "StellarMinAccountBalance")
	c.Assert(StellarBaseFee.String(), Equals, "StellarBaseFee")
	c.Assert(StellarMaxMemoLength.String(), Equals, "StellarMaxMemoLength")
}

func (ConstantsTestSuite) TestStellarConstantValidation(c *C) {
	// Test that all Stellar constants have valid values
	constantValues := NewConstantValue()

	// Ensure Stellar constants are not zero (which would indicate they weren't set)
	c.Assert(constantValues.GetInt64Value(StellarMinAccountBalance) > 0, Equals, true)
	c.Assert(constantValues.GetInt64Value(StellarBaseFee) > 0, Equals, true)
	c.Assert(constantValues.GetInt64Value(StellarMaxMemoLength) > 0, Equals, true)
}

func (ConstantsTestSuite) TestStellarConstantReasonableValues(c *C) {
	// Test that Stellar constants have reasonable values
	constantValues := NewConstantValue()

	// Min account balance should be 1 XLM (10,000,000 stroops)
	minBalance := constantValues.GetInt64Value(StellarMinAccountBalance)
	c.Assert(minBalance, Equals, int64(10000000))

	// Base fee should be reasonable (100 stroops is standard)
	baseFee := constantValues.GetInt64Value(StellarBaseFee)
	c.Assert(baseFee >= 100, Equals, true)
	c.Assert(baseFee <= 10000, Equals, true) // Should not be too high

	// Max memo length should match Stellar's limit
	maxMemoLength := constantValues.GetInt64Value(StellarMaxMemoLength)
	c.Assert(maxMemoLength, Equals, int64(28))
}
