package common

import (
	"crypto/elliptic"
	"testing"

	btss "github.com/binance-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/decred/dcrd/dcrec/edwards/v2"
)

// TestWithCurveForAlgo verifies the global tss-lib curve is set to the right curve inside the callback
// and restored afterwards. (Plain testing.T alongside the package's gopkg.in/check.v1 suite.)
func TestWithCurveForAlgo(t *testing.T) {
	prev := btss.EC()

	// EdDSA -> Edwards inside fn.
	var inside elliptic.Curve
	if err := WithCurveForAlgo(EdDSA, func() error { inside = btss.EC(); return nil }); err != nil {
		t.Fatalf("WithCurveForAlgo(EdDSA) error: %v", err)
	}
	if inside.Params().N.Cmp(edwards.Edwards().Params().N) != 0 {
		t.Fatalf("expected Edwards curve inside EdDSA callback")
	}
	if btss.EC().Params().N.Cmp(prev.Params().N) != 0 {
		t.Fatalf("curve not restored after EdDSA ceremony")
	}

	// ECDSA and empty (legacy) -> secp256k1 inside fn.
	for _, a := range []Algo{ECDSA, Algo("")} {
		var c elliptic.Curve
		if err := WithCurveForAlgo(a, func() error { c = btss.EC(); return nil }); err != nil {
			t.Fatalf("WithCurveForAlgo(%q) error: %v", a, err)
		}
		if c.Params().N.Cmp(s256k1.S256().Params().N) != 0 {
			t.Fatalf("expected secp256k1 inside callback for algo %q", a)
		}
	}
}
