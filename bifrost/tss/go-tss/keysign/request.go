package keysign

import "github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/common"

// Request request to sign a message
type Request struct {
	PoolPubKey    string   `json:"pool_pub_key"` // pub key of the pool that we would like to send this message from
	Messages      []string `json:"messages"`     // base64 encoded message to be signed
	SignerPubKeys []string `json:"signer_pub_keys"`
	BlockHeight   int64    `json:"block_height"`
	Version       string   `json:"tss_version"`
	// Algo selects the signing scheme. Empty/"ecdsa" = secp256k1 (every chain today); "eddsa" =
	// ed25519 (Stellar). Older nodes omit it, so empty must be treated as ECDSA.
	Algo common.Algo `json:"algo,omitempty"`
}

func NewRequest(pk string, msgs []string, blockHeight int64, signers []string, version string) Request {
	return Request{
		PoolPubKey:    pk,
		Messages:      msgs,
		SignerPubKeys: signers,
		BlockHeight:   blockHeight,
		Version:       version,
	}
}
