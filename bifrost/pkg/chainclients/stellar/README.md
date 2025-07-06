# Stellar Chain Client

This package provides a comprehensive Stellar blockchain client for THORChain's Bifrost, supporting native XLM, non-native assets, and smart contract router integration.

## Features Overview

### Core Functionality
- **Native XLM Support**: Full support for Stellar Lumens (XLM)
- **Credit Assets**: Support for whitelisted non-native Stellar assets
- **Router Contract Integration**: Smart contract support via Soroban RPC
- **Dual API Support**: Both Horizon API and Soroban RPC integration
- **Event Processing**: Router contract event detection and processing
- **TSS Integration**: Multi-signature transaction support

### Router Contract Support
- **Deposit Events**: Processes router deposit transactions
- **Transfer Out Events**: Monitors outbound router transactions
- **Vault Rotation**: Handles transfer allowance and vault asset returns
- **Event Filtering**: Automatically filters router events from contract calls
- **Graceful Degradation**: Functions with or without Soroban RPC availability

## Architecture

### Block Scanner
The `StellarBlockScanner` processes three types of operations:

1. **Payment Operations**: Standard Stellar payments
2. **InvokeHostFunction Operations**: Smart contract calls (router events)
3. **CreateAccount Operations**: Account creation transactions

### Router Event Processing
```go
// Router events are automatically detected and processed
switch eventType {
case "deposit", "router_deposit":
    // Creates TxInItem for inbound deposits
case "transfer_out", "router_transfer_out":
    // Logs outbound transfers (monitoring only)
case "transfer_allowance", "transferallowance":
    // Handles vault rotation
case "return_vault_assets", "returnvaultassets":
    // Processes vault asset returns
}
```

## Supported Assets

### Native Assets
- **XLM**: The native Stellar Lumens token

### Non-Native Assets (Credit Assets)
- **USDC**: USD Coin issued by `GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN`

### Asset Mapping System

The asset mapping system provides:

1. **Whitelisting**: Only explicitly defined assets are supported
2. **Decimal Conversion**: Automatic conversion between Stellar decimals and THORChain's 1e8 standard
3. **Bidirectional Mapping**: Convert between Stellar assets and THORChain assets
4. **Router Asset Support**: Maps router contract asset addresses to THORChain assets

## Configuration

### Environment Variables
- `XLM_HOST`: Stellar Horizon API endpoint (required)
- `XLM_CHAIN_NETWORK`: Network type (`testnet` or `mainnet`)

### Soroban RPC Configuration
```go
// Soroban RPC client is automatically configured
sorobanRPCClient := NewSorobanRPCClient(cfg, logger, stellarNetwork)

// Falls back to public networks if no RPC host configured
// Testnet: https://soroban-testnet.stellar.org
// Mainnet: https://soroban-mainnet.stellar.org
```

## Transaction Processing

### Inbound Transactions
The scanner processes multiple operation types:

```go
func (c *StellarBlockScanner) processOperation(tx horizon.Transaction, op operations.Operation, height int64) (*types.TxInItem, error) {
    switch operation := op.(type) {
    case operations.Payment:
        return c.processPaymentOperation(tx, operation, height)
    case operations.InvokeHostFunction:
        return c.processInvokeHostFunctionOperation(tx, operation, height)
    case operations.CreateAccount:
        return c.processCreateAccountOperation(tx, operation, height)
    default:
        return nil, nil // Skip unsupported operations
    }
}
```

### Router Contract Events
Router events are automatically detected and converted to witness transactions:

```go
// Example router deposit event processing
txInItem := &types.TxInItem{
    BlockHeight: height,
    Tx:          event.TransactionHash,
    Sender:      event.FromAddress,
    To:          event.ToAddress,
    Coins:       common.Coins{coin},
    Memo:        event.Memo,
    Gas:         common.Gas{{Asset: common.XLMAsset, Amount: cosmos.NewUint(baseFeeStroops)}},
}
```

### Outbound Transactions
- Creates Stellar payment operations for supported assets
- Handles TSS signing for multi-signature transactions
- Supports both native and credit asset transfers
- Proper decimal conversion for each asset type

## Error Handling & Resilience

### Rate Limiting
```go
// Automatic retry with exponential backoff
func (c *StellarBlockScanner) retryHorizonCall(operation string, fn func() error) error {
    for i := 0; i <= maxRetries; i++ {
        err = fn()
        if err == nil {
            return nil
        }
        
        // Handle rate limits with exponential backoff
        if strings.Contains(err.Error(), "Rate Limit Exceeded") {
            delay := time.Duration(1<<uint(i)) * retryDelay
            time.Sleep(delay)
            continue
        }
        break
    }
    return err
}
```

### Graceful Degradation
- **Soroban RPC Optional**: Functions without Soroban RPC if unavailable
- **Asset Filtering**: Skips unsupported assets without errors
- **Event Processing**: Continues processing even if some events fail

## Adding New Assets

To add support for a new asset, update the `StellarAssetMappings` slice in `asset_mapping.go`:

```go
{
    StellarAssetType:   "credit_alphanum4", // or "credit_alphanum12"
    StellarAssetCode:   "ASSET_CODE",
    StellarAssetIssuer: "ISSUER_ADDRESS",
    StellarDecimals:    7, // Asset decimals on Stellar
    THORChainAsset:     common.Asset{
        Chain:  common.StellarChain,
        Symbol: "ASSET_CODE-ISSUER_ADDRESS",
        Ticker: "ASSET_CODE",
    },
},
```

## Router Contract Integration

### Contract Event Types
- `deposit`: User deposits to router contract
- `transfer_out`: Router transfers to users
- `deposit_with_expiry`: Time-limited deposits
- `transfer_allowance`: Vault rotation transfers
- `return_vault_assets`: Vault asset returns

### Event Processing Flow
1. **Detection**: `InvokeHostFunction` operations are scanned
2. **Filtering**: Only router contract addresses are processed
3. **Event Retrieval**: Soroban RPC fetches detailed event data
4. **Conversion**: Events are converted to THORChain `TxInItem` format
5. **Witness Creation**: Transactions are submitted to THORChain for processing

## Decimal Handling

- **Stellar**: Most assets use 7 decimal places (10^7)
- **THORChain**: Uses 8 decimal places (10^8) as standard
- **Conversion**: Automatic conversion maintains precision

Example conversions:
- 1 XLM = 10,000,000 stroops (Stellar) = 100,000,000 units (THORChain)
- 1 USDC = 10,000,000 units (Stellar) = 100,000,000 units (THORChain)

## Testing

Run the complete test suite:

```bash
# Run all Stellar tests
go test -v ./bifrost/pkg/chainclients/stellar/...

# Run specific test suites
go test -v ./bifrost/pkg/chainclients/stellar/ -run TestStellarBlockScanner
go test -v ./bifrost/pkg/chainclients/stellar/ -run TestSorobanRPCClient
```

The test suite includes:
- Asset mapping functionality tests
- Transaction processing tests
- Router event processing tests
- Balance query tests
- Error handling and resilience tests
- Soroban RPC integration tests

