package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type MsgSetVersionSuite struct{}

var _ = Suite(&MsgSetVersionSuite{})

func (MsgSetVersionSuite) TestMsgSetVersionSuite(c *C) {
	acc1 := GetRandomBech32Addr()
	c.Assert(acc1.Empty(), Equals, false)
	msg := NewMsgSetVersion("2.0.0", acc1)
	c.Assert(msg.ValidateBasic(), IsNil)
	c.Assert(msg.GetSigners(), NotNil)
	c.Assert(msg.GetSigners()[0].String(), Equals, acc1.String())

	msg1 := NewMsgSetVersion("2.0.0", cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Check(err1, NotNil)
	c.Check(errors.Is(err1, se.ErrInvalidAddress), Equals, true)

	v := GetCurrentVersion()
	v.Build = []string{
		"whatever",
		"",
	}
	msg2 := NewMsgSetVersion(v.String(), acc1)
	err2 := msg2.ValidateBasic()
	c.Check(err2, NotNil)
	c.Check(errors.Is(err2, se.ErrUnknownRequest), Equals, true)
}
