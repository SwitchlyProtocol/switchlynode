package stellar

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	switchlytypes "github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// TestVaultStellarAddressEd25519 asserts the resolver returns the vault's real, ed25519-derived
// Stellar account — not the secp256k1 placeholder that the vault identity alone yields.
func (s *StellarClientTestSuite) TestVaultStellarAddressEd25519(c *C) {
	secp := newTestVaultPubKey(c) // random per call, so no global-cache collision across tests

	edRaw := make([]byte, 32)
	for i := range edRaw {
		edRaw[i] = byte(i + 7)
	}
	edPub, err := common.NewPubKeyFromEd25519(edRaw)
	c.Assert(err, IsNil)

	bridge := &MockSwitchlyBridge{VaultToReturn: switchlytypes.Vault{PubKey: secp, Ed25519PubKey: edPub}}
	addr, err := vaultStellarAddress(bridge, secp)
	c.Assert(err, IsNil)

	want, err := common.Ed25519PubKeyToStellarAddress(edRaw)
	c.Assert(err, IsNil)
	c.Assert(addr.String(), Equals, want.String())

	// must differ from the secp256k1 placeholder the vault key alone would derive
	placeholder, err := secp.GetAddress(common.StellarChain)
	c.Assert(err, IsNil)
	c.Assert(addr.String(), Not(Equals), placeholder.String())
}

// TestVaultStellarAddressLegacyFallback asserts a vault with no ed25519 key (legacy/pre-EdDSA) falls
// back to the secp256k1 placeholder address, preserving the previous behaviour.
func (s *StellarClientTestSuite) TestVaultStellarAddressLegacyFallback(c *C) {
	secp := newTestVaultPubKey(c)
	bridge := &MockSwitchlyBridge{VaultToReturn: switchlytypes.Vault{PubKey: secp}}
	addr, err := vaultStellarAddress(bridge, secp)
	c.Assert(err, IsNil)

	placeholder, err := secp.GetAddress(common.StellarChain)
	c.Assert(err, IsNil)
	c.Assert(addr.String(), Equals, placeholder.String())
}
