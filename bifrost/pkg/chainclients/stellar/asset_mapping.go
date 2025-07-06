package stellar

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/stellar/go/txnbuild"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

// StellarNetwork represents the network type
type StellarNetwork string

const (
	StellarMainnet StellarNetwork = "mainnet"
	StellarTestnet StellarNetwork = "testnet"
)

// StellarAssetMapping maps Stellar assets to THORChain assets
type StellarAssetMapping struct {
	// Stellar asset identification
	StellarAssetType   string // "native", "credit_alphanum4", "credit_alphanum12", "contract"
	StellarAssetCode   string // Asset code (e.g., "USDC", "XLM" for native)
	StellarAssetIssuer string // Issuer address for classic assets, contract address for Soroban tokens
	StellarDecimals    int    // Decimal precision

	// Network-specific contract addresses for Soroban tokens
	ContractAddresses map[StellarNetwork]string // Network -> Contract Address mapping

	// THORChain asset representation
	THORChainAsset common.Asset
}

// stellarAssetMappings contains the mapping of known Stellar assets to THORChain assets
var stellarAssetMappings = []StellarAssetMapping{

	// XLM SEP-41 Token
	{
		StellarAssetType:   "native",
		StellarAssetCode:   "XLM",
		StellarAssetIssuer: "",
		StellarDecimals:    7,
		ContractAddresses: map[StellarNetwork]string{
			StellarMainnet: "CAS3J7GYLGXMF6TDJBBYYSE3HQ6BBSMLNUQ34T6TZMYMW2EVH34XOWMA",
			StellarTestnet: "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC",
		},
		THORChainAsset: common.Asset{Chain: common.StellarChain, Symbol: "XLM", Ticker: "XLM"},
	},

	// USDC SEP-41 Token
	{
		StellarAssetType:   "contract",
		StellarAssetCode:   "USDC",
		StellarAssetIssuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
		StellarDecimals:    7,
		ContractAddresses: map[StellarNetwork]string{
			StellarMainnet: "CCW67TSZV3SSS2HXMBQ5JFGCKJNXKZM7UQUWUZPUTHXSTZLEO7SJMI75",
			StellarTestnet: "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA",
		},
		THORChainAsset: common.Asset{Chain: common.StellarChain, Symbol: "USDC", Ticker: "USDC"},
	},
}

// currentNetwork holds the current network configuration
var currentNetwork StellarNetwork = StellarTestnet // Default to testnet for safety

// SetNetwork sets the current Stellar network
func SetNetwork(network StellarNetwork) {
	currentNetwork = network
	// Update contract addresses in mappings based on network
	updateMappingsForNetwork(network)
}

// GetCurrentNetwork returns the current network
func GetCurrentNetwork() StellarNetwork {
	return currentNetwork
}

// updateMappingsForNetwork updates the StellarAssetIssuer field for contract assets based on the current network
func updateMappingsForNetwork(network StellarNetwork) {
	for i := range stellarAssetMappings {
		mapping := &stellarAssetMappings[i]
		if mapping.StellarAssetType == "contract" {
			if contractAddr, exists := mapping.ContractAddresses[network]; exists {
				mapping.StellarAssetIssuer = contractAddr
			}
		}
	}
}

// GetAssetByStellarAsset finds the asset mapping by Stellar asset parameters
func GetAssetByStellarAsset(assetType, assetCode, assetIssuer string) (StellarAssetMapping, bool) {
	for _, mapping := range stellarAssetMappings {
		if mapping.StellarAssetType == assetType &&
			mapping.StellarAssetCode == assetCode &&
			mapping.StellarAssetIssuer == assetIssuer {
			return mapping, true
		}
	}
	return StellarAssetMapping{}, false
}

// GetAssetByTHORChainAsset finds the asset mapping by THORChain asset
func GetAssetByTHORChainAsset(thorAsset common.Asset) (StellarAssetMapping, bool) {
	for _, mapping := range stellarAssetMappings {
		if mapping.THORChainAsset.Equals(thorAsset) {
			return mapping, true
		}
	}
	return StellarAssetMapping{}, false
}

