//go:build mocknet
// +build mocknet

package stellar

// placeholderStellarSigningAllowed permits the INSECURE placeholder Stellar key derivation
// (see DeriveStellarkeyFromVaultPubKey). It is true only for mocknet builds.
//
// The placeholder derives the vault's ed25519 signing key from the PUBLIC secp256k1 vault pubkey, so
// the private key is recoverable by anyone from public data and must NEVER sign with real funds. Real
// signing requires EdDSA threshold signing — see docs/architecture/stellar-eddsa-tss.md.
const placeholderStellarSigningAllowed = true
