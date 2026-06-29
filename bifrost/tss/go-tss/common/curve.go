package common

import (
	"sync"

	btss "github.com/binance-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/decred/dcrd/dcrec/edwards/v2"
)

// The pinned tss-lib selects the signing curve via a process global (tss.SetCurve / tss.EC()) rather
// than per *Parameters. So an ECDSA (secp256k1) ceremony and an EdDSA (Edwards) ceremony must not run
// concurrently in one process — one would read a curve left set by the other and produce invalid
// points (e.g. an active node's ECDSA migration keysign racing the EdDSA churn keygen). The proper fix
// is a tss-lib that takes the curve per *Parameters; until then we gate on the global curve. See
// docs/architecture/stellar-eddsa-tss.md §9.
//
// The gate must NOT be a plain mutex held for the whole ceremony: the in-process multi-party test
// (tss_4nodes_test.go) runs several interdependent parties of the SAME ceremony in one process, and a
// plain mutex would let the first party hold it while it waits for messages from the others — who are
// blocked acquiring the same mutex — a deadlock. Instead this is an algo-gate: any number of ceremonies
// of the SAME algo may hold it concurrently (they share one curve), but switching to the other algo
// waits until all in-flight ceremonies of the current algo have finished.
var (
	curveGateMu    sync.Mutex
	curveGateCond  = sync.NewCond(&curveGateMu)
	curveGateAlgo  Algo
	curveGateCount int
)

// WithCurveForAlgo runs fn with tss-lib's global curve set for algo, serialized against ceremonies of
// the other algo (concurrent same-algo ceremonies are allowed). fn MUST run the ceremony to completion
// before returning — tss-lib parties read the global curve throughout their rounds.
func WithCurveForAlgo(algo Algo, fn func() error) error {
	algo = NormalizeAlgo(algo)

	curveGateMu.Lock()
	// wait until no ceremony of a different algo is in flight
	for curveGateCount > 0 && curveGateAlgo != algo {
		curveGateCond.Wait()
	}
	if curveGateCount == 0 {
		// first concurrent ceremony of this algo — set the global curve for the batch
		switch algo {
		case EdDSA:
			btss.SetCurve(edwards.Edwards())
		default:
			btss.SetCurve(s256k1.S256())
		}
		curveGateAlgo = algo
	}
	curveGateCount++
	curveGateMu.Unlock()

	defer func() {
		curveGateMu.Lock()
		curveGateCount--
		if curveGateCount == 0 {
			// idle again — restore the default (secp256k1) curve so any code reading tss.EC() outside a
			// ceremony sees the historical default, then let a waiting other-algo ceremony proceed.
			btss.SetCurve(s256k1.S256())
			curveGateCond.Broadcast()
		}
		curveGateMu.Unlock()
	}()

	return fn()
}
