package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss/go-tss/keygen"
)

func TestPackage(t *testing.T) { TestingT(t) }

type TssHttpServerTestSuite struct{}

var _ = Suite(&TssHttpServerTestSuite{})

func (TssHttpServerTestSuite) TestNewTssHttpServer(c *C) {
	tssServer := &MockTssServer{}
	s := NewTssHttpServer("127.0.0.1:8080", tssServer)
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
	tssServer.failToStart = true
	c.Assert(s.Start(), NotNil)
}

func (TssHttpServerTestSuite) TestPingHandler(c *C) {
	tssServer := &MockTssServer{}
	s := NewTssHttpServer("127.0.0.1:8080", tssServer)
	c.Assert(s, NotNil)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	res := httptest.NewRecorder()
	s.pingHandler(res, req)
	c.Assert(res.Code, Equals, http.StatusOK)
}

func (TssHttpServerTestSuite) TestGetP2pIDHandler(c *C) {
	tssServer := &MockTssServer{}
	s := NewTssHttpServer("127.0.0.1:8080", tssServer)
	c.Assert(s, NotNil)
	req := httptest.NewRequest(http.MethodGet, "/p2pid", nil)
	res := httptest.NewRecorder()
	s.getP2pIDHandler(res, req)
	c.Assert(res.Code, Equals, http.StatusOK)
}

func (TssHttpServerTestSuite) TestKeygenHandler(c *C) {
	normalKeygenRequest := `{"keys":["tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3", "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3", "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3", "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3"]}`
	testCases := []struct {
		name          string
		reqProvider   func() *http.Request
		setter        func(s *MockTssServer)
		resultChecker func(c *C, w *httptest.ResponseRecorder)
	}{
		{
			name: "method get should return status method not allowed",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/keygen", nil)
			},
			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusMethodNotAllowed)
			},
		},
		{
			name: "nil request body should return status bad request",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/keygen", nil)
			},
			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusBadRequest)
			},
		},
		{
			name: "fail to keygen should return status internal server error",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/keygen",
					bytes.NewBufferString(normalKeygenRequest))
			},
			setter: func(s *MockTssServer) {
				s.failToKeyGen = true
			},
			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusOK)
			},
		},
		{
			name: "normal",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/keygen",
					bytes.NewBufferString(normalKeygenRequest))
			},

			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusOK)
				var resp keygen.Response
				c.Assert(json.Unmarshal(w.Body.Bytes(), &resp), IsNil)
			},
		},
	}
	for _, tc := range testCases {
		c.Log(tc.name)
		tssServer := &MockTssServer{}
		s := NewTssHttpServer("127.0.0.1:8080", tssServer)
		c.Assert(s, NotNil)
		if tc.setter != nil {
			tc.setter(tssServer)
		}
		req := tc.reqProvider()
		res := httptest.NewRecorder()
		s.keygenHandler(res, req)
		tc.resultChecker(c, res)
	}
}

func (TssHttpServerTestSuite) TestKeysignHandler(c *C) {
	var normalKeySignRequest string = `{
    "pool_pub_key": "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
    "message": "helloworld",
    "signer_pub_keys": [
        "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
        "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
        "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3",
        "tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3"
    ]
}`
	testCases := []struct {
		name          string
		reqProvider   func() *http.Request
		setter        func(s *MockTssServer)
		resultChecker func(c *C, w *httptest.ResponseRecorder)
	}{
		{
			name: "method get should return status method not allowed",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/keysign", nil)
			},
			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusMethodNotAllowed)
			},
		},
		{
			name: "nil request body should return status bad request",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/keysign", nil)
			},
			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusBadRequest)
			},
		},
		{
			name: "fail to keygen should return status internal server error",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/keysign",
					bytes.NewBufferString(normalKeySignRequest))
			},
			setter: func(s *MockTssServer) {
				s.failToKeySign = true
			},
			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusInternalServerError)
			},
		},
		{
			name: "normal",
			reqProvider: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/keysign",
					bytes.NewBufferString(normalKeySignRequest))
			},

			resultChecker: func(c *C, w *httptest.ResponseRecorder) {
				c.Assert(w.Code, Equals, http.StatusOK)
				var resp keygen.Response
				c.Assert(json.Unmarshal(w.Body.Bytes(), &resp), IsNil)
			},
		},
	}
	for _, tc := range testCases {
		c.Log(tc.name)
		tssServer := &MockTssServer{}
		s := NewTssHttpServer("127.0.0.1:8080", tssServer)
		c.Assert(s, NotNil)
		if tc.setter != nil {
			tc.setter(tssServer)
		}
		req := tc.reqProvider()
		res := httptest.NewRecorder()
		s.keySignHandler(res, req)
		tc.resultChecker(c, res)
	}
}
