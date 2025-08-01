package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/blame"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/common"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keygen"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keysign"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/tss"
	. "gopkg.in/check.v1"
)

func TestPackage(t *testing.T) { TestingT(t) }

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
	return keysign.NewResponse(nil, common.Success, blame.Blame{}), nil
}

type HealthServerTestSuite struct{}

var _ = Suite(&HealthServerTestSuite{})

func (HealthServerTestSuite) TestHealthServer(c *C) {
	tssServer := &MockTssServer{}
	s := NewHealthServer("127.0.0.1:8080", tssServer, nil)
	c.Assert(s, NotNil)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.Start()
		c.Assert(err, IsNil)
	}()
	time.Sleep(time.Second)
	c.Assert(s.Stop(), IsNil)
}

func (HealthServerTestSuite) TestPingHandler(c *C) {
	tssServer := &MockTssServer{}
	s := NewHealthServer("127.0.0.1:8080", tssServer, nil)
	c.Assert(s, NotNil)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	res := httptest.NewRecorder()
	s.pingHandler(res, req)
	c.Assert(res.Code, Equals, http.StatusOK)
}

func (HealthServerTestSuite) TestGetP2pIDHandler(c *C) {
	tssServer := &MockTssServer{}
	s := NewHealthServer("127.0.0.1:8080", tssServer, nil)
	c.Assert(s, NotNil)
	req := httptest.NewRequest(http.MethodGet, "/p2pid", nil)
	res := httptest.NewRecorder()
	s.getP2pIDHandler(res, req)
	c.Assert(res.Code, Equals, http.StatusOK)
}
