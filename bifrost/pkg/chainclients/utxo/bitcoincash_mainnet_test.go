//go:build !stagenet && !mocknet
// +build !stagenet,!mocknet

package utxo

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	ttypes "github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

func (s *BitcoinCashSuite) TestGetAddress(c *C) {
	ttypes.SetupConfigForTest()
	pubkey := common.PubKey("swtcpub1addwnpepqfpgf05gts34mk3mdq3dc7qz5yjssydw9xy3237ny30jzkd6v78qs35ztdg")
	addr := s.client.GetAddress(pubkey)
	c.Assert(addr, Equals, "qz2m57zx46nuqtwfawn0pgrmump9r4e4jcshwa2ts6")
}

func (s *BitcoinCashSuite) TestConfirmationCountReady(c *C) {
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain:    common.BCHChain,
		TxArray:  nil,
		Filtered: true,
		MemPool:  false,
	}), Equals, true)
	pkey := ttypes.GetRandomPubKey()
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(123456)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered: true,
		MemPool:  true,
	}), Equals, true)
	s.client.currentBlockHeight.Store(3)
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(123456)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, true)

	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(12345600000)),
				},
				Gas: common.Gas{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(40000)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 5,
	}), Equals, false)
}

func (s *BitcoinCashSuite) TestGetConfirmationCount(c *C) {
	pkey := ttypes.GetRandomPubKey()
	// no tx in item , confirmation count should be 0
	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain:   common.BCHChain,
		TxArray: nil,
	}), Equals, int64(0))
	// mempool txin , confirmation count should be 0
	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.BTCChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BTCAsset, cosmos.NewUint(123456)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              true,
		ConfirmationRequired: 0,
	}), Equals, int64(0))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(123456)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(0))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(12345600)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(0))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(22345600)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(1))

	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.BCHChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "24ed2d26fd5d4e0e8fa86633e40faf1bdfc8d1903b1cd02855286312d48818a2",
				Sender:      "qqqzdh86crxjpyh2tgfy7gyfcwk4k74ze55ympqehp",
				To:          "qpfztpuwwujkvvenjm7mg9d6mzqkmqwshv07z34njm",
				Coins: common.Coins{
					common.NewCoin(common.BCHAsset, cosmos.NewUint(123456000)),
				},
				Memo:                "MEMO",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(6))
}