// GetAssetByAddress finds the asset mapping by various address formats
// This function handles:
// 1. "native" for XLM
// 2. Contract addresses for Soroban tokens (SEP-41 compliant) - network-aware
// 3. "ASSETCODE:ISSUER" for classic assets
// 4. Just the issuer/contract address
func GetAssetByAddress(address string) (StellarAssetMapping, bool) {
	// Handle native XLM asset
	if address == "native" {
		return GetAssetByStellarAsset("native", "XLM", "")
	}

	// Check if it's a Soroban contract address (starts with 'C' and is 56 chars)
	if len(address) == 56 && strings.HasPrefix(address, "C") {
		// Look for contract token by current network's contract address
		for _, mapping := range stellarAssetMappings {
			if mapping.StellarAssetType == "contract" {
				// Check if this address matches the current network's contract address
				if currentAddr, exists := mapping.ContractAddresses[currentNetwork]; exists && currentAddr == address {
					// Return a copy with the correct issuer set for current network
					networkMapping := mapping
					networkMapping.StellarAssetIssuer = address
					return networkMapping, true
				}
				// Also check other networks in case we're looking up a different network's address
				for _, contractAddr := range mapping.ContractAddresses {
					if contractAddr == address {
						// Return a copy with the correct issuer set
						networkMapping := mapping
						networkMapping.StellarAssetIssuer = address
						return networkMapping, true
					}
				}
			}
		}
	}

	// Try to parse as classic asset format (ASSETCODE:ISSUER)
	if strings.Contains(address, ":") {
		parts := strings.Split(address, ":")
		if len(parts) == 2 {
			assetCode := parts[0]
			assetIssuer := parts[1]

			// Try credit_alphanum4 first
			if mapping, found := GetAssetByStellarAsset("credit_alphanum4", assetCode, assetIssuer); found {
				return mapping, true
			}

			// Try credit_alphanum12
			if mapping, found := GetAssetByStellarAsset("credit_alphanum12", assetCode, assetIssuer); found {
				return mapping, true
			}
		}
	}

	// Try to match by issuer address only (for classic assets)
	for _, mapping := range stellarAssetMappings {
		if mapping.StellarAssetIssuer == address {
			return mapping, true
		}
	}

	return StellarAssetMapping{}, false
}

// GetAssetByAddressAndNetwork finds the asset mapping by address for a specific network
func GetAssetByAddressAndNetwork(address string, network StellarNetwork) (StellarAssetMapping, bool) {
	// Handle native XLM asset
	if address == "native" {
		return GetAssetByStellarAsset("native", "XLM", "")
	}

	// Check if it's a Soroban contract address for the specified network
	if len(address) == 56 && strings.HasPrefix(address, "C") {
		for _, mapping := range stellarAssetMappings {
			if mapping.StellarAssetType == "contract" {
				if contractAddr, exists := mapping.ContractAddresses[network]; exists && contractAddr == address {
					// Return a copy with the correct issuer set for the specified network
					networkMapping := mapping
					networkMapping.StellarAssetIssuer = address
					return networkMapping, true
				}
			}
		}
	}

	// Fall back to regular address lookup for classic assets
	return GetAssetByAddress(address)
}

// GetContractAddressForNetwork returns the contract address for a given asset on a specific network
func GetContractAddressForNetwork(assetCode string, network StellarNetwork) (string, bool) {
	for _, mapping := range stellarAssetMappings {
		if mapping.StellarAssetType == "contract" && mapping.StellarAssetCode == assetCode {
			if contractAddr, exists := mapping.ContractAddresses[network]; exists {
				return contractAddr, true
			}
		}
	}
	return "", false
}

// GetAllNetworksForAsset returns all networks where a contract asset is deployed
func GetAllNetworksForAsset(assetCode string) []StellarNetwork {
	var networks []StellarNetwork
	for _, mapping := range stellarAssetMappings {
		if mapping.StellarAssetType == "contract" && mapping.StellarAssetCode == assetCode {
			for network := range mapping.ContractAddresses {
				networks = append(networks, network)
			}
			break
		}
	}
	return networks
}

// IsAssetAvailableOnNetwork checks if an asset is available on a specific network
func IsAssetAvailableOnNetwork(assetCode string, network StellarNetwork) bool {
	for _, mapping := range stellarAssetMappings {
		if mapping.StellarAssetCode == assetCode {
			// Native and classic assets are available on all networks
			if mapping.StellarAssetType == "native" || IsClassicAsset(mapping) {
				return true
			}
			// Contract assets need to check network-specific addresses
			if mapping.StellarAssetType == "contract" {
				_, exists := mapping.ContractAddresses[network]
				return exists
			}
		}
	}
	return false
}

// AddAssetMapping adds a new asset mapping (useful for dynamic asset registration)
func AddAssetMapping(mapping StellarAssetMapping) {
	stellarAssetMappings = append(stellarAssetMappings, mapping)
	// Update the issuer for contract assets based on current network
	if mapping.StellarAssetType == "contract" {
		updateMappingsForNetwork(currentNetwork)
	}
}

// UpdateContractAddress updates the contract address for an asset on a specific network
func UpdateContractAddress(assetCode string, network StellarNetwork, contractAddress string) bool {
	for i := range stellarAssetMappings {
		mapping := &stellarAssetMappings[i]
		if mapping.StellarAssetType == "contract" && mapping.StellarAssetCode == assetCode {
			if mapping.ContractAddresses == nil {
				mapping.ContractAddresses = make(map[StellarNetwork]string)
			}
			mapping.ContractAddresses[network] = contractAddress
			// Update issuer if this is the current network
			if network == currentNetwork {
				mapping.StellarAssetIssuer = contractAddress
			}
			return true
		}
	}
	return false
}

// GetAllAssetMappings returns all configured asset mappings
func GetAllAssetMappings() []StellarAssetMapping {
	return stellarAssetMappings
}

