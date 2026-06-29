package keysign

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"

	tsslibcommon "github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/p2p/conversion"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/common"
)

type NotifierTestSuite struct{}

var _ = Suite(&NotifierTestSuite{})

func (*NotifierTestSuite) SetUpSuite(c *C) {
	conversion.SetupBech32Prefix()
}

func (NotifierTestSuite) TestNewNotifier(c *C) {
	testMSg := [][]byte{[]byte("hello"), []byte("world")}
	poolPubKey := conversion.GetRandomPubKey()
	n, err := NewNotifier("", testMSg, poolPubKey, common.ECDSA)
	c.Assert(err, NotNil)
	c.Assert(n, IsNil)
	n, err = NewNotifier("aasfdasdf", nil, poolPubKey, common.ECDSA)
	c.Assert(err, NotNil)
	c.Assert(n, IsNil)

	n, err = NewNotifier("hello", testMSg, "", common.ECDSA)
	c.Assert(err, NotNil)
	c.Assert(n, IsNil)

	n, err = NewNotifier("hello", testMSg, poolPubKey, common.ECDSA)
	c.Assert(err, IsNil)
	c.Assert(n, NotNil)
	ch := n.GetResponseChannel()
	c.Assert(ch, NotNil)
}

func (NotifierTestSuite) TestNotifierHappyPath(c *C) {
	messageToSign := "yhEwrxWuNBGnPT/L7PNnVWg7gFWNzCYTV+GuX3tKRH8="
	buf, err := base64.StdEncoding.DecodeString(messageToSign)
	c.Assert(err, IsNil)
	messageID, err := common.MsgToHashString(buf)
	c.Assert(err, IsNil)
	poolPubKey := `tswitchpub1qg39rnhj7egrrhxmgx2rq3wsaes4lgeh2t2jtluqqhntxsr5qfwpsccayz3`
	n, err := NewNotifier(messageID, [][]byte{buf}, poolPubKey, common.ECDSA)
	c.Assert(err, IsNil)
	c.Assert(n, NotNil)
	sigFile := "../test_data/signature_notify/sig1.json"
	content, err := ioutil.ReadFile(sigFile)
	c.Assert(err, IsNil)
	c.Assert(content, NotNil)
	var signature signing.SignatureData
	err = json.Unmarshal(content, &signature)
	c.Assert(err, IsNil)

	sigInvalidFile := `../test_data/signature_notify/sig_invalid.json`
	contentInvalid, err := ioutil.ReadFile(sigInvalidFile)
	c.Assert(err, IsNil)
	c.Assert(contentInvalid, NotNil)
	var sigInvalid signing.SignatureData
	c.Assert(json.Unmarshal(contentInvalid, &sigInvalid), IsNil)
	// valid keysign peer , but invalid signature we should continue to listen
	finish, err := n.ProcessSignature([]*tsslibcommon.ECSignature{sigInvalid.GetSignature()})
	c.Assert(err, NotNil)
	c.Assert(finish, Equals, false)
	// valid signature from a keysign peer , we should accept it and bail out
	finish, err = n.ProcessSignature([]*tsslibcommon.ECSignature{signature.GetSignature()})
	c.Assert(err, IsNil)
	c.Assert(finish, Equals, true)

	result := <-n.GetResponseChannel()
	c.Assert(result, NotNil)
	c.Assert(signature.GetSignature().String() == result[0].String(), Equals, true)
}

// TestNotifierEdDSA verifies the ed25519 branch of verifySignature: a real crypto/ed25519 signature
// over the message, with the hex-encoded 32-byte pubkey as the pool key, must verify; a tampered
// signature must not. This mirrors how a threshold ed25519 signature is checked for the Stellar vault.
func (NotifierTestSuite) TestNotifierEdDSA(c *C) {
	pub, priv, err := ed25519.GenerateKey(nil)
	c.Assert(err, IsNil)
	msg := []byte("stellar-tx-hash-placeholder-32by")
	sig := ed25519.Sign(priv, msg)

	n, err := NewNotifier("eddsa-msg-id", [][]byte{msg}, hex.EncodeToString(pub), common.EdDSA)
	c.Assert(err, IsNil)
	c.Assert(n, NotNil)

	good := &tsslibcommon.ECSignature{Signature: sig, M: msg}
	ok, err := n.verifySignature(good, msg)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// tamper the signature -> must fail verification (no error, just false)
	bad := make([]byte, len(sig))
	copy(bad, sig)
	bad[0] ^= 0xff
	ok, err = n.verifySignature(&tsslibcommon.ECSignature{Signature: bad, M: msg}, msg)
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, false)
}
