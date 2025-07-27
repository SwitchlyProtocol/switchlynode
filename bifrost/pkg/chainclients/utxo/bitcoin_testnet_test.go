//go:build testnet
// +build testnet

package utxo

import (
	"github.com/btcsuite/btcd/btcjson"
	. "gopkg.in/check.v1"
)

func (s *BitcoinSuite) TestGetAddressesFromScriptPubKeyResult(c *C) {
	addresses := s.client.getAddressesFromScriptPubKeyBTC(btcjson.ScriptPubKeyResult{
		Asm:     "0 de4f4fce2642935d2b9fc7b28bcc9de20ebf2864",
		Hex:     "0014de4f4fce2642935d2b9fc7b28bcc9de20ebf2864",
		ReqSigs: 1,
		Type:    "witness_v0_keyhash",
		Addresses: []string{
			"tb1qme85ln3xg2f462ulc7eghnyaug8t72ryhwzs8f",
		},
	})
	c.Assert(addresses, HasLen, 1)
	c.Assert(addresses[0], Equals, "tb1qme85ln3xg2f462ulc7eghnyaug8t72ryhwzs8f")

	addresses = s.client.getAddressesFromScriptPubKeyBTC(btcjson.ScriptPubKeyResult{
		Asm:       "0 de4f4fce2642935d2b9fc7b28bcc9de20ebf2864",
		Hex:       "0014de4f4fce2642935d2b9fc7b28bcc9de20ebf2864",
		ReqSigs:   1,
		Type:      "witness_v0_keyhash",
		Addresses: nil,
	})
	c.Assert(addresses, HasLen, 1)
	c.Assert(addresses[0], Equals, "tb1qme85ln3xg2f462ulc7eghnyaug8t72ryhwzs8f")
}

func (s *BitcoinSuite) TestGetAccount(c *C) {
	acct, err := s.client.GetAccount("tswitchpub1addwnpepqvvj6ln6xq4afle87jxhs5nzrcxnewrr5mwk0h2yuqc55fdr42rxs4208ul", nil)
	c.Assert(err, IsNil)
	c.Assert(acct.AccountNumber, Equals, int64(0))
	c.Assert(acct.Sequence, Equals, int64(0))
	c.Assert(acct.Coins[0].Amount.Uint64(), Equals, uint64(2502000000))

	acct1, err := s.client.GetAccount("", nil)
	c.Assert(err, NotNil)
	c.Assert(acct1.AccountNumber, Equals, int64(0))
	c.Assert(acct1.Sequence, Equals, int64(0))
	c.Assert(acct1.Coins, HasLen, 0)
}
