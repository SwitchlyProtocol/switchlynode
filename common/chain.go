package common

import (
	"errors"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/cosmos/cosmos-sdk/types"
	dogchaincfg "github.com/eager7/dogd/chaincfg"
	"github.com/hashicorp/go-multierror"
	ltcchaincfg "github.com/ltcsuite/ltcd/chaincfg"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

const (
	EmptyChain    = Chain("")
	BSCChain      = Chain("BSC")
	ETHChain      = Chain("ETH")
	BTCChain      = Chain("BTC")
	LTCChain      = Chain("LTC")
	BCHChain      = Chain("BCH")
	DOGEChain     = Chain("DOGE")
	SWITCHLYChain = Chain("SWITCHLY")
	GAIAChain     = Chain("GAIA")
	AVAXChain     = Chain("AVAX")
	BASEChain     = Chain("BASE")
	XRPChain      = Chain("XRP")
	StellarChain  = Chain("XLM")

	SigningAlgoSecp256k1 = SigningAlgo("secp256k1")
	SigningAlgoEd25519   = SigningAlgo("ed25519")
)

var AllChains = [...]Chain{
	BSCChain,
	ETHChain,
	BTCChain,
	LTCChain,
	BCHChain,
	DOGEChain,
	SWITCHLYChain,
	GAIAChain,
	AVAXChain,
	BASEChain,
	XRPChain,
	StellarChain,
}

type SigningAlgo string

type Chain string

// Chains represent a slice of Chain
type Chains []Chain

// Valid validates chain format, should consist only of uppercase letters
func (c Chain) Valid() error {
	if len(c) < 3 {
		return errors.New("chain id len is less than 3")
	}
	if len(c) > 10 {
		return errors.New("chain id len is more than 10")
	}
	for _, ch := range string(c) {
		if ch < 'A' || ch > 'Z' {
			return errors.New("chain id can consist only of uppercase letters")
		}
	}
	return nil
}

// NewChain create a new Chain and default the siging_algo to Secp256k1
func NewChain(chainID string) (Chain, error) {
	chain := Chain(strings.ToUpper(chainID))
	if err := chain.Valid(); err != nil {
		return chain, err
	}
	return chain, nil
}

// Equals compare two chain to see whether they represent the same chain
func (c Chain) Equals(c2 Chain) bool {
	return strings.EqualFold(c.String(), c2.String())
}

func (c Chain) IsSWITCHLYChain() bool {
	return c.Equals(SWITCHLYChain)
}

// IsTHORChain checks if chain is THORChain (legacy method)
// NOTE: This method is deprecated, use IsSWITCHLYChain instead
func (c Chain) IsTHORChain() bool {
	return c.IsSWITCHLYChain()
}

// IsSwitchlyChain checks if chain is SwitchlyChain (preferred method)
func (c Chain) IsSwitchlyChain() bool {
	return c.Equals(SWITCHLYChain)
}

// GetEVMChains returns all "EVM" chains connected to SWITCHLYChain
// "EVM" is defined, in thornode's context, as a chain that:
// - uses 0x as an address prefix
// - has a "Router" Smart Contract
func GetEVMChains() []Chain {
	return []Chain{ETHChain, AVAXChain, BSCChain, BASEChain}
}

// GetRouterChains returns all chains that have router smart contracts
// This includes EVM chains and other chains like Stellar that use router contracts
func GetRouterChains() []Chain {
	return []Chain{ETHChain, AVAXChain, BSCChain, BASEChain, StellarChain}
}

// GetUTXOChains returns all "UTXO" chains connected to SWITCHLYChain.
func GetUTXOChains() []Chain {
	return []Chain{BTCChain, LTCChain, BCHChain, DOGEChain}
}

// IsEVM returns true if given chain is an EVM chain.
// See working definition of an "EVM" chain in the
// `GetEVMChains` function description
func (c Chain) IsEVM() bool {
	evmChains := GetEVMChains()
	for _, evm := range evmChains {
		if c.Equals(evm) {
			return true
		}
	}
	return false
}

// HasRouter returns true if the chain has a router smart contract
// This includes EVM chains and other chains like Stellar
func (c Chain) HasRouter() bool {
	routerChains := GetRouterChains()
	for _, chain := range routerChains {
		if c.Equals(chain) {
			return true
		}
	}
	return false
}

// IsUTXO returns true if given chain is a UTXO chain.
func (c Chain) IsUTXO() bool {
	utxoChains := GetUTXOChains()
	for _, utxo := range utxoChains {
		if c.Equals(utxo) {
			return true
		}
	}
	return false
}

// IsEmpty is to determinate whether the chain is empty
func (c Chain) IsEmpty() bool {
	return strings.TrimSpace(c.String()) == ""
}

// String implement fmt.Stringer
func (c Chain) String() string {
	// convert it to upper case again just in case someone created a ticker via Chain("rune")
	return strings.ToUpper(string(c))
}

// GetSigningAlgo get the signing algorithm for the given chain
func (c Chain) GetSigningAlgo() SigningAlgo {
	// TSS only supports secp256k1 for now, but we can derive ed25519 addresses
	// from secp256k1 keys for chains that need them (like Stellar)
	return SigningAlgoSecp256k1
}

// GetGasAsset return gas asset, only relevant for THORChain
func (c Chain) GetGasAsset() Asset {
	switch c {
	case SWITCHLYChain:
		return SwitchNative
	case BSCChain:
		return BNBBEP20Asset
	case BTCChain:
		return BTCAsset
	case LTCChain:
		return LTCAsset
	case BCHChain:
		return BCHAsset
	case DOGEChain:
		return DOGEAsset
	case ETHChain:
		return ETHAsset
	case AVAXChain:
		return AVAXAsset
	case GAIAChain:
		return ATOMAsset
	case BASEChain:
		return BaseETHAsset
	case XRPChain:
		return XRPAsset
	case StellarChain:
		return XLMAsset
	default:
		return EmptyAsset
	}
}

// GetGasUnits returns name of the gas unit for each chain
func (c Chain) GetGasUnits() string {
	switch c {
	case AVAXChain:
		return "nAVAX"
	case BTCChain:
		return "satsperbyte"
	case BCHChain:
		return "satsperbyte"
	case DOGEChain:
		return "satsperbyte"
	case ETHChain, BSCChain, BASEChain:
		return "gwei"
	case GAIAChain:
		return "uatom"
	case LTCChain:
		return "satsperbyte"
	case XRPChain:
		return "drop"
	case StellarChain:
		return "stroop"
	default:
		return ""
	}
}

// GetGasAssetDecimal returns decimals for the gas asset of the given chain. Currently
// Gaia is 1e6 and all others are 1e8. If an external chain's gas asset is larger than
// 1e8, just return cosmos.DefaultCoinDecimals.
func (c Chain) GetGasAssetDecimal() int64 {
	switch c {
	case GAIAChain:
		return 6
	case XRPChain:
		return 6
	case StellarChain:
		return 7
	default:
		return cosmos.DefaultCoinDecimals
	}
}

// IsValidAddress make sure the address is correct for the chain
// And this also make sure mocknet doesn't use mainnet address vice versa
func (c Chain) IsValidAddress(addr Address) bool {
	network := CurrentChainNetwork
	prefix := c.AddressPrefix(network)
	return strings.HasPrefix(addr.String(), prefix)
}

// AddressPrefix return the address prefix used by the given network (mocknet/mainnet)
func (c Chain) AddressPrefix(cn ChainNetwork) string {
	if c.IsEVM() {
		return "0x"
	}
	switch cn {
	case MockNet:
		switch c {
		case GAIAChain:
			return "cosmos"
		case SWITCHLYChain:
			// TODO update this to use mocknet address prefix
			return types.GetConfig().GetBech32AccountAddrPrefix()
		case BTCChain:
			return chaincfg.RegressionNetParams.Bech32HRPSegwit
		case LTCChain:
			return ltcchaincfg.RegressionNetParams.Bech32HRPSegwit
		case DOGEChain:
			return dogchaincfg.RegressionNetParams.Bech32HRPSegwit
		}
	case MainNet, StageNet:
		switch c {
		case GAIAChain:
			return "cosmos"
		case SWITCHLYChain:
			return types.GetConfig().GetBech32AccountAddrPrefix()
		case BTCChain:
			return chaincfg.MainNetParams.Bech32HRPSegwit
		case LTCChain:
			return ltcchaincfg.MainNetParams.Bech32HRPSegwit
		case DOGEChain:
			return dogchaincfg.MainNetParams.Bech32HRPSegwit
		}
	}
	return ""
}

// DustThreshold returns the min dust threshold for each chain
// The min dust threshold defines the lower end of the withdraw range of memoless savers txs
// The native coin value provided in a memoless tx defines a basis points amount of Withdraw or Add to a savers position as follows:
// Withdraw range: (dust_threshold + 1) -> (dust_threshold + 10_000)
// Add range: dust_threshold -> Inf
// NOTE: these should all be in 8 decimal places
func (c Chain) DustThreshold() cosmos.Uint {
	switch c {
	case BTCChain, LTCChain, BCHChain:
		return cosmos.NewUint(10_000)
	case DOGEChain:
		return cosmos.NewUint(100_000_000)
	case ETHChain, AVAXChain, GAIAChain, BSCChain, BASEChain:
		return cosmos.OneUint()
	case XRPChain:
		// XRP's dust threshold is being set to 1 XRP. This is the base reserve requirement on XRP's ledger.
		// It is set to this value for two reasons:
		//    1. to prevent edge cases of outbound XRP to new addresses where this is the minimum that must be transferred
		//    2. to burn this amount on churns of each XRP vault, effectively leaving it behind as it cannot be transferred, but still transferring all other XRP
		// On churns, we can optionally delete the account to recover an additional .8 XRP, but would increases code complexity and will remove related ledger entries
		// Comparing to BTC, this dust threshold should be reasonable.
		return cosmos.NewUint(One) // 1 XRP
	case StellarChain:
		// XLM's dust threshold is being set to 1 XLM. This is the base reserve requirement on Stellar's ledger.
		// It is set to this value for two reasons:
		//    1. to prevent edge cases of outbound XLM to new addresses where this is the minimum that must be transferred
		//    2. to burn this amount on churns of each XLM vault, effectively leaving it behind as it cannot be transferred, but still transferring all other XLM
		// On churns, we can optionally delete the account to recover an additional 0.5 XLM, but would increases code complexity and will remove related ledger entries
		// Comparing to BTC, this dust threshold should be reasonable.
		return cosmos.NewUint(One) // 1 XLM
	default:
		return cosmos.ZeroUint()
	}
}

// MaxMemoLength returns the max memo length for each chain.
func (c Chain) MaxMemoLength() int {
	switch c {
	case BTCChain, LTCChain, BCHChain, DOGEChain:
		return constants.MaxOpReturnDataSize
	default:
		// Default to the max memo size that we will process, regardless
		// of any higher memo size capable on other chains.
		return constants.MaxMemoSize
	}
}

// DefaultCoinbase returns the default coinbase address for each chain, returns 0 if no
// coinbase emission is used. This is used used at the time of writing as a fallback
// value in Bifrost, and for inbound confirmation count estimates in the quote APIs.
func (c Chain) DefaultCoinbase() float64 {
	switch c {
	case BTCChain:
		return 3.125
	case LTCChain:
		return 6.25
	case BCHChain:
		return 3.125
	case DOGEChain:
		return 10000
	default:
		return 0
	}
}

func (c Chain) ApproximateBlockMilliseconds() int64 {
	switch c {
	case BTCChain:
		return 600_000
	case LTCChain:
		return 150_000
	case BCHChain:
		return 600_000
	case DOGEChain:
		return 60_000
	case ETHChain:
		return 12_000
	case AVAXChain:
		return 3_000
	case BSCChain:
		return 1_500
	case GAIAChain:
		return 6_000
	case SWITCHLYChain:
		return 6_000
	case BASEChain:
		return 2_000
	case XRPChain:
		return 4_000 // approx 3-5 seconds
	case StellarChain:
		return 5_000 // approx 5 seconds
	default:
		return 0
	}
}

func (c Chain) InboundNotes() string {
	switch c {
	case BTCChain, LTCChain, BCHChain, DOGEChain:
		return "First output should be to inbound_address, second output should be change back to self, third output should be OP_RETURN, limited to 80 bytes. Do not send below the dust threshold. Do not use exotic spend scripts, locks or address formats."
	case ETHChain, AVAXChain, BSCChain, BASEChain:
		return "Base Asset: Send the inbound_address the asset with the memo encoded in hex in the data field. Tokens: First approve router to spend tokens from user: asset.approve(router, amount). Then call router.depositWithExpiry(inbound_address, asset, amount, memo, expiry). Asset is the token contract address. Amount should be in native asset decimals (eg 1e18 for most tokens). Do not swap to smart contract addresses."
	case GAIAChain:
		return "Transfer the inbound_address the asset with the memo. Do not use multi-in, multi-out transactions."
	case SWITCHLYChain:
		return "Broadcast a MsgDeposit to the Switchly network with the appropriate memo. Do not use multi-in, multi-out transactions."
	case XRPChain:
		return "Transfer the inbound_address the asset with the memo. Only a single memo is supported and only MemoData is used."
	case StellarChain:
		return "Transfer the inbound_address the asset with the memo. Use MemoText for the memo field. Do not use multi-in, multi-out transactions."
	default:
		return ""
	}
}

func NewChains(raw []string) (Chains, error) {
	var returnErr error
	var chains Chains
	for _, c := range raw {
		chain, err := NewChain(c)
		if err == nil {
			chains = append(chains, chain)
		} else {
			returnErr = multierror.Append(returnErr, err)
		}
	}
	return chains, returnErr
}

// Has check whether chain c is in the list
func (chains Chains) Has(c Chain) bool {
	for _, ch := range chains {
		if ch.Equals(c) {
			return true
		}
	}
	return false
}

// Distinct return a distinct set of chains, no duplicates
func (chains Chains) Distinct() Chains {
	var newChains Chains
	for _, chain := range chains {
		if !newChains.Has(chain) {
			newChains = append(newChains, chain)
		}
	}
	return newChains
}

func (chains Chains) Strings() []string {
	strings := make([]string, len(chains))
	for i, c := range chains {
		strings[i] = c.String()
	}
	return strings
}
