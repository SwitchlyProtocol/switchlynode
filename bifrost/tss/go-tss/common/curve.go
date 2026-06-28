package common

import (
	"sync"

	btss "github.com/binance-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/decred/dcrd/dcrec/edwards/v2"
)

// curveMu serializes tss-lib's process-global signing curve (tss.SetCurve / tss.EC()).
//
// The pinned tss-lib selects the curve via a process global rather than per *Parameters, so ECDSA
// (secp256k1) and EdDSA (Edwards) ceremonies cannot run concurrently — a concurrent ECDSA ceremony
// would read a curve left on Edwards by an in-flight EdDSA ceremony, and vice versa. WithCurveForAlgo
// holds this lock for the entire ceremony.
//
// IMPORTANT: because the curve is global, this serializes ALL ceremonies routed through it (even two
// ECDSA ones). That is a correctness-over-concurrency trade-off forced by the library; the proper fix
// is a tss-lib that accepts the curve per *Parameters. See docs/architecture/stellar-eddsa-tss.md.
var curveMu sync.Mutex

// WithCurveForAlgo sets tss-lib's global curve for algo, runs fn, then restores the previous curve.
//
// fn MUST run the ceremony to completion before returning: tss-lib parties read the global curve
// throughout their rounds, so the curve must not change until keygen/keysign has fully finished.
func WithCurveForAlgo(algo Algo, fn func() error) error {
	curveMu.Lock()
	defer curveMu.Unlock()

	prev := btss.EC()
	defer btss.SetCurve(prev)

	switch NormalizeAlgo(algo) {
	case EdDSA:
		btss.SetCurve(edwards.Edwards())
	default:
		btss.SetCurve(s256k1.S256())
	}
	return fn()
}
