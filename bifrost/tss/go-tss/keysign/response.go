package keysign

import (
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/blame"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/common"
)

// signature
type Signature struct {
	Msg        string `json:"signed_msg"`
	R          string `json:"r"`
	S          string `json:"s"`
	RecoveryID string `json:"recovery_id"`
	// EncodedSignature is the base64 of the scheme's canonical serialized signature. For EdDSA this is
	// the 64-byte ed25519 signature (little-endian R||S) — R/S above are big-endian big.Int bytes and
	// must NOT be reassembled for ed25519. Empty/unused for ECDSA, which is reconstructed from R/S.
	EncodedSignature string `json:"encoded_signature,omitempty"`
}

// Response key sign response
type Response struct {
	Signatures []Signature   `json:"signatures"`
	Status     common.Status `json:"status"`
	Blame      blame.Blame   `json:"blame"`
}

func NewSignature(msg, r, s, recoveryID, encodedSignature string) Signature {
	return Signature{
		Msg:              msg,
		R:                r,
		S:                s,
		RecoveryID:       recoveryID,
		EncodedSignature: encodedSignature,
	}
}

func NewResponse(signatures []Signature, status common.Status, blame blame.Blame) Response {
	return Response{
		Signatures: signatures,
		Status:     status,
		Blame:      blame,
	}
}
