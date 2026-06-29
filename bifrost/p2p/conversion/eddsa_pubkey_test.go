package conversion

import (
	"encoding/hex"
	"testing"

	eddsakeygen "github.com/binance-chain/tss-lib/eddsa/keygen"
	btss "github.com/binance-chain/tss-lib/tss"
	"github.com/decred/dcrd/dcrec/edwards/v2"

	"github.com/switchlyprotocol/switchlynode/v3/common"
)

// TestGetTssPubKeyEdDSA checks the EdDSA group-pubkey encoder produces the canonical 32-byte ed25519
// key (matching decred's serialize) and that it maps to a valid Stellar address — the seam the EdDSA
// keygen path (Layer 2) will use to report the group key.
func TestGetTssPubKeyEdDSA(t *testing.T) {
	prev := btss.EC()
	btss.SetCurve(edwards.Edwards())
	defer btss.SetCurve(prev)

	keys, _, err := eddsakeygen.LoadKeygenTestFixtures(1)
	if err != nil {
		t.Fatalf("load eddsa fixtures: %v", err)
	}
	point := keys[0].EDDSAPub

	got, err := GetTssPubKeyEdDSA(point)
	if err != nil {
		t.Fatalf("GetTssPubKeyEdDSA: %v", err)
	}
	if len(got) != 64 { // 32-byte ed25519 key -> 64 hex chars
		t.Fatalf("expected 64 hex chars, got %d (%s)", len(got), got)
	}
	want := hex.EncodeToString(edwards.PublicKey{Curve: edwards.Edwards(), X: point.X(), Y: point.Y()}.Serialize())
	if got != want {
		t.Fatalf("encoding mismatch: got %s want %s", got, want)
	}

	raw, err := hex.DecodeString(got)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	addr, err := common.Ed25519PubKeyToStellarAddress(raw)
	if err != nil {
		t.Fatalf("stellar address: %v", err)
	}
	if !common.IsValidXLMAddress(addr.String()) {
		t.Fatalf("invalid stellar address %s", addr.String())
	}
	t.Logf("eddsa group pubkey %s -> stellar %s", got, addr.String())
}
