# Stellar Chain Client

This package provides a Stellar blockchain client for THORChain's Bifrost, supporting both native XLM and non-native assets.

## Supported Assets

The Stellar client supports multiple asset types through an asset mapping system:

### Native Assets
- **XLM**: The native Stellar Lumens token

### Non-Native Assets (Credit Assets)
- **USDC**: USD Coin issued by `GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN`

## Asset Mapping System

The asset mapping system (`asset_mapping.go`) provides:

1. **Whitelisting**: Only explicitly defined assets are supported
2. **Decimal Conversion**: Automatic conversion between Stellar decimals and THORChain's 1e8 standard
3. **Bidirectional Mapping**: Convert between Stellar assets and THORChain assets

### Adding New Assets

To add support for a new asset, update the `StellarAssetMappings` slice in `asset_mapping.go`:

```go
{
    StellarAssetType:   "credit_alphanum4", // or "credit_alphanum12" for longer codes
    StellarAssetCode:   "ASSET_CODE",
    StellarAssetIssuer: "ISSUER_ADDRESS",
    StellarDecimals:    7, // Asset decimals on Stellar
    THORChainAsset:     common.Asset{Chain: common.StellarChain, Symbol: "ASSET_CODE-ISSUER_ADDRESS", Ticker: "ASSET_CODE"},
},
```

## Features

### Inbound Transactions
- Monitors Stellar ledgers for payment operations
- Supports both native XLM and whitelisted credit assets
- Converts amounts to THORChain's decimal standard
- Extracts transaction memos for THORChain processing

### Outbound Transactions
- Creates Stellar payment operations for supported assets
- Handles TSS signing for multi-signature transactions
- Supports both native and credit asset transfers
- Proper decimal conversion for each asset type

### Account Balance Queries
- Retrieves balances for all supported assets
- Converts balances to THORChain format
- Filters out unsupported assets

## Example Usage

### Processing a USDC Transaction

```go
// Create a USDC asset
usdcAsset := common.Asset{
    Chain:  common.StellarChain,
    Symbol: "USDC-GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
    Ticker: "USDC",
}

// Create a transaction
coin := common.NewCoin(usdcAsset, cosmos.NewUint(100000000)) // 1 USDC
txOutItem := stypes.TxOutItem{
    Chain:       common.StellarChain,
    ToAddress:   common.Address("DESTINATION_ADDRESS"),
    VaultPubKey: vaultPubKey,
    Coins:       common.Coins{coin},
    Memo:        "swap:BTC.BTC:bc1qaddress",
}

// Process the transaction
payment, err := client.processOutboundTx(txOutItem)
```

## Decimal Handling

- **Stellar**: Most assets use 7 decimal places (10^7)
- **THORChain**: Uses 8 decimal places (10^8) as standard
- **Conversion**: Automatic conversion maintains precision

Example:
- 1 XLM = 10,000,000 stroops (Stellar) = 100,000,000 units (THORChain)
- 1 USDC = 10,000,000 units (Stellar) = 100,000,000 units (THORChain)

## Testing

Run the test suite:

```bash
go test -v ./bifrost/pkg/chainclients/stellar/...
```

The test suite includes:
- Asset mapping functionality tests
- Transaction processing tests
- Balance query tests
- Error handling tests 