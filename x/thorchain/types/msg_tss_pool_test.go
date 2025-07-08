package types

import (
	"errors"
	"time"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
)

type MsgTssPoolSuite struct{}

var _ = Suite(&MsgTssPoolSuite{})

func (s *MsgTssPoolSuite) TestMsgTssPool(c *C) {
	pks := GetRandomPubKeySet()
	pkStrings := []string{pks.Secp256k1.String(), pks.Ed25519.String()}
	pk := GetRandomPubKey()
	// Use the address from one of the public keys in the keygen members
	addr, err := pks.Secp256k1.GetThorAddress()
	c.Assert(err, IsNil)
	keygenTime := time.Now().Unix()
	msg, err := NewMsgTssPool(pkStrings, pk, nil, nil, KeygenType_AsgardKeygen, 1, Blame{}, []string{common.SWTCNative.Chain.String()}, addr, keygenTime)
	c.Assert(err, IsNil)
	EnsureMsgBasicCorrect(msg, c)

	chains := []string{common.SWTCNative.Chain.String()}
	c.Check(msg.Chains, DeepEquals, chains)

	// ensure we can set the signer to another valid keygen member
	addr2, err := pks.Ed25519.GetThorAddress()
	c.Assert(err, IsNil)
	msg.Signer = addr2
	c.Check(msg.ValidateBasic(), IsNil)

	// invalid signer should fail
	msg.Signer = types.AccAddress{}
	c.Check(msg.ValidateBasic(), NotNil)
	msg.Signer = addr

	// duplicated chains should fail
	msg, err = NewMsgTssPool(pkStrings, pk, nil, nil, KeygenType_AsgardKeygen, 1, Blame{}, []string{common.SWTCNative.Chain.String(), common.SWTCNative.Chain.String()}, addr, keygenTime)
	c.Assert(err, IsNil)
	c.Check(msg.ValidateBasic(), NotNil)

	msg1, err := NewMsgTssPool(pkStrings, pk, nil, nil, KeygenType_AsgardKeygen, 1, Blame{}, chains, addr, keygenTime)
	c.Assert(err, IsNil)
	msg1.ID = ""
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Check(errors.Is(err1, se.ErrUnknownRequest), Equals, true)

	msg2, err := NewMsgTssPool(append(pkStrings, ""), pk, nil, nil, KeygenType_AsgardKeygen, 1, Blame{}, chains, addr, keygenTime)
	c.Assert(err, IsNil)
	err2 := msg2.ValidateBasic()
	c.Assert(err2, NotNil)
	c.Check(errors.Is(err2, se.ErrUnknownRequest), Equals, true)

	var allPks []string
	for i := 0; i < 110; i++ {
		allPks = append(allPks, GetRandomPubKey().String())
	}
	msg3, err := NewMsgTssPool(allPks, pk, nil, nil, KeygenType_AsgardKeygen, 1, Blame{}, chains, addr, keygenTime)
	c.Assert(err, IsNil)
	err3 := msg3.ValidateBasic()
	c.Assert(err3, NotNil)
	c.Check(errors.Is(err3, se.ErrUnknownRequest), Equals, true)
}
