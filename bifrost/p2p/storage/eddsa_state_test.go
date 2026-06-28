package storage

import (
	"encoding/json"
	"testing"
)

// TestKeygenLocalStateEdDSARoundTrip checks that an EdDSA keyshare (algo + opaque eddsa save data)
// survives a JSON round-trip, and that legacy state (no algo / no eddsa data) decodes as ECDSA by
// default. The eddsa save data is stored as opaque JSON here (see KeygenLocalState) so this package
// never imports tss-lib's eddsa protos.
func TestKeygenLocalStateEdDSARoundTrip(t *testing.T) {
	raw := json.RawMessage(`{"share_id":"7","eddsa_pub":{"x":"1","y":"2"}}`)
	st := KeygenLocalState{
		PubKey:          "pubkey",
		Algo:            AlgoEdDSA,
		EdDSALocalData:  raw,
		ParticipantKeys: []string{"a", "b"},
		LocalPartyKey:   "a",
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got KeygenLocalState
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Algo != AlgoEdDSA {
		t.Fatalf("algo not preserved: got %q", got.Algo)
	}
	if len(got.EdDSALocalData) == 0 {
		t.Fatalf("eddsa local data not preserved")
	}
	if string(got.EdDSALocalData) != string(raw) {
		t.Fatalf("eddsa local data altered: got %s", string(got.EdDSALocalData))
	}

	// Legacy state: no algo, no eddsa data -> empty algo (treated as ECDSA), nil eddsa data.
	legacy := `{"pub_key":"x","local_data":{},"participant_keys":[],"local_party_key":""}`
	var ls KeygenLocalState
	if err := json.Unmarshal([]byte(legacy), &ls); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if ls.Algo != "" {
		t.Fatalf("legacy algo should be empty, got %q", ls.Algo)
	}
	if ls.EdDSALocalData != nil {
		t.Fatalf("legacy eddsa data should be nil")
	}
}
