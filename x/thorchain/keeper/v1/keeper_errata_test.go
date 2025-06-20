package keeperv1

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
)

type KeeperErrataTxSuite struct{}

var _ = Suite(&KeeperErrataTxSuite{})

func (s *KeeperErrataTxSuite) TestErrataTxVoter(c *C) {
	ctx, k := setupKeeperForTest(c)

	txID := GetRandomTxHash()
	voter := NewErrataTxVoter(txID, common.ETHChain)

	k.SetErrataTxVoter(ctx, voter)
	voter, err := k.GetErrataTxVoter(ctx, voter.TxID, voter.Chain)
	c.Assert(err, IsNil)
	c.Check(voter.TxID.Equals(txID), Equals, true)
	c.Check(voter.Chain.Equals(common.ETHChain), Equals, true)
	c.Check(k.GetErrataTxVoterIterator(ctx), NotNil)

	errtaVoter, err := k.GetErrataTxVoter(ctx, GetRandomTxHash(), common.ETHChain)
	c.Check(err, IsNil)
	c.Check(errtaVoter.Empty(), Equals, false)
}
