package main

import (
	"errors"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/blame"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/common"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keygen"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keysign"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/tss"
)

type MockTssServer struct {
	failToStart   bool
	failToKeyGen  bool
	failToKeySign bool
}

func (mts *MockTssServer) Start() error {
	if mts.failToStart {
		return errors.New("you ask for it")
	}
	return nil
}

func (mts *MockTssServer) Stop() {
}

func (mts *MockTssServer) GetLocalPeerID() string {
	return conversion.GetRandomPeerID().String()
}

func (mts *MockTssServer) GetKnownPeers() []tss.PeerInfo {
	return []tss.PeerInfo{}
}

func (mts *MockTssServer) Keygen(req keygen.Request) (keygen.Response, error) {
	if mts.failToKeyGen {
		return keygen.Response{}, errors.New("you ask for it")
	}
	return keygen.NewResponse(conversion.GetRandomPubKey(), "whatever", common.Success, blame.Blame{}), nil
}

func (mts *MockTssServer) KeySign(req keysign.Request) (keysign.Response, error) {
	if mts.failToKeySign {
		return keysign.Response{}, errors.New("you ask for it")
	}
	newSig := keysign.NewSignature("", "", "", "")
	return keysign.NewResponse([]keysign.Signature{newSig}, common.Success, blame.Blame{}), nil
}
