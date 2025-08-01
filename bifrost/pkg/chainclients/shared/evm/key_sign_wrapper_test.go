package evm

import (
	"math/big"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cKeys "github.com/cosmos/cosmos-sdk/crypto/keyring"
	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/tss"
	"github.com/switchlyprotocol/switchlynode/v1/cmd"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/config"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

const MaxContractGas = 80000

type KeysignWrapperTestSuite struct {
	thorKeys *thorclient.Keys
	wrapper  *KeySignWrapper
}

var _ = Suite(
	&KeysignWrapperTestSuite{},
)

// SetUpSuite setup the test conditions
func (s *KeysignWrapperTestSuite) SetUpSuite(c *C) {
	cfg := config.BifrostClientConfiguration{
		ChainID:      "thorchain",
		SignerName:   "bob",
		SignerPasswd: "password",
	}

	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kb := cKeys.NewInMemory(cdc)
	_, _, err := kb.NewMnemonic(cfg.SignerName, cKeys.English, cmd.SwitchlyHDPath, cfg.SignerPasswd, hd.Secp256k1)
	c.Assert(err, IsNil)
	s.thorKeys = thorclient.NewKeysWithKeybase(kb, cfg.SignerName, cfg.SignerPasswd)

	privateKey, err := s.thorKeys.GetPrivateKey()
	c.Assert(err, IsNil)
	temp, err := cryptocodec.ToCmtPubKeyInterface(privateKey.PubKey())
	c.Assert(err, IsNil)
	pk, err := common.NewPubKeyFromCrypto(temp)
	c.Assert(err, IsNil)
	keyMgr := &tss.MockThorchainKeyManager{}
	ethPrivateKey, err := GetPrivateKey(privateKey)
	c.Assert(err, IsNil)
	c.Assert(ethPrivateKey, NotNil)
	wrapper, err := NewKeySignWrapper(ethPrivateKey, pk, keyMgr, big.NewInt(15), "AVAX")
	c.Assert(err, IsNil)
	c.Assert(wrapper, NotNil)
	s.wrapper = wrapper
}

func (s *KeysignWrapperTestSuite) TestGetPrivKey(c *C) {
	c.Assert(s.wrapper.GetPrivKey(), NotNil)
}

func (s *KeysignWrapperTestSuite) TestGetPubKey(c *C) {
	c.Assert(s.wrapper.GetPubKey(), NotNil)
}

func (s *KeysignWrapperTestSuite) TestSign(c *C) {
	buf, err := s.wrapper.Sign(nil, types.GetRandomPubKey())
	c.Assert(err, NotNil)
	c.Assert(buf, IsNil)
	createdTx := etypes.NewTransaction(0, ecommon.HexToAddress("0x7d182d6a138eaa06f6f452bc3f8fc57e17d1e193"), big.NewInt(1), MaxContractGas, big.NewInt(1), []byte("whatever"))
	buf, err = s.wrapper.Sign(createdTx, common.EmptyPubKey)
	c.Assert(err, NotNil)
	c.Assert(buf, IsNil)
	_, err = s.wrapper.Sign(createdTx, s.wrapper.pubKey)
	c.Assert(err, IsNil)
	// test sign with TSS
	buf, err = s.wrapper.Sign(createdTx, types.GetRandomPubKey())
	c.Assert(err, NotNil)
	c.Assert(buf, IsNil)
}
