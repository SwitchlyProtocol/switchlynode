//go:build !stagenet && !mocknet
// +build !stagenet,!mocknet

package utxo

import (
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	ttypes "github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
	. "gopkg.in/check.v1"
)

func (s *LitecoinSuite) TestGetAddress(c *C) {
	ttypes.SetupConfigForTest()
	pubkey := common.PubKey("switchpub1addwnpepq23zu782ghdfmdskqujyfr7sdllvk7jpu8y883ry4lnqsansvuzyjlzs7yj")
	addr := s.client.GetAddress(pubkey)
	c.Assert(addr, Not(Equals), "")
}

func (s *LitecoinSuite) TestConfirmationCountReady(c *C) {
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain:    common.LTCChain,
		TxArray:  nil,
		Filtered: true,
		MemPool:  false,
	}), Equals, true)
	pkey := ttypes.GetRandomPubKey()
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(123456)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered: true,
		MemPool:  true,
	}), Equals, true)
	s.client.currentBlockHeight.Store(3)
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(123456)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, true)

	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(12345600000)),
				},
				Gas: common.Gas{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(40000)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 5,
	}), Equals, false)
}

func (s *LitecoinSuite) TestGetConfirmationCount(c *C) {
	pkey := ttypes.GetRandomPubKey()
	// no tx in item , confirmation count should be 0
	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain:   common.LTCChain,
		TxArray: nil,
	}), Equals, int64(0))
	// mempool txin , confirmation count should be 0
	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.BTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(123456)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              true,
		ConfirmationRequired: 0,
	}), Equals, int64(0))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(123456)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(0))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(12345600)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(0))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(22345600)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(1))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.LTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "ltc1qycp4dta4wse4825nk9j4wt42357nj3tdn26ezm",
				To:          "ltc1qjw8h4l3dtz5xxc7uyh5ys70qkezspgfu8hg5j3",
				Coins: common.Coins{
					common.NewCoin(common.LTCAsset, cosmos.NewUint(123456000)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(6))
}
