package tss

import (
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keygen"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keysign"
)

// Server define the necessary functionality should be provide by a TSS Server implementation
type Server interface {
	Start() error
	Stop()
	GetLocalPeerID() string
	GetKnownPeers() []PeerInfo
	Keygen(req keygen.Request) (keygen.Response, error)
	KeySign(req keysign.Request) (keysign.Response, error)
}
