# SwitchlyProtocol Development Guide

## Quick Start

```bash
# Clone the repository
git clone https://gitlab.com/switchlyprotocol/switchlynode.git
cd switchlynode

# Install switchlynode
make install

# Check installation
switchlynode help
```

## Project Structure

```
switchlynode/
├── app/                    # Cosmos SDK application setup
├── bifrost/               # External chain bridge
├── build/                 # Build scripts and Docker files
├── chain/                 # Chain-specific implementations
├── cmd/                   # Command-line interface
├── common/                # Shared utilities and types
├── config/                # Configuration management
├── constants/             # Protocol constants
├── openapi/               # API documentation
├── proto/                 # Protocol buffer definitions
├── scripts/               # Development scripts
├── tools/                 # Development tools
├── x/                     # Cosmos SDK modules
└── Makefile              # Build automation
```

## Development Workflow

### 1. Local Development

```bash
# Start local development environment
make run-mocknet

# Run tests
make test

# Build binary
make build

# Install binary
make install
```

### 2. Docker Development

```bash
# Build Docker image
make docker-build

# Run with Docker Compose
make run-docker

# Clean Docker environment
make docker-clean
```

### 3. Testing

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run regression tests
make test-regression

# Run simulation tests
make test-simulation
```

## Architecture Overview

SwitchlyProtocol is a decentralized liquidity network built with Cosmos SDK and TSS-lib. The architecture consists of several key components:

### Core Components

1. **SwitchlyNode**: The main blockchain node
   - Built on Cosmos SDK
   - Implements the SwitchlyProtocol state machine logic
   - Handles consensus and transaction processing

2. **Bifrost**: Bridge component connecting external chains to SwitchlyProtocol
   - Observes external chain transactions
   - Signs outbound transactions using TSS
   - Manages chain-specific logic
   - Communicates with SwitchlyNode via gRPC

### Transaction Flow

1. **Inbound**: External chains → Bifrost → SwitchlyNode
   - Bifrost observes external chain transactions
   - Validates and processes inbound transactions
   - Communicates with SwitchlyNode via gRPC

2. **Outbound**: SwitchlyNode → Bifrost → External chains
   - SwitchlyNode processes swaps, adds liquidity, etc.
   - Creates outbound transactions
   - Bifrost signs and broadcasts to external chains

### Key Modules

- **x/switchlyprotocol**: Main module implementing SwitchlyProtocol-specific logic
  - Pool management
  - Liquidity provision
  - Swapping logic
  - Node management
  - Governance

### Network Types

SwitchlyProtocol supports multiple network types, each with specific configurations:

1. **Mocknet**: Local development network
2. **Testnet**: Public testing network  
3. **Stagenet**: Pre-production network
4. **Mainnet**: Production network

## Development Guidelines

### Code Style

- Follow standard Go conventions
- Use meaningful variable names
- Add comprehensive comments
- Write unit tests for new functionality

### Testing Strategy

- Unit tests for individual functions
- Integration tests for module interactions
- Regression tests for bug fixes
- Simulation tests for complex scenarios

### Pull Request Process

1. Create feature branch from `develop`
2. Implement changes with tests
3. Update documentation if needed
4. Submit PR with clear description
5. Address review feedback
6. Merge after approval

### Key Development Areas

- Handler logic in `/x/switchlyprotocol/handler_*.go`
- Keeper functions in `/x/switchlyprotocol/keeper/`
- Message types in `/x/switchlyprotocol/types/`
- Chain clients in `/bifrost/pkg/chainclients/`
- Protocol constants in `/constants/`

## Common Tasks

### Adding New Chain Support

1. Create chain client in `/bifrost/pkg/chainclients/`
2. Implement required interfaces
3. Add chain configuration
4. Update genesis parameters
5. Add comprehensive tests

### Modifying Protocol Logic

1. Update handler functions
2. Modify keeper methods
3. Update message types if needed
4. Add migration logic if required
5. Test thoroughly with regression and simulation tests

## Debugging

### Local Debugging

```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run with race detection
make test-race

# Profile CPU usage
make profile-cpu

# Profile memory usage
make profile-mem
```

### Common Issues

1. **Genesis state issues**: Check genesis.json format
2. **Network connectivity**: Verify port configurations
3. **TSS issues**: Check key generation and signing
4. **Chain sync issues**: Verify external chain connections
