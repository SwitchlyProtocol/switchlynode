# Stellar Client for SwitchlyProtocol

This package provides a comprehensive Stellar blockchain client for SwitchlyProtocol's Bifrost, supporting native XLM, non-native assets, and smart contract router integration.

## Overview

The Stellar client enables SwitchlyProtocol to:
- Monitor Stellar blockchain for incoming transactions
- Send outbound transactions to Stellar network  
- Support native XLM and Stellar-issued assets
- Handle router contract interactions for advanced DeFi operations
- Maintain accurate balance tracking and solvency reporting

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   SwitchlyNode  │    │   Bifrost        │    │   Stellar       │
│                 │◄──►│   Stellar Client │◄──►│   Network       │
│                 │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Features

### Asset Support

1. **Native XLM**: Full support for Stellar's native lumens
2. **Stellar Assets**: Support for assets issued on Stellar network
3. **Asset Mapping**: Automatic conversion between Stellar and SwitchlyProtocol asset formats

### Transaction Processing

1. **Inbound Monitoring**: Real-time scanning of Stellar blocks for relevant transactions
2. **Decimal Conversion**: Automatic conversion between Stellar decimals and SwitchlyProtocol's 1e8 standard
3. **Bidirectional Mapping**: Convert between Stellar assets and SwitchlyProtocol assets
4. **Router Asset Support**: Maps router contract asset addresses to SwitchlyProtocol assets

## Asset Mapping

The client uses a comprehensive asset mapping system to handle conversions between Stellar and SwitchlyProtocol formats.

### Example Mapping Configuration

```go
var StellarAssetMappings = []StellarAssetMapping{
    {
        StellarAsset: StellarAsset{
            Code:   "XLM",
            Issuer: "",  // Native asset
        },
        SwitchlyProtocolAsset: common.Asset{
            Chain:  common.XLMChain,
            Symbol: "XLM",
            Ticker: "XLM",
        },
        StellarDecimals: 7,
    },
    {
        StellarAsset: StellarAsset{
            Code:   "USDC",
            Issuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
        },
        SwitchlyProtocolAsset: common.Asset{
            Chain:  common.XLMChain, 
            Symbol: "USDC",
            Ticker: "USDC-GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
        },
        StellarDecimals: 7,
    },
}
```

### Router Asset Mapping

For router contracts, assets are mapped using their contract addresses:

```go
SwitchlyProtocolAsset:     common.Asset{
    Chain:  common.XLMChain,
    Symbol: "USDC",
    Ticker: "USDC",
},
RouterAssetAddress: "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQAHHAGCN",
```

## Transaction Flow

### Inbound Transactions

1. **Block Scanning**: Monitor Stellar blockchain for new blocks
2. **Transaction Filtering**: Identify transactions relevant to SwitchlyProtocol vaults
3. **Asset Conversion**: Convert Stellar amounts to SwitchlyProtocol format
4. **Conversion**: Events are converted to SwitchlyProtocol `TxInItem` format
5. **Witness Creation**: Transactions are submitted to SwitchlyProtocol for processing

### Decimal Handling

The client handles decimal conversion between different standards:

- **SwitchlyProtocol**: Uses 8 decimal places (10^8) as standard
- **Stellar**: Uses 7 decimal places (10^7) for most assets
- **Router Contracts**: May use different decimal standards per asset

Examples:
- 1 XLM = 10,000,000 stroops (Stellar) = 100,000,000 units (SwitchlyProtocol)
- 1 USDC = 10,000,000 units (Stellar) = 100,000,000 units (SwitchlyProtocol)

### Outbound Transactions

1. **Transaction Building**: Construct Stellar transactions from SwitchlyProtocol TxOut items
2. **Asset Conversion**: Convert SwitchlyProtocol amounts back to Stellar format
3. **Signing**: Use TSS (Threshold Signature Scheme) for secure transaction signing
4. **Broadcasting**: Submit signed transactions to Stellar network
5. **Confirmation**: Monitor for transaction confirmation and report back to SwitchlyProtocol

## Configuration

### Key Configuration Parameters

```go
type StellarClientConfig struct {
    ChainID              common.Chain
    ChainHost            string           // Stellar Horizon API endpoint
    ChainNetwork         string           // "testnet" or "mainnet" 
    BlockScanner         BlockScannerConfig
    SignerName           string
    SignerPasswd         string
}
```

### Environment Variables

- `STELLAR_HOST`: Stellar Horizon API endpoint
- `STELLAR_NETWORK`: Network to connect to (testnet/mainnet)
- `STELLAR_START_BLOCK_HEIGHT`: Block height to start scanning from

## Error Handling

The client implements comprehensive error handling for:

- Network connectivity issues
- Invalid transaction formats  
- Asset mapping failures
- Signing failures
- Broadcast failures

## Testing

The package includes comprehensive tests covering:

- Asset mapping functionality
- Transaction parsing and conversion
- Decimal conversion accuracy
- Error handling scenarios
- Integration with SwitchlyProtocol

Run tests with:
```bash
go test ./bifrost/pkg/chainclients/stellar/...
```

## Security Considerations

- All private keys are managed through TSS (Threshold Signature Scheme)
- Transaction amounts are validated before signing
- Asset mappings are verified against whitelist
- Network requests include proper timeout and retry logic

