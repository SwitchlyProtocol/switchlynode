package common

import (
	. "gopkg.in/check.v1"
)

type TickerSuite struct{}

var _ = Suite(&TickerSuite{})

func (s TickerSuite) TestTicker(c *C) {
	const SWTCTicker = Ticker("SWTC")

	swtcTicker, err := NewTicker("swtc")
	c.Assert(err, IsNil)
	c.Check(swtcTicker.IsEmpty(), Equals, false)
	c.Check(swtcTicker.Equals(SWTCTicker), Equals, true)
	c.Check(swtcTicker.String(), Equals, "SWTC")

	tomobTicker, err := NewTicker("TOMOB-1E1")
	c.Assert(err, IsNil)
	c.Assert(tomobTicker.String(), Equals, "TOMOB-1E1")
	_, err = NewTicker("t") // too short
	c.Assert(err, IsNil)

	maxCharacterTicker, err := NewTicker("TICKER789-XXX")
	c.Assert(err, IsNil)
	c.Assert(maxCharacterTicker.IsEmpty(), Equals, false)
	_, err = NewTicker("too long of a ticker") // too long
	c.Assert(err, NotNil)
}
