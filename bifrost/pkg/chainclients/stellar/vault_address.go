package stellar

import (
	"fmt"
	"sync"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/common"
)

// vaultStellarAddrCache memoises the secp256k1-vault-identity -> Stellar-account-address mapping.
// The mapping is immutable for the life of a vault (the ed25519 group key is fixed at keygen), so a
// successful resolution can be cached forever; this keeps the per-tx scanner path from issuing a
// GetVault for every transaction.
var vaultStellarAddrCache sync.Map // vault secp pubkey string -> stellar address string

// vaultStellarAddress resolves a vault's Stellar account address from its EdDSA (ed25519) group key
// — the real account that holds XLM, receives inbounds, and authorizes the router — falling back to
// the secp256k1 placeholder for legacy/pre-EdDSA vaults.
//
// A vault is identified network-wide by its secp256k1 PubKey, but on Stellar that key only derives
// the (unspendable) placeholder address. The real account is derived from the vault's ed25519 key,
// which lives on the Vault record, not the bare pubkey. So every XLM address resolution for a vault
// — the inbound watch address, the outbound source account, the sequence lookup, and the router
// `vault` argument — must go through here rather than `vaultPubKey.GetAddress(StellarChain)`.
func vaultStellarAddress(bridge switchlyclient.SwitchlyBridge, vaultPubKey common.PubKey) (common.Address, error) {
	if v, ok := vaultStellarAddrCache.Load(vaultPubKey.String()); ok {
		return common.Address(v.(string)), nil
	}
	vault, err := bridge.GetVault(vaultPubKey.String())
	if err != nil {
		return common.NoAddress, fmt.Errorf("fail to get vault %s: %w", vaultPubKey, err)
	}
	addr, err := vault.PubKeyForChain(common.StellarChain).GetAddress(common.StellarChain)
	if err != nil {
		return common.NoAddress, fmt.Errorf("fail to derive stellar address for vault %s: %w", vaultPubKey, err)
	}
	if !addr.IsEmpty() {
		vaultStellarAddrCache.Store(vaultPubKey.String(), addr.String())
	}
	return addr, nil
}
