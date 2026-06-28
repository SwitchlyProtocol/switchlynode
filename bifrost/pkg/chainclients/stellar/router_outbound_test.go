package stellar

// Tests for the router-based outbound path (transfer_out with the full memo).
// These cover the deterministic, network-free construction logic; the live
// simulate/sign/broadcast path requires a Stellar testnet to validate end-to-end.

import (
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/stellar/go/xdr"
	. "gopkg.in/check.v1"

	stypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

const testRouterContract = "CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH"

// newTestVaultPubKey generates a valid bech32 pubkey that derives a Stellar address
// under the test build, so we don't depend on hard-coded fixtures.
func newTestVaultPubKey(c *C) common.PubKey {
	priv := secp256k1.GenPrivKey()
	tmp, err := codec.FromTmPubKeyInterface(priv.PubKey())
	c.Assert(err, IsNil)
	bech32, err := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, tmp)
	c.Assert(err, IsNil)
	pk, err := common.NewPubKey(bech32)
	c.Assert(err, IsNil)
	return pk
}

func (s *StellarClientTestSuite) TestScvalI128FromBaseUnits(c *C) {
	v, err := scvalI128FromBaseUnits("15000000")
	c.Assert(err, IsNil)
	c.Assert(v.Type, Equals, xdr.ScValTypeScvI128)
	c.Assert(v.I128, NotNil)
	c.Assert(v.I128.Hi, Equals, xdr.Int64(0))
	c.Assert(v.I128.Lo, Equals, xdr.Uint64(15000000))

	z, err := scvalI128FromBaseUnits("0")
	c.Assert(err, IsNil)
	c.Assert(z.I128.Lo, Equals, xdr.Uint64(0))

	_, err = scvalI128FromBaseUnits("not-a-number")
	c.Assert(err, NotNil)

	_, err = scvalI128FromBaseUnits("-5")
	c.Assert(err, NotNil)
}

func (s *StellarClientTestSuite) TestBuildTransferOutInvokeOp(c *C) {
	orig := s.client.routerAddress
	defer func() { s.client.routerAddress = orig }()
	s.client.routerAddress = testRouterContract

	vaultPubKey := newTestVaultPubKey(c)
	vaultAddr := s.client.GetAddress(vaultPubKey)
	c.Assert(vaultAddr, Not(Equals), "")

	// 1.5 XLM in SwitchlyProtocol's 8-decimal units; XLM uses 7 decimals on Stellar,
	// so this must convert to 15,000,000 stroops.
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(150000000))
	memo := "OUT:56D832CB5365562BC87F8A309CB3D3A518A5D86715C574D6BED791F42F2F9762"
	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GB3F6OUNAIAMDGV5LVT2UFWAWACQ2LRE6OMF2OTSYCY7AKXHAMEXG37V"),
		VaultPubKey: vaultPubKey,
		Coins:       common.Coins{coin},
		Memo:        memo,
	}

	op, err := s.client.buildTransferOutInvokeOp(txOutItem, memo)
	c.Assert(err, IsNil)
	c.Assert(op, NotNil)
	c.Assert(op.SourceAccount, Equals, vaultAddr)

	hf := op.HostFunction
	c.Assert(hf.Type, Equals, xdr.HostFunctionTypeHostFunctionTypeInvokeContract)
	c.Assert(hf.InvokeContract, NotNil)
	c.Assert(string(hf.InvokeContract.FunctionName), Equals, "transfer_out")
	c.Assert(hf.InvokeContract.ContractAddress.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)

	args := hf.InvokeContract.Args
	c.Assert(len(args), Equals, 5)

	// vault (account), to (account), asset (SAC contract)
	c.Assert(args[0].Type, Equals, xdr.ScValTypeScvAddress)
	c.Assert(args[0].Address.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)
	c.Assert(args[1].Type, Equals, xdr.ScValTypeScvAddress)
	c.Assert(args[1].Address.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)
	c.Assert(args[2].Type, Equals, xdr.ScValTypeScvAddress)
	c.Assert(args[2].Address.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)

	// amount: 1.5 XLM -> 15,000,000 stroops
	c.Assert(args[3].Type, Equals, xdr.ScValTypeScvI128)
	c.Assert(args[3].I128.Hi, Equals, xdr.Int64(0))
	c.Assert(args[3].I128.Lo, Equals, xdr.Uint64(15000000))

	// memo: full string preserved (longer than Stellar's 28-byte limit)
	c.Assert(args[4].Type, Equals, xdr.ScValTypeScvString)
	c.Assert(string(*args[4].Str), Equals, memo)
	c.Assert(len(memo) > 28, Equals, true)
}

func (s *StellarClientTestSuite) TestBuildTransferOutInvokeOpErrors(c *C) {
	orig := s.client.routerAddress
	defer func() { s.client.routerAddress = orig }()

	vaultPubKey := newTestVaultPubKey(c)
	base := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GB3F6OUNAIAMDGV5LVT2UFWAWACQ2LRE6OMF2OTSYCY7AKXHAMEXG37V"),
		VaultPubKey: vaultPubKey,
		Coins:       common.Coins{common.NewCoin(common.XLMAsset, cosmos.NewUint(150000000))},
		Memo:        "OUT:abc",
	}

	// no router configured
	s.client.routerAddress = ""
	_, err := s.client.buildTransferOutInvokeOp(base, base.Memo)
	c.Assert(err, NotNil)

	s.client.routerAddress = testRouterContract

	// empty coins
	noCoins := base
	noCoins.Coins = common.Coins{}
	_, err = s.client.buildTransferOutInvokeOp(noCoins, noCoins.Memo)
	c.Assert(err, NotNil)

	// unsupported asset
	bad := base
	bad.Coins = common.Coins{common.NewCoin(common.Asset{Chain: common.StellarChain, Symbol: "NOPE", Ticker: "NOPE"}, cosmos.NewUint(1))}
	_, err = s.client.buildTransferOutInvokeOp(bad, bad.Memo)
	c.Assert(err, NotNil)
}
