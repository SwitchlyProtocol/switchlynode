package common

import (
	"fmt"
	"regexp"
	"strings"

	xrp "github.com/Peersyst/xrpl-go/address-codec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/bech32"
	sdk "github.com/cosmos/cosmos-sdk/types"
	dogchaincfg "github.com/eager7/dogd/chaincfg"
	"github.com/eager7/dogutil"
	eth "github.com/ethereum/go-ethereum/common"
	bchchaincfg "github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	ltcchaincfg "github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcutil"
	"github.com/stellar/go/strkey"

	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type Address string

const (
	NoAddress       = Address("")
	NoopAddress     = Address("noop")
	EVMNullAddress  = Address("0x0000000000000000000000000000000000000000")
	GaiaZeroAddress = Address("cosmos100000000000000000000000000000000708mjz")
)

var alphaNumRegex = regexp.MustCompile("^[:A-Za-z0-9]*$")

// NewAddress create a new Address. Supports ETH/bech2/BTC/LTC/BCH/DOGE/XRP/XLM.
func NewAddress(address string) (Address, error) {
	if len(address) == 0 {
		return NoAddress, nil
	}

	if !alphaNumRegex.MatchString(address) {
		return NoAddress, fmt.Errorf("address format not supported: %s", address)
	}

	// Check is eth address
	if eth.IsHexAddress(address) {
		return Address(address), nil
	}

	// Check bech32 addresses, would succeed any string bech32 encoded (e.g. GAIA)
	_, _, err := bech32.Decode(address)
	if err == nil {
		return Address(address), nil
	}

	// Check is xrp address
	if IsValidXRPAddress(address) {
		return Address(address), nil
	}

	// Check is xlm address
	if IsValidXLMAddress(address) {
		return Address(address), nil
	}

	// Network-specific (with build tags) address checking.
	return newAddress(address)
}

func IsValidXRPAddress(address string) bool {
	// checks checksum and returns prefix (1 byte, 0x00) + account id (20 bytes)
	decoded, err := xrp.Base58CheckDecode(address)
	if err != nil {
		return false
	}

	return len(decoded) == 21 && decoded[0] == 0x00
}

func IsValidXLMAddress(address string) bool {
	// Validate Stellar address using strkey
	// Account addresses start with 'G' and use Ed25519 public key format
	if strkey.IsValidEd25519PublicKey(address) {
		return true
	}

	// Contract IDs start with 'C' and are 56 characters long
	// We'll do a basic validation since strkey doesn't have IsValidContract
	if len(address) == 56 && strings.HasPrefix(address, "C") {
		// Check if it's valid base32 encoding (A-Z, 2-7)
		for _, char := range address[1:] {
			if !((char >= 'A' && char <= 'Z') || (char >= '2' && char <= '7')) {
				return false
			}
		}
		return true
	}

	return false
}

// IsValidBCHAddress determinate whether the address is a valid new BCH address format
func (addr Address) IsValidBCHAddress() bool {
	// Check mainnet other formats
	bchAddr, err := bchutil.DecodeAddress(addr.String(), &bchchaincfg.MainNetParams)
	if err == nil {
		switch bchAddr.(type) {
		case *bchutil.LegacyAddressPubKeyHash, *bchutil.LegacyAddressScriptHash:
			return false
		}
		return true
	}
	bchAddr, err = bchutil.DecodeAddress(addr.String(), &bchchaincfg.TestNet3Params)
	if err == nil {
		switch bchAddr.(type) {
		case *bchutil.LegacyAddressPubKeyHash, *bchutil.LegacyAddressScriptHash:
			return false
		}
		return true
	}
	bchAddr, err = bchutil.DecodeAddress(addr.String(), &bchchaincfg.RegressionNetParams)
	if err == nil {
		switch bchAddr.(type) {
		case *bchutil.LegacyAddressPubKeyHash, *bchutil.LegacyAddressScriptHash:
			return false
		}
		return true
	}
	return false
}

// ConvertToNewBCHAddressFormat convert the given BCH to new address format
func ConvertToNewBCHAddressFormat(addr Address) (Address, error) {
	if !addr.IsChain(BCHChain) {
		return NoAddress, fmt.Errorf("address(%s) is not BCH chain", addr)
	}
	network := CurrentChainNetwork
	var param *bchchaincfg.Params
	switch network {
	case MockNet:
		param = &bchchaincfg.RegressionNetParams
	case MainNet:
		param = &bchchaincfg.MainNetParams
	case StageNet:
		param = &bchchaincfg.MainNetParams
	}
	bchAddr, err := bchutil.DecodeAddress(addr.String(), param)
	if err != nil {
		return NoAddress, fmt.Errorf("fail to decode address(%s), %w", addr, err)
	}
	return getBCHAddress(bchAddr, param)
}

func getBCHAddress(address bchutil.Address, cfg *bchchaincfg.Params) (Address, error) {
	switch address.(type) {
	case *bchutil.LegacyAddressPubKeyHash, *bchutil.AddressPubKeyHash:
		h, err := bchutil.NewAddressPubKeyHash(address.ScriptAddress(), cfg)
		if err != nil {
			return NoAddress, fmt.Errorf("fail to convert to new pubkey hash address: %w", err)
		}
		return NewAddress(h.String())
	case *bchutil.LegacyAddressScriptHash, *bchutil.AddressScriptHash:
		h, err := bchutil.NewAddressScriptHash(address.ScriptAddress(), cfg)
		if err != nil {
			return NoAddress, fmt.Errorf("fail to convert to new address script hash address: %w", err)
		}
		return NewAddress(h.String())
	}
	return NoAddress, fmt.Errorf("invalid address type")
}

// Note that this can have false positives, such as being unable to distinguish between ETH and AVAX.
func (addr Address) IsChain(chain Chain) bool {
	if chain.IsEVM() {
		return strings.HasPrefix(addr.String(), "0x")
	}
	switch chain {
	case XRPChain:
		return IsValidXRPAddress(addr.String())
	case StellarChain:
		return IsValidXLMAddress(addr.String())
	case GAIAChain:
		// Note: Gaia does not use a special prefix for testnet
		prefix, _, _ := bech32.Decode(addr.String())
		return prefix == "cosmos"
	case SWITCHLYChain:
		prefix, _, _ := bech32.Decode(addr.String())
		return prefix == "switch" || prefix == "sswitch" || prefix == "tswitch" ||
			prefix == "switchvaloper" || prefix == "sswitchvaloper" || prefix == "tswitchvaloper"
	case BTCChain:
		prefix, _, err := bech32.Decode(addr.String())
		if err == nil && (prefix == "bc" || prefix == "tb") {
			return true
		}
		// Check mainnet other formats
		_, err = btcutil.DecodeAddress(addr.String(), &chaincfg.MainNetParams)
		if err == nil {
			return true
		}
		// Check testnet other formats
		_, err = btcutil.DecodeAddress(addr.String(), &chaincfg.TestNet3Params)
		if err == nil {
			return true
		}
		return false
	case LTCChain:
		prefix, _, err := bech32.Decode(addr.String())
		if err == nil && (prefix == "ltc" || prefix == "tltc" || prefix == "rltc") {
			return true
		}
		// Check mainnet other formats
		_, err = ltcutil.DecodeAddress(addr.String(), &ltcchaincfg.MainNetParams)
		if err == nil {
			return true
		}
		// Check testnet other formats
		_, err = ltcutil.DecodeAddress(addr.String(), &ltcchaincfg.TestNet4Params)
		if err == nil {
			return true
		}
		return false
	case BCHChain:
		// Check mainnet other formats
		_, err := bchutil.DecodeAddress(addr.String(), &bchchaincfg.MainNetParams)
		if err == nil {
			return true
		}
		// Check testnet other formats
		_, err = bchutil.DecodeAddress(addr.String(), &bchchaincfg.TestNet3Params)
		if err == nil {
			return true
		}
		// Check mocknet / regression other formats
		_, err = bchutil.DecodeAddress(addr.String(), &bchchaincfg.RegressionNetParams)
		if err == nil {
			return true
		}
		return false
	case DOGEChain:
		// Check mainnet other formats
		_, err := dogutil.DecodeAddress(addr.String(), &dogchaincfg.MainNetParams)
		if err == nil {
			return true
		}
		// Check testnet other formats
		_, err = dogutil.DecodeAddress(addr.String(), &dogchaincfg.TestNet3Params)
		if err == nil {
			return true
		}
		// Check mocknet / regression other formats
		_, err = dogutil.DecodeAddress(addr.String(), &dogchaincfg.RegressionNetParams)
		if err == nil {
			return true
		}
		return false
	default:
		return true // if SwitchlyNode don't specifically check a chain yet, assume its ok.
	}
	return false
}

// Note that this will always return ETHChain for an AVAXChain address,
// so perhaps only use it when determining a network (e.g. mainnet/testnet).
func (addr Address) GetChain() Chain {
	for _, chain := range []Chain{ETHChain, SWITCHLYChain, BTCChain, LTCChain, BCHChain, DOGEChain, GAIAChain, AVAXChain, XRPChain, StellarChain} {
		if addr.IsChain(chain) {
			return chain
		}
	}
	return EmptyChain
}

func (addr Address) GetNetwork(chain Chain) ChainNetwork {
	currentNetwork := CurrentChainNetwork
	mainNetPredicate := func() ChainNetwork {
		if currentNetwork == StageNet {
			return StageNet
		}
		return MainNet
	}
	// EVM addresses don't have different prefixes per network
	if chain.IsEVM() {
		return currentNetwork
	}
	switch chain {
	case SWITCHLYChain:
		prefix, _, _ := bech32.Decode(addr.String())
		if strings.EqualFold(prefix, "switch") || strings.EqualFold(prefix, "switchvaloper") {
			return mainNetPredicate()
		}
		if strings.EqualFold(prefix, "tswitch") || strings.EqualFold(prefix, "tswitchvaloper") {
			return MockNet
		}
		if strings.EqualFold(prefix, "sswitch") || strings.EqualFold(prefix, "sswitchvaloper") {
			return StageNet
		}
	case BTCChain:
		prefix, _, _ := bech32.Decode(addr.String())
		switch prefix {
		case "bc":
			return mainNetPredicate()
		case "bcrt", "tb":
			return MockNet
		default:
			_, err := btcutil.DecodeAddress(addr.String(), &chaincfg.MainNetParams)
			if err == nil {
				return mainNetPredicate()
			}
			_, err = btcutil.DecodeAddress(addr.String(), &chaincfg.TestNet3Params)
			if err == nil {
				return MockNet
			}
			_, err = btcutil.DecodeAddress(addr.String(), &chaincfg.RegressionNetParams)
			if err == nil {
				return MockNet
			}
		}
	case LTCChain:
		prefix, _, _ := bech32.Decode(addr.String())
		switch prefix {
		case "ltc":
			return mainNetPredicate()
		case "rltc", "tltc":
			return MockNet
		default:
			_, err := ltcutil.DecodeAddress(addr.String(), &ltcchaincfg.MainNetParams)
			if err == nil {
				return mainNetPredicate()
			}
			_, err = ltcutil.DecodeAddress(addr.String(), &ltcchaincfg.TestNet4Params)
			if err == nil {
				return MockNet
			}
			_, err = ltcutil.DecodeAddress(addr.String(), &ltcchaincfg.RegressionNetParams)
			if err == nil {
				return MockNet
			}
		}
	case BCHChain:
		// Check mainnet other formats
		_, err := bchutil.DecodeAddress(addr.String(), &bchchaincfg.MainNetParams)
		if err == nil {
			return mainNetPredicate()
		}
		// Check testnet other formats
		_, err = bchutil.DecodeAddress(addr.String(), &bchchaincfg.TestNet3Params)
		if err == nil {
			return MockNet
		}
		// Check mocknet / regression other formats
		_, err = bchutil.DecodeAddress(addr.String(), &bchchaincfg.RegressionNetParams)
		if err == nil {
			return MockNet
		}
	case DOGEChain:
		// Check mainnet other formats
		_, err := dogutil.DecodeAddress(addr.String(), &dogchaincfg.MainNetParams)
		if err == nil {
			return mainNetPredicate()
		}
		// Check testnet other formats
		_, err = dogutil.DecodeAddress(addr.String(), &dogchaincfg.TestNet3Params)
		if err == nil {
			return MockNet
		}
		// Check mocknet / regression other formats
		_, err = dogutil.DecodeAddress(addr.String(), &dogchaincfg.RegressionNetParams)
		if err == nil {
			return MockNet
		}
	case StellarChain:
		// Stellar addresses don't have different formats per network
		return currentNetwork
	}
	return currentNetwork
}

func (addr Address) AccAddress() (cosmos.AccAddress, error) {
	return cosmos.AccAddressFromBech32(addr.String())
}

func (addr Address) Equals(addr2 Address) bool {
	return strings.EqualFold(addr.String(), addr2.String())
}

func (addr Address) IsEmpty() bool {
	return strings.TrimSpace(addr.String()) == ""
}

func (addr Address) IsNoop() bool {
	return addr.Equals(NoopAddress)
}

func (addr Address) String() string {
	return string(addr)
}

func (addr Address) MappedAccAddress() (cosmos.AccAddress, error) {
	// TODO: Add support to map EVM addresses -> bech32.
	// Will require new PubKey type to validate.
	_, data, err := bech32.Decode(addr.String())
	if err != nil {
		return nil, err
	}
	encoded, err := bech32.Encode(sdk.GetConfig().GetBech32AccountAddrPrefix(), data)
	if err != nil {
		return nil, err
	}

	return cosmos.AccAddressFromBech32(encoded)
}