// IsNativeAsset checks if the given asset is the native XLM asset
func IsNativeAsset(mapping StellarAssetMapping) bool {
	return mapping.StellarAssetType == "native"
}

// IsSorobanToken checks if the given asset is a Soroban token contract
func IsSorobanToken(mapping StellarAssetMapping) bool {
	return mapping.StellarAssetType == "contract"
}

// IsClassicAsset checks if the given asset is a classic Stellar asset
func IsClassicAsset(mapping StellarAssetMapping) bool {
	return mapping.StellarAssetType == "credit_alphanum4" || mapping.StellarAssetType == "credit_alphanum12"
}

// GetTokenContractAddress returns the contract address for Soroban tokens
func GetTokenContractAddress(mapping StellarAssetMapping) string {
	if IsSorobanToken(mapping) {
		return mapping.StellarAssetIssuer
	}
	return ""
}

// ToStellarAsset converts a THORChain asset to a Stellar txnbuild.Asset
func (s StellarAssetMapping) ToStellarAsset() txnbuild.Asset {
	if s.StellarAssetType == "native" {
		return txnbuild.NativeAsset{}
	}

	return txnbuild.CreditAsset{
		Code:   s.StellarAssetCode,
		Issuer: s.StellarAssetIssuer,
	}
}

// FromStellarAsset creates a StellarAssetMapping from a Stellar asset
func FromStellarAsset(asset txnbuild.Asset) (StellarAssetMapping, error) {
	switch a := asset.(type) {
	case txnbuild.NativeAsset:
		mapping, found := GetAssetByStellarAsset("native", "XLM", "")
		if !found {
			return StellarAssetMapping{}, fmt.Errorf("native asset not found in mapping")
		}
		return mapping, nil
	case txnbuild.CreditAsset:
		// First try to find as contract asset (Soroban token)
		mapping, found := GetAssetByStellarAsset("contract", a.Code, a.Issuer)
		if found {
			return mapping, nil
		}
		// Try credit_alphanum4 for classic assets
		mapping, found = GetAssetByStellarAsset("credit_alphanum4", a.Code, a.Issuer)
		if !found {
			// Try credit_alphanum12 for longer asset codes
			mapping, found = GetAssetByStellarAsset("credit_alphanum12", a.Code, a.Issuer)
		}
		if found {
			return mapping, nil
		}
		return StellarAssetMapping{}, fmt.Errorf("unsupported asset: %s:%s", a.Code, a.Issuer)
	default:
		return StellarAssetMapping{}, fmt.Errorf("unknown asset type: %T", asset)
	}
}

// ConvertToTHORChainAmount converts a Stellar amount to THORChain amount
func (s StellarAssetMapping) ConvertToTHORChainAmount(stellarAmount string) (common.Coin, error) {
	// Parse the stellar amount
	stellarAmountBig, ok := new(big.Int).SetString(stellarAmount, 10)
	if !ok {
		return common.Coin{}, fmt.Errorf("invalid stellar amount: %s", stellarAmount)
	}

	// Convert based on decimal difference
	// THORChain uses 8 decimals, Stellar assets can have different decimals
	thorchainDecimals := 8
	decimalDiff := thorchainDecimals - s.StellarDecimals

	var thorchainAmount *big.Int
	if decimalDiff > 0 {
		// THORChain has more decimals, multiply
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalDiff)), nil)
		thorchainAmount = new(big.Int).Mul(stellarAmountBig, multiplier)
	} else if decimalDiff < 0 {
		// THORChain has fewer decimals, divide
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-decimalDiff)), nil)
		thorchainAmount = new(big.Int).Div(stellarAmountBig, divisor)
	} else {
		// Same decimals, no conversion needed
		thorchainAmount = stellarAmountBig
	}

	// Convert to cosmos.Uint
	thorchainAmountCosmos := cosmos.NewUintFromString(thorchainAmount.String())

	return common.Coin{
		Asset:  s.THORChainAsset,
		Amount: thorchainAmountCosmos,
	}, nil
}

// ConvertFromTHORChainAmount converts a THORChain amount to Stellar amount
func (s StellarAssetMapping) ConvertFromTHORChainAmount(thorchainAmount cosmos.Uint) string {
	// Convert based on decimal difference
	// THORChain uses 8 decimals, Stellar assets can have different decimals
	thorchainDecimals := 8
	decimalDiff := thorchainDecimals - s.StellarDecimals

	thorchainAmountBig := thorchainAmount.BigInt()
	var stellarAmount *big.Int

	if decimalDiff > 0 {
		// THORChain has more decimals, divide
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimalDiff)), nil)
		stellarAmount = new(big.Int).Div(thorchainAmountBig, divisor)
	} else if decimalDiff < 0 {
		// THORChain has fewer decimals, multiply
		multiplier := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-decimalDiff)), nil)
		stellarAmount = new(big.Int).Mul(thorchainAmountBig, multiplier)
	} else {
		// Same decimals, no conversion needed
		stellarAmount = thorchainAmountBig
	}

	return stellarAmount.String()
}
