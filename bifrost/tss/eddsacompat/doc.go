// Package eddsacompat is an executable compatibility reference for the Stellar EdDSA threshold-signing
//
// It proves, against the tss-lib already vendored by this repo, that:
//  1. tss-lib's eddsa/{keygen,signing} produces a real threshold ed25519 signature, and
//  2. that signature verifies under the STANDARD crypto/ed25519 verifier (the one Stellar uses) — not
//     only under tss-lib's own decred/edwards verifier, and
//  3. the EdDSA group public key maps to a valid Stellar account address via
//     common.Ed25519PubKeyToStellarAddress.
//
// This de-risks the biggest unknown in the plan (Stellar-verifiability of threshold EdDSA signatures)
// and is the wiring template for go-tss's EdDSA keygen/keysign path. It is intentionally test-only:
// no production code lives here; the real wiring belongs in bifrost/tss/go-tss.
//
// NOTE: tss-lib selects the curve via a process-global (tss.SetCurve); EdDSA requires the Edwards
// curve, so the EdDSA path must run with the global set to Edwards and serialized against the ECDSA
// (secp256k1) path. This package isolates that global to its own test binary.
package eddsacompat
