package common

// Algo identifies the threshold-signing scheme/curve used for a keygen or keysign ceremony.
//
// Every existing vault and chain uses ECDSA (secp256k1). EdDSA (ed25519) is required for Stellar,
// whose classic accounts use ed25519 signatures — see docs/architecture/stellar-eddsa-tss.md. This
// type is the seam through which keygen.Request / keysign.Request will select the scheme so that the
// ECDSA path stays byte-for-byte unchanged while the EdDSA path is added alongside.
//
// NOTE: the EdDSA code paths (tss-lib eddsa keygen/signing, party construction, verification, blame
// rounds) are not implemented yet; this only defines the selector.
type Algo string

const (
	// ECDSA is secp256k1 threshold signing (the default for all existing chains).
	ECDSA Algo = "ecdsa"
	// EdDSA is ed25519 threshold signing (required for Stellar).
	EdDSA Algo = "eddsa"
)

// NormalizeAlgo returns a valid Algo, defaulting an empty or unrecognized value to ECDSA. Requests
// from older nodes omit the algorithm field, so an empty value must keep the current secp256k1
// behavior for wire/back-compat.
func NormalizeAlgo(a Algo) Algo {
	if a == EdDSA {
		return EdDSA
	}
	return ECDSA
}
