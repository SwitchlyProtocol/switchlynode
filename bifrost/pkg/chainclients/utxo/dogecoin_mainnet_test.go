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

func (s *DogecoinSuite) TestGetAddress(c *C) {
	ttypes.SetupConfigForTest()
	pubkey := common.PubKey("tswitchpub19daym3mcdfmlr4hck2qmg2l87sdnwh88avr28fju6e87rv6qsq2sytzzrta")
	addr := s.client.GetAddress(pubkey)
	c.Assert(addr, Equals, "DNHwstQwWzW3yTBarsmrzNs47JujMczfbN")
}

func (s *DogecoinSuite) TestConfirmationCountReady(c *C) {
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain:    common.DOGEChain,
		TxArray:  nil,
		Filtered: true,
		MemPool:  false,
	}), Equals, true)
	pkey := ttypes.GetRandomPubKey()
	c.Assert(s.client.ConfirmationCountReady(types.TxIn{
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(123456)),
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
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(123456)),
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
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(12345600000)),
				},
				Gas: common.Gas{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(40000)),
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

func (s *DogecoinSuite) TestGetConfirmationCount(c *C) {
	pkey := ttypes.GetRandomPubKey()
	// no tx in item , confirmation count should be 0
	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain:   common.DOGEChain,
		TxArray: nil,
	}), Equals, int64(0))
	// mempool txin , confirmation count should be 0
	c.Assert(s.client.GetConfirmationCount(types.TxIn{
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(123456)),
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
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(123456)),
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
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(12345600)),
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
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(223456000)),
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
		Chain: common.DOGEChain,
		TxArray: []*types.TxInItem{
			{
				BlockHeight: 2,
				Tx:          "9efcbb7f05f810434f777ff48c11bf526320493cb7721975efb938f1ecb6c031",
				Sender:      "bc1q0s4mg25tu6termrk8egltfyme4q7sg3h0e56p3",
				To:          "bc1q2gjc0rnhy4nrxvuklk6ptwkcs9kcr59mcl2q9j",
				Coins: common.Coins{
					common.NewCoin(common.DOGEAsset, cosmos.NewUint(123456000000)),
				},
				Memo:                "noop",
				ObservedVaultPubKey: pkey,
			},
		},
		Filtered:             true,
		MemPool:              false,
		ConfirmationRequired: 0,
	}), Equals, int64(20))
}
