package types

import (
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/rs/zerolog/log"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/cmd"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/config"
)

////////////////////////////////////////////////////////////////////////////////////////
// Account
////////////////////////////////////////////////////////////////////////////////////////

// User holds a set of chain clients configured with a given private key.
type User struct {
	// Switchly is the switchly client for the account.
	Switchly switchlyclient.SwitchlyBridge

	// ChainClients is a map of chain to the corresponding client for the account.
	ChainClients map[common.Chain]LiteChainClient

	lock     chan struct{}
	pubkey   common.PubKey
	mnemonic string
}

// NewUser returns a new client using the private key from the given mnemonic.
func NewUser(mnemonic string, constructors map[common.Chain]LiteChainClientConstructor) *User {
	// create pubkey for mnemonic
	derivedPriv, err := hd.Secp256k1.Derive()(mnemonic, "", cmd.SwitchlyHDPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to derive private key")
	}
	privKey := hd.Secp256k1.Generate()(derivedPriv)
	s, err := cosmos.Bech32ifyPubKey(cosmos.Bech32PubKeyTypeAccPub, privKey.PubKey())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to bech32ify pubkey")
	}
	pubkey := common.PubKey(s)

	// add key to keyring
	registry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	kr := keyring.NewInMemory(cdc)
	name := strings.Split(mnemonic, " ")[0]
	_, err = kr.NewAccount(name, mnemonic, "", cmd.SwitchlyHDPath, hd.Secp256k1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to add account to keyring")
	}

	// create switchlyclient.Keys for chain client construction
	keys := switchlyclient.NewKeysWithKeybase(kr, name, "")

	// bifrost config for chain client construction
	cfg := config.GetBifrost()

	// create chain clients
	chainClients := make(map[common.Chain]LiteChainClient)
	for chain := range constructors {
		chainClients[chain], err = constructors[chain](chain, keys)
		if err != nil {
			log.Fatal().Err(err).Stringer("chain", chain).Msg("failed to create chain client")
		}
	}

	// create switchly bridge
	switchlyCfg := cfg.Switchly
	switchlyCfg.SignerName = name
	switchly, err := switchlyclient.NewSwitchlyBridge(switchlyCfg, nil, keys)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create switchly client")
	}

	return &User{
		ChainClients: chainClients,
		Switchly:     switchly,
		lock:         make(chan struct{}, 1),
		pubkey:       pubkey,
		mnemonic:     mnemonic,
	}
}

// Name returns the name of the account.
func (u *User) Name() string {
	return strings.Split(u.mnemonic, " ")[0]
}

// Acquire will attempt to acquire the lock. If the lock is already acquired, it will
// return false. If true is returned, the caller has locked and must release when done.
func (u *User) Acquire() bool {
	select {
	case u.lock <- struct{}{}:
		return true
	default:
		return false
	}
}

// IsLocked will return true if the lock is already acquired.
func (u *User) IsLocked() bool {
	select {
	case u.lock <- struct{}{}:
		<-u.lock
		return false
	default:
		return true
	}
}

// Release will release the lock.
func (u *User) Release() {
	<-u.lock
}

// PubKey returns the public key of the client.
func (u *User) PubKey() common.PubKey {
	return u.pubkey
}

// Address returns the address of the client for the given chain.
func (u *User) Address(chain common.Chain) common.Address {
	address, err := u.pubkey.GetAddress(chain)
	if err != nil {
		log.Fatal().Err(err).Stringer("chain", chain).Msg("failed to get address")
	}
	return address
}
