# Introduction

## Overview

THORChain is a decentralised cross-chain liquidity protocol that allows users to add liquidity or swap over that liquidity. It does not peg or wrap assets. Swaps are processed as easily as making a single on-chain transaction.

THORChain works by observing transactions to its vaults across all the chains it supports. When the majority of nodes observe funds flowing into the system, they agree on the user's intent (usually expressed through a [memo](concepts/memos.md) within a transaction) and take the appropriate action.

```admonish info
For more information see [Understanding THORChain](https://docs.thorchain.org/learn/understanding-thorchain) [Technology](https://docs.thorchain.org/how-it-works/technology) or [Concepts](broken-reference).
```

For wallets/interfaces to interact with THORChain, they need to:

1. Connect to THORChain to obtain information from one or more endpoints.
2. Construct transactions with the correct memos.
3. Send the transactions to THORChain Inbound Vaults.

See this [THORChain Development Guide](https://youtu.be/Qowrasst2UQ) video for more information or check out the [Front-end](./#front-end-development-guides) guides below for fast and simple implementation.

## Front-end Development Guides

### [Native Swaps Guide](swap-guide/quickstart-guide.md)

Frontend developers can use THORChain to access decentralised layer1 swaps between BTC, ETH, ATOM and more.

### [Affiliate Guide](affiliate-guide/affiliate-fee-guide.md)

THORChain offers user interfaces affiliate fees up to 10% for using THORChain.

### [Aggregators](aggregators/aggregator-overview.md)

Aggregators can deploy contracts that use custom `swapIn` and `swapOut` cross-chain aggregation to perform swaps before and after THORChain.

Eg, swap from an asset on Sushiswap, then THORChain, then an asset on TraderJoe in one transaction.

### [Concepts](concepts/connecting-to-thorchain.md)

In-depth guides to understand THORChain's implementation have been created.

### [Libraries](concepts/code-libraries.md)

Several libraries exist to allow for rapid integration. [`xchainjs`](https://docs.xchainjs.org/overview/) has seen the most development is recommended.

Eg, swap from layer 1 ETH to BTC and back.

### Analytics

Analysts can build on Midgard or Flipside to access cross-chain metrics and analytics. See [Connecting to THORChain](concepts/connecting-to-thorchain.md "mention") for more information.

### Connecting to THORChain

THORChain has several APIs with Swagger documentation.

- Midgard - [https://midgard.ninerealms.com/v2/doc](https://midgard.ninerealms.com/v2/doc)
- THORNode - [https://thornode.ninerealms.com/thorchain/doc](https://thornode.ninerealms.com/thorchain/doc)
- Cosmos RPC - [https://docs.cosmos.network/v0.50/learn/advanced/grpc_rest](https://docs.cosmos.network/v0.50/learn/advanced/grpc_rest)
- CometBFT RPC - [https://docs.cometbft.com/v0.38/rpc/](https://docs.cometbft.com/v0.38/rpc/)

A list of endpoints operated by Nine Realms is located at [NineRealms THORChain Ops Dashboard](https://ops.ninerealms.com/links).

See [Connecting to THORChain](concepts/connecting-to-thorchain.md "mention") for more information.

### Support and Questions

Join the [THORChain Dev Discord](https://discord.gg/7RRmc35UEG) for any questions or assistance.
