package eddsacompat

import (
	"crypto/ed25519"
	"crypto/sha256"
	"math/big"
	"sync/atomic"
	"testing"

	ecdsakeygen "github.com/binance-chain/tss-lib/ecdsa/keygen"
	eddsakeygen "github.com/binance-chain/tss-lib/eddsa/keygen"
	eddsasigning "github.com/binance-chain/tss-lib/eddsa/signing"
	"github.com/binance-chain/tss-lib/test"
	btss "github.com/binance-chain/tss-lib/tss"
	"github.com/decred/dcrd/dcrec/edwards/v2"
	"github.com/stretchr/testify/require"

	"github.com/switchlyprotocol/switchlynode/v3/common"
)

// Linking BOTH ecdsa/keygen and eddsa/keygen from the tss-lib fork in one binary must NOT panic at
// init. Before the fork-patch (giving the eddsa protos a distinct `eddsa` proto package) this
// collided — both registered protobuf messages named KGRound2Message2 under the same (empty) package.
// Referencing an ecdsa type here forces its proto package to link alongside eddsa's.
var _ = ecdsakeygen.LocalPartySaveData{}

// TestThresholdEdDSAVerifiesUnderStellarEd25519 runs a real in-process t-of-n threshold EdDSA signing
// ceremony (using tss-lib's eddsa keygen fixtures) and asserts the resulting signature + group public
// key are accepted by the standard ed25519 verifier that Stellar uses, and that the group key maps to
// a valid Stellar address. This is the executable evidence behind the EdDSA plan.
func TestThresholdEdDSAVerifiesUnderStellarEd25519(t *testing.T) {
	// tss-lib's curve is a process global; EdDSA needs Edwards. Restore afterwards.
	prevCurve := btss.EC()
	btss.SetCurve(edwards.Edwards())
	defer btss.SetCurve(prevCurve)

	threshold := test.TestThreshold // t; signing needs t+1 participants

	// Load precomputed keygen fixtures (group key already established across n parties).
	keys, signPIDs, err := eddsakeygen.LoadKeygenTestFixturesRandomSet(threshold+1, test.TestParticipants)
	require.NoError(t, err, "load eddsa keygen fixtures")
	require.Len(t, keys, threshold+1)

	// The "transaction hash" to sign (Stellar signs a 32-byte hash).
	hash := sha256.Sum256([]byte("switchly:stellar:eddsa:threshold:reference"))
	msg := new(big.Int).SetBytes(hash[:])

	p2pCtx := btss.NewPeerContext(signPIDs)
	outCh := make(chan btss.Message, len(signPIDs))
	endCh := make(chan *eddsasigning.SignatureData, len(signPIDs))
	errCh := make(chan *btss.Error, len(signPIDs))

	parties := make([]*eddsasigning.LocalParty, 0, len(signPIDs))
	for i := range signPIDs {
		params := btss.NewParameters(p2pCtx, signPIDs[i], len(signPIDs), threshold)
		p := eddsasigning.NewLocalParty(msg, params, keys[i], outCh, endCh).(*eddsasigning.LocalParty)
		parties = append(parties, p)
		go func(p *eddsasigning.LocalParty) {
			if startErr := p.Start(); startErr != nil {
				errCh <- startErr
			}
		}(p)
	}

	var sig *eddsasigning.SignatureData
	var ended int32
	updater := test.SharedPartyUpdater

loop:
	for {
		select {
		case tErr := <-errCh:
			require.FailNow(t, "tss signing error", tErr.Error())
		case m := <-outCh:
			dest := m.GetTo()
			if dest == nil { // broadcast
				for _, p := range parties {
					if p.PartyID().Index == m.GetFrom().Index {
						continue
					}
					go updater(p, m, errCh)
				}
			} else { // point-to-point
				require.NotEqual(t, dest[0].Index, m.GetFrom().Index, "party tried to message itself")
				go updater(parties[dest[0].Index], m, errCh)
			}
		case s := <-endCh:
			sig = s
			if atomic.AddInt32(&ended, 1) == int32(len(signPIDs)) {
				break loop
			}
		}
	}

	require.NotNil(t, sig, "signing must produce a signature")

	// The group ed25519 public key, encoded as the standard 32-byte compressed point.
	pk := edwards.PublicKey{Curve: btss.EC(), X: keys[0].EDDSAPub.X(), Y: keys[0].EDDSAPub.Y()}
	pub32 := pk.Serialize()
	sig64 := sig.Signature.GetSignature()
	signed := msg.Bytes() // exactly the bytes tss-lib signed

	// 1) Sanity: verifies under tss-lib's own (decred) verifier.
	parsed, err := edwards.ParseSignature(sig64)
	require.NoError(t, err, "parse signature")
	require.True(t, edwards.Verify(&pk, signed, parsed.R, parsed.S), "decred edwards verify")

	// 2) THE interop check: verifies under the STANDARD ed25519 verifier (what Stellar uses).
	require.Len(t, pub32, ed25519.PublicKeySize)
	require.Len(t, sig64, ed25519.SignatureSize)
	require.True(t, ed25519.Verify(ed25519.PublicKey(pub32), signed, sig64),
		"threshold EdDSA signature must verify under standard crypto/ed25519 (Stellar verifier)")

	// 3) The group public key maps to a valid Stellar account address via the shared seam.
	addr, err := common.Ed25519PubKeyToStellarAddress(pub32)
	require.NoError(t, err, "derive Stellar address from group pubkey")
	require.True(t, common.IsValidXLMAddress(addr.String()), "derived Stellar address must be valid")

	t.Logf("threshold EdDSA group key -> Stellar address %s (verified under crypto/ed25519)", addr.String())
}
