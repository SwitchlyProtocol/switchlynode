//go:build !mocknet
// +build !mocknet

package stellar

// placeholderStellarSigningAllowed permits the INSECURE placeholder Stellar key derivation
// (see DeriveStellarkeyFromVaultPubKey). It is false for all non-mocknet builds (stagenet/mainnet),
// so the placeholder can never sign with real funds.
//
// Real signing requires EdDSA threshold signing — see docs/architecture/stellar-eddsa-tss.md. Until
// that lands, Stellar outbound signing is unavailable on these builds (DeriveStellarkeyFromVaultPubKey
// returns an error).
const placeholderStellarSigningAllowed = false
