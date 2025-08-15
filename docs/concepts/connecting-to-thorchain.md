# Connecting to SWITCHLYChain

This guide helps developers connect to SWITCHLYChain’s network for querying data, building wallets, dashboards, or debugging network issues. SWITCHLYChain provides four primary data sources:

- **Midgard**: Consumer data for swaps, pools, and volume, ideal for dashboards.
- **SWITCHLYNode**: Raw blockchain data for SWITCHLYChain-specific queries, used by wallets and explorers.
- **Cosmos RPC**: Generic Cosmos SDK data, such as balances and transactions.
- **Tendermint RPC**: Consensus and node status data for monitoring.

```admonish info
The below endpoints are run by specific organisations for public use. There is a cost to running these services. If you want to run your own full node, please see [https://docs.switchly.org/switchlynodes/overview.](https://docs.switchly.org/switchlynodes/overview)
A list of endpoints operated by Nine Realms is located at [NineRealms SWITCHLYChain Ops Dashboard](https://ops.ninerealms.com/links)
```

## Quick Reference

| Source             | Purpose                            | Mainnet URLs                                                                                                                                               | Stagenet URLs                         |
| ------------------ | ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------- |
| **Midgard**        | Swap, pool, volume, and user data  | \- midgard.thorswap.net<br>- midgard.ninerealms.com<br>- midgard.switchly.liquify.com                                                                     | \- stagenet-midgard.ninerealms.com    |
| **SWITCHLYNode**       | SWITCHLYChain-specific blockchain data | \- switchlynode.thorswap.net<br>- switchlynode.ninerealms.com<br>- switchlynode.switchly.liquify.com<br>- Pre-hard-fork (blocks ≤4786559): switchlynode-v0.ninerealms.com | \- stagenet-switchlynode.ninerealms.com   |
| **Cosmos RPC**     | Cosmos SDK data (e.g., balances)   | \- Example: switchlynode.ninerealms.com                                                                                                                        | \- Not publicly available; run a node |
| **Tendermint RPC** | Consensus and node status data     | \- rpc.ninerealms.com<br>- rpc.switchly.liquify.com<br>- rpc.thorswap.net<br>- Pre-hard-fork (blocks ≤4786559): rpc-v0.ninerealms.com                     | \- stagenet-rpc.ninerealms.com        |

## Usage Guidelines

- **Rate Limits**: Public endpoints may enforce limits (e.g., 100 requests/minute for Midgard). Check provider docs or contact operators (e.g., SWITCHLYSwap, Nine Realms).
- **Run Your Own Node**: For production apps, run a SWITCHLYNode to avoid rate limits and ensure uptime. See SWITCHLYNode Setup Guide.
- **Hard-Fork Note**: Mainnet hard-fork at block 4786560 requires post-hard-fork endpoints for newer data; use pre-hard-fork endpoints for historical queries (blocks ≤4786559).
- **Error Handling**: Handle HTTP 429 (rate limit) or 503 (node overload) with exponential backoff for retries.

## Midgard

Midgard provides time-series data for swaps, pools, volume, and liquidity providers, ideal for dashboards and analytics. It proxies SWITCHLYNode queries to reduce network load and runs on every node.

- **Mainnet:**
  - [https://midgard.thorswap.net/v2/doc](https://midgard.thorswap.net/v2/doc)
  - [https://midgard.ninerealms.com/v2/doc](https://midgard.ninerealms.com/v2/doc)
  - [https://midgard.switchly.liquify.com/v2/doc](https://midgard.switchly.liquify.com/v2/doc)
- **Stagenet:**
  - [https://stagenet-midgard.ninerealms.com/v2/doc](https://stagenet-midgard.ninerealms.com/v2/doc)

## SWITCHLYNode

SWITCHLYNode provides raw blockchain data (e.g., balances, transactions) specific to SWITCHLYChain’s state machine, critical for wallets and block explorers. Avoid excessive queries to public nodes to prevent overloading.

- **Mainnet (for post-hard-fork blocks 4786560 and later):**
  - [https://switchlynode.thorswap.net/switchly/doc](https://switchlynode.thorswap.net/switchly/doc)
  - [https://switchlynode.ninerealms.com/switchly/doc](https://switchlynode.ninerealms.com/switchly/doc)
  - [https://switchlynode.switchly.liquify.com/switchly/doc](https://switchlynode.switchly.liquify.com/switchly/doc)
  - **Pre-hard-fork blocks 4786559 and earlier**\
    [https://switchlynode-v0.ninerealms.com/switchly/doc](https://switchlynode-v0.ninerealms.com/switchly/doc)
- **Stagenet:**
  - [https://stagenet-switchlynode.ninerealms.com/switchly/doc](https://stagenet-switchlynode.ninerealms.com/switchly/doc)

## Cosmos RPC

Cosmos RPC provides generic Cosmos SDK data (e.g., account balances, transactions). Common endpoints include `/cosmos/bank/v1beta1/balances` and `/cosmos/base/tendermint/v1beta1/blocks`. Not all endpoints are enabled.

### Cosmos Documentation

- Cosmos SDK v0.50 RPC - [Cosmos gRPC Guide](https://docs.cosmos.network/v0.50/learn/advanced/grpc_rest)

### Cosmos Endpoints

Use SWITCHLYNode URLs with `/cosmos/...` paths.

- **Example**
  - **Mainnet** [https://switchlynode.ninerealms.com/cosmos/bank/v1beta1/balances/thor1dheycdevq39qlkxs2a6wuuzyn4aqxhve4qxtxt](https://switchlynode.ninerealms.com/cosmos/bank/v1beta1/balances/thor1dheycdevq39qlkxs2a6wuuzyn4aqxhve4qxtxt)
  - **Stagenet** [https://stagenet-switchlynode.ninerealms.com/cosmos/bank/v1beta1/balances/sthor1qhm0wjsrlw8wpvzrnpj8xxqu87tcucd6h98le4](https://stagenet-switchlynode.ninerealms.com/cosmos/bank/v1beta1/balances/sthor1qhm0wjsrlw8wpvzrnpj8xxqu87tcucd6h98le4)

## Tendermint RPC

Tendermint (CometBFT) RPC provides consensus and node status data (e.g., block height, validator status), useful for monitoring and debugging.

### Tendermint Documentation

- CometBFT v0.38 RPC - [CometBFT RPC Guide](https://docs.cometbft.com/v0.38/rpc/)

### Ports

- MAINNET Port: `27147`
- STAGENET Port: `26657`

### Tendermint RPC Endpoints

- **Mainnet:(for post-hard-fork blocks 4786560 and later)**
  - [https://rpc.ninerealms.com](https://rpc.ninerealms.com)
  - [https://rpc.switchly.liquify.com/genesis](https://rpc.switchly.liquify.com/genesis)
  - [https://rpc.thorswap.net/](https://rpc.thorswap.net/)
- **Mainnet:Pre-hard-fork blocks 4786559 and earlier.**
  - [https://rpc-v0.ninerealms.com](https://rpc-v0.ninerealms.com)
- **Stagenet:**
  - [https://stagenet-rpc.ninerealms.com](https://stagenet-rpc.ninerealms.com/)

## P2P

P2P is the peer-to-peer network layer for node communication, useful for debugging connectivity issues between SWITCHLYChain nodes.

- MAINNET Port: `27146`
- STAGENET Port: `26656`

## P2P Guide

- [https://docs.tendermint.com/master/spec/rpc/](https://docs.tendermint.com/master/spec/rpc/)

### Example: Check Peer Connections

```bash
curl -X GET "https://rpc.ninerealms.com/net_info" -H "accept: application/json"
```
