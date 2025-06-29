package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"

	cosmos "github.com/switchlyprotocol/switchlynode/v1/common/cosmos"

	. "gopkg.in/check.v1"
)

type MsgMimirSuite struct{}

var _ = Suite(&MsgMimirSuite{})

func (MsgMimirSuite) TestMsgMimir(c *C) {
	addr := GetRandomBech32Addr()
	m := NewMsgMimir("key", 12, addr)
	c.Check(m.ValidateBasic(), IsNil)
	EnsureMsgBasicCorrect(m, c)
	mEmpty := NewMsgMimir("", 0, cosmos.AccAddress{})
	c.Assert(mEmpty.ValidateBasic(), NotNil)
	msg1 := NewMsgMimir("ddd", 1, cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Assert(errors.Is(err1, se.ErrInvalidAddress), Equals, true)
}
