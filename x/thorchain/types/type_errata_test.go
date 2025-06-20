package types

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
)

type TypeErrataTxSuite struct{}

var _ = Suite(&TypeErrataTxSuite{})

func (s *TypeErrataTxSuite) TestVoter(c *C) {
	errata := NewErrataTxVoter(
		GetRandomTxHash(),
		common.ETHChain,
	)
	c.Check(errata.Empty(), Equals, false)
	c.Check(errata.String() != "", Equals, true)

	addr := GetRandomBech32Addr()
	c.Check(errata.HasSigned(addr), Equals, false)
	errata.Sign(addr)
	c.Check(errata.Signers, HasLen, 1)
	c.Check(errata.HasSigned(addr), Equals, true)
	errata.Sign(addr) // ensure signing twice doesn't duplicate
	c.Check(errata.Signers, HasLen, 1)

	c.Check(errata.HasConsensus(nil), Equals, false)
	nas := NodeAccounts{
		NodeAccount{NodeAddress: addr, Status: NodeStatus_Active},
	}
	c.Check(errata.HasConsensus(nas), Equals, true)
	errata1 := NewErrataTxVoter(GetRandomTxHash(), common.EmptyChain)
	c.Check(errata1.Empty(), Equals, true)
}
