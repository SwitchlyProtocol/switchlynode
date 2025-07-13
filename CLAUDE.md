# Switchly Development Guide

This document provides comprehensive information about the Switchly development environment.

## Quick Start

```bash
git clone https://gitlab.com/switchlyprotocol/switchlynode.git
cd switchlynode
make install
```

## Architecture Overview

Switchly is a decentralized liquidity network built with Cosmos SDK and TSS-lib. The architecture consists of several key components:

### Core Components

1. **Switchly Node**: The main blockchain node
   - Implements the Switchly state machine logic
   - Handles consensus and validation
   - Manages cross-chain operations

2. **Bifrost**: Bridge component connecting external chains to Switchly
   - Observes external chain transactions
   - Signs outbound transactions using TSS
   - Manages chain-specific logic

### Module Structure

- **x/switchly**: Main module implementing Switchly-specific logic
- **bifrost/**: Bridge client for external chain integration
- **common/**: Shared utilities and types
- **constants/**: Network constants and parameters

## Network Configuration

Switchly supports multiple network types, each with specific configurations:

### Address Prefixes

- **Mainnet**: `switch` (accounts), `switchpub` (public keys)
- **Testnet**: `tswitch` (accounts), `tswitchpub` (public keys)
- **Stagenet**: `sswitch` (accounts), `sswitchpub` (public keys)

### Network Asset

- **Symbol**: SWITCH
- **Denom**: switch
- **Decimals**: 8

## Development

### Handler Logic

Business logic is implemented in handlers:

- Handler logic in `/x/switchly/handler_*.go`
- Keeper functions in `/x/switchly/keeper/`
- Message types in `/x/switchly/types/`

### Testing

```bash
make test
```

### Building

```bash
make build
```
