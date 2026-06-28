package keygen

import "github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/common"

// Request request to do keygen
type Request struct {
	Keys        []string `json:"keys"`
	BlockHeight int64    `json:"block_height"`
	Version     string   `json:"tss_version"`
	// Algo selects the keygen scheme. Empty/"ecdsa" = secp256k1 (every chain today); "eddsa" =
	// ed25519 (Stellar). Older nodes omit it, so empty must be treated as ECDSA.
	Algo common.Algo `json:"algo,omitempty"`
}

// NewRequest creeate a new instance of keygen.Request
func NewRequest(keys []string, blockHeight int64, version string) Request {
	return Request{
		Keys:        keys,
		BlockHeight: blockHeight,
		Version:     version,
	}
}
