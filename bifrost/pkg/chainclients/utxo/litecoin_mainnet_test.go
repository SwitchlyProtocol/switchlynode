//go:build !stagenet && !mocknet
// +build !stagenet,!mocknet

package utxo

import (
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	ttypes "github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
	. "gopkg.in/check.v1"
)

func (s *LitecoinSuite) TestGetAddress(c *C) {
	ttypes.SetupConfigForTest()
	pubkey := common.PubKey("swtcpub1addwnpepqf0tc7xeqy23je53kmhyf420y76xhk2ufwjr6hkt52xe5na9w6rcy90jkdm")
	addr := s.client.GetAddress(pubkey)
	c.Assert(addr, Equals, "ltc1qhsn2d5m3rj8dev79zkx9rv84f809f4qf0wdkhd")
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
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
