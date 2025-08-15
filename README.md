<!-- markdownlint-disable MD041 -->

# SwitchlyNode

[![pipeline status](https://gitlab.com/switchly/switchlynode/badges/develop/pipeline.svg)](https://gitlab.com/switchly/switchlynode/commits/develop)
[![coverage report](https://gitlab.com/switchly/switchlynode/badges/develop/coverage.svg)](https://gitlab.com/switchly/switchlynode/-/commits/develop)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/switchly/switchlynode)](https://goreportcard.com/report/gitlab.com/switchly/switchlynode)

SwitchlyNode is the core blockchain protocol powering SwitchlyProtocol, a decentralized cross-chain automated market maker (AMM) built on the Cosmos SDK.

## Key Features

SwitchlyProtocol enables:
- **Cross-chain swaps**: Native asset swaps across different blockchains without wrapped tokens
- **Liquidity provision**: Earn yield by providing liquidity to cross-chain pools  
- **Savers vaults**: Single-sided liquidity provision with native yield
- **Lending protocol**: Collateralized lending using cross-chain assets
- **Trade accounts**: Margin trading with cross-chain collateral

## Architecture

SwitchlyNode consists of several key components:

- **SwitchlyNode**: Cosmos SDK-based blockchain handling consensus, validation, and cross-chain logic
- **Bifrost**: Multi-chain bridge client connecting SwitchlyProtocol to external blockchains
- **TSS (Threshold Signature Scheme)**: Distributed key management for secure cross-chain transactions

## Supported Chains

SwitchlyProtocol supports native assets from:
- Bitcoin (BTC)
- Ethereum (ETH) and ERC-20 tokens  
- Binance Smart Chain (BSC) and BEP-20 tokens
- Avalanche (AVAX) and ARC-20 tokens
- Cosmos (ATOM)
- Dogecoin (DOGE)
- Bitcoin Cash (BCH)
- Litecoin (LTC)
- And more...

## Documentation

- [SwitchlyProtocol Documentation](https://docs.switchlyprotocol.org)
- [Developer Resources](https://dev.switchlyprotocol.org)
- [API Documentation](https://switchlynode.docs.switchlyprotocol.org)

## Installation

### Prerequisites

Ensure you have a recent version of go ([scripts/check-env.sh](https://gitlab.com/switchly/switchlynode/-/blob/develop/scripts/check-env.sh#L46-48)) and enabled go modules.<br/>
Also, make sure you have `jq` installed.

### Build

```bash
git clone https://gitlab.com/switchly/switchlynode.git
cd switchlynode
make install
```

This will install `switchlynode` binary into your `$GOPATH/bin`.

### Run a local development environment

For a quick start with a local development environment:

```bash
make reset-mocknet-standalone
```

This will:
1. Build the `switchlynode` binary
2. Initialize a single-node testnet
3. Start the node with pre-configured accounts and balances

### Configuration

The node can be configured through:
- Environment variables
- Command line flags  
- Configuration files in `~/.switchlynode/config/`

Key configuration files:
- `app.toml`: Application-specific settings
- `config.toml`: Tendermint consensus settings
- `genesis.json`: Genesis state and parameters

### Running

To start the node:

```bash
switchlynode start
```

For development with automatic restarts:

```bash
make run-mocknet-standalone
```

## Development

### Testing

Run the full test suite:

```bash
make test
```

Run specific tests:

```bash
go test ./x/switchlyprotocol/...
```

### Linting

```bash
make lint
```

### Building Docker Images

```bash
make docker-build
```

## API Usage

### Query Examples

Get node information:
```bash
curl -s localhost:1317/switchlyprotocol/nodes | jq
```

Get pool information:
```bash  
curl -s localhost:1317/switchlyprotocol/pools | jq
```

Get network status:
```bash
curl -s localhost:1317/switchlyprotocol/network | jq
```

### Transaction Examples

Set mimir values (admin only):
```bash
switchlynode tx switchlyprotocol mimir CHURNINTERVAL 1000 --from admin $TX_FLAGS
```

## Contributing

SwitchlyProtocol follows a structured development process:

- Create an issue or find an existing issue on https://gitlab.com/switchly/switchlynode/-/issues
- Fork the repository and create a feature branch
- Make your changes following the coding standards
- Add tests for new functionality
- Submit a merge request with a clear description

### Architectural Decision Records

SwitchlyProtocol follows an Architectural Decision Record process outlined here:
https://gitlab.com/switchly/switchlynode/-/blob/develop/docs/architecture/PROCESS.md?ref_type=heads

### Release Process

Releases are automated through GitLab CI/CD. When a merge request is merged to the `develop` branch,
if the merge request upgrades the [version](https://gitlab.com/switchly/switchlynode/-/blob/develop/version), then a new release will be created automatically, and the repository will be tagged with
the new version.

### Chain Integration

The process to integrate a new chain into SwitchlyProtocol is multifaceted, requiring changes to multiple repositories across different languages (`golang`, `python`, and `javascript`).
