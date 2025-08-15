# Packages Breakdown

## How xChainjs is constructed

### xchain-\[chain] clients

Each blockchian that is integrated into SWITCHLYChain has a corresponding xchain client with a suite of functionality to work with that chain. They all extend the `xchain-client` class.

### xchain-switchly-amm

Switchly automatic market maker that uses Switchlynode & Midgard Api's AMM functions like swapping, adding and removing liquidity. It wraps xchain clients and creates a new wallet class and balance collection.

### xchain-switchly-query

Uses midgard and switchlynode Api's to query Switchly for information. This module should be used as the starting place get any SWITCHLYChain information that resides in SWITCHLYNode or Midgard as it does the heaving lifting and configuration.

Default endpoints are provided with redundancy, custom SWITCHLYNode or Midgard endpoints can be provided in the constructor.

### **xchain-midgard**

This package is built from OpenAPI-generator. It is used by the switchly-query.

Switchly-query contains midgard class that uses xchain-midgard and the following end points:

- /v2/switchly/mimir
- /v2/switchly/inbound_addresses
- /v2/switchly/constants
- /v2/switchly/queue

For simplicity, is recommended to use the midgard class within switchly-query instead of using the midgard package directly.

#### Midgard Configuration in switchly-query

Default endpoints `defaultMidgardConfig` are provided with redundancy within the Midgard class.

```typescript
// How switchly-query constructs midgard
const defaultMidgardConfig: Record<Network, MidgardConfig> = {
  mainnet: {
    apiRetries: 3,
    midgardBaseUrls: [
      'https://midgard.ninerealms.com',
      'https://midgard.ninerealms.com',
      'https://midgard.thorswap.net',
    ],
  },
  ...
  export class Midgard {
  private config: MidgardConfig
  readonly network: Network
  private midgardApis: MidgardApi[]

  constructor(network: Network = Network.Mainnet, config?: MidgardConfig) {
    this.network = network
    this.config = config ?? defaultMidgardConfig[this.network]
    axiosRetry(axios, { retries: this.config.apiRetries, retryDelay: axiosRetry.exponentialDelay })
    this.midgardApis = this.config.midgardBaseUrls.map((url) => new MidgardApi(new Configuration({ basePath: url })))
  }
```

Custom Midgard endpoints can be provided in the constructor using the `MidgardConfig` type.

```typescript
// adding custom endpoints
  const network = Network.Mainnet
  const customMidgardConfig: MidgardConfig = {
    apiRetries: 3,
    midgardBaseUrls: [
      'https://midgard.customURL.com',
    ],
  }
  const midgard = new Midgard(network, customMidgardConfig)
}
```

See [ListPools](query-package.md#list-pools) for a working example.

### xchain-switchlynode

This package is built from OpenAPI-generator and is also used by the switchly-query. The design is similar to the midgard. Switchlynode should only be used when time-sensitive data is required else midgard should be used.

```typescript
// How switchly-query constructs switchlynode
const defaultSwitchlynodeConfig: Record<Network, SwitchlynodeConfig> = {
  mainnet: {
    apiRetries: 3,
    switchlynodeBaseUrls: [
      `https://switchlynode.ninerealms.com`,
      `https://switchlynode.thorswap.net`,
      `https://switchlynode.ninerealms.com/`,
    ],
  },
  ...
  export class Switchlynode {
  private config: SwitchlynodeConfig
  private network: Network
 ...
  constructor(network: Network = Network.Mainnet, config?: SwitchlynodeConfig) {
    this.network = network
    this.config = config ?? defaultSwitchlynodeConfig[this.network]
    axiosRetry(axios, { retries: this.config.apiRetries, retryDelay: axiosRetry.exponentialDelay })
    this.transactionsApi = this.config.switchlynodeBaseUrls.map(
      (url) => new TransactionsApi(new Configuration({ basePath: url })),
    )
    this.queueApi = this.config.switchlynodeBaseUrls.map((url) => new QueueApi(new Configuration({ basePath: url })))
    this.networkApi = this.config.switchlynodeBaseUrls.map((url) => new NetworkApi(new Configuration({ basePath: url })))
    this.poolsApi = this.config.switchlynodeBaseUrls.map((url) => new PoolsApi(new Configuration({ basePath: url })))
    this.liquidityProvidersApi = this.config.switchlynodeBaseUrls.map(
      (url) => new LiquidityProvidersApi(new Configuration({ basePath: url })),
    )
  }
```

### Switchlynode Configuration in switchly-query

As with the midgard package, switchlynode can also be given custom end points via the `SwitchlynodeConfig` type.

## **xchain-util**

A helper packager used by all the other packages. It has the following modules:

- `asset` - Utilities for handling assets
- `async` - Utitilies for `async` handling
- `bn` - Utitilies for using `bignumber.js`
- `chain` - Utilities for multi-chain
- `string` - Utilities for strings
