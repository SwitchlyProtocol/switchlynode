package messages

import (
	"math/big"
	"testing"

	btss "github.com/binance-chain/tss-lib/tss"
	. "gopkg.in/check.v1"
)

func TestPackage(t *testing.T) { TestingT(t) }

type SWITCHLYChainTSSMessageTypeSuite struct{}

var _ = Suite(&SWITCHLYChainTSSMessageTypeSuite{})

func (SWITCHLYChainTSSMessageTypeSuite) TestSWITCHLYChainTSSMessageType_String(c *C) {
	m := map[SWITCHLYChainTSSMessageType]string{
		TSSKeyGenMsg:     "TSSKeyGenMsg",
		TSSKeySignMsg:    "TSSKeySignMsg",
		TSSKeyGenVerMsg:  "TSSKeyGenVerMsg",
		TSSKeySignVerMsg: "TSSKeySignVerMsg",
	}
	for k, v := range m {
		c.Assert(k.String(), Equals, v)
	}
}

func (SWITCHLYChainTSSMessageTypeSuite) TestWireMessage(c *C) {
	bi := new(big.Int).SetBytes([]byte("whatever"))
	wm := WireMessage{
		Routing: &btss.MessageRouting{
			From:                    btss.NewPartyID("1", "", bi),
			To:                      nil,
			IsBroadcast:             true,
			IsToOldCommittee:        false,
			IsToOldAndNewCommittees: false,
		},
		RoundInfo: "hello",
		Message:   nil,
	}
	cacheKey := wm.GetCacheKey()
	c.Assert(cacheKey, Equals, "1-hello")
}
