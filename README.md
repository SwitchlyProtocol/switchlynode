<!-- markdownlint-disable MD041 -->

[![pipeline status](https://gitlab.com/thorchain/thornode/badges/develop/pipeline.svg)](https://gitlab.com/thorchain/thornode/commits/develop)
[![coverage report](https://gitlab.com/thorchain/thornode/badges/develop/coverage.svg)](https://gitlab.com/thorchain/thornode/-/commits/develop)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/thorchain/thornode)](https://goreportcard.com/report/gitlab.com/thorchain/thornode)

# SwitchlyProtocol

SwitchlyProtocol is a decentralized cross-chain automated market maker (AMM) protocol built on the Cosmos SDK. It enables users to swap native assets across different blockchains without the need for wrapped tokens or pegged assets.

## Features

- **Cross-chain swaps**: Swap native assets across different blockchains
- **Continuous liquidity pools**: Provide liquidity and earn yield
- **Decentralized**: No centralized authorities or intermediaries
- **Secure**: Built on proven cryptographic primitives and consensus mechanisms

## Installation

### Prerequisites

- Go 1.21 or higher
- Git

### Build from source

```bash
git clone https://gitlab.com/switchlyprotocol/switchlynode.git
cd switchlynode
make install
```

## Usage

### Running a node

```bash
# Initialize node
switchlynode init <node-name> --chain-id switchlyprotocol-mainnet-v1

# Start node
switchlynode start
```

### Command line interface

```bash
# Get help
switchlynode help

# Query commands
switchlynode query help

# Transaction commands  
switchlynode tx help
```

## Development

### Running tests

```bash
make test
```

### Building Docker image

```bash
make docker-build
```

### Running local network

```bash
make run-mocknet
```

## Documentation

- [API Documentation](https://switchlyprotocol.com/docs/api)
- [Developer Guide](https://switchlyprotocol.com/docs/dev)
- [Node Operation Guide](https://switchlyprotocol.com/docs/node)

## Contributing

Please read our [Contributing Guide](CONTRIBUTING.md) for details on our code of conduct and the process for submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Community

- [Discord](https://discord.gg/switchlyprotocol)
- [Telegram](https://t.me/switchlyprotocol)
- [Twitter](https://twitter.com/switchlyprotocol)
- [Reddit](https://reddit.com/r/switchlyprotocol)

======================================

## Setup

Install dependencies, you may skip packages you already have.

Linux:

```bash
apt-get update
apt-get install -y git make golang-go protobuf-compiler
```

Mac:

```bash
brew install golang coreutils binutils diffutils findutils gnu-tar gnu-sed gawk grep make git protobuf

# Follow post-setup instructions...
# Your $PATH should look something like this
export PATH=/opt/homebrew/opt/make/libexec/gnubin:/opt/homebrew/opt/gnu-sed/libexec/gnubin:opt/homebrew/opt/libpq/bin:/opt/homebrew/opt/findutils/libexec/gnubin:$GOPATH/bin:/usr/local/bin:$PATH
```

On recent Mac machines, you may need to set a higher-priority path to replace `awk` with `gawk`:

```bash
ln -sf $(which gawk) /usr/local/bin/awk
```

Install [Docker and Docker Compose V2](https://docs.docker.com/engine/install/).

Ensure you have a recent version of go ([scripts/check-env.sh](https://gitlab.com/thorchain/thornode/-/blob/develop/scripts/check-env.sh#L46-48)) and enabled go modules.<br/>
Add `GOBIN` to your `PATH`.

```bash
export GOBIN=$GOPATH/bin
```

### Automated Install Locally

Clone repo

```bash
git clone https://gitlab.com/thorchain/thornode.git
cd thornode
```

Install via this `make` command.

```bash
make go-generate openapi proto-gen install
```

Once you've installed `thornode`, check that they are there.

```bash
thornode help
```

### Start Standalone Full Stack

For development and running a full chain locally (your own separate network), use the following command on the project root folder:

```bash
make run-mocknet
```

See [build/docker/README.md](./build/docker/README.md) for more detailed documentation on the THORNode images and local mocknet environment.

### Simulate Local Churn

```bash
# reset mocknet cluster
make reset-mocknet-cluster

# increase churn interval as desired from the default 60 blocks
make cli-mocknet
> thornode tx thorchain mimir CHURNINTERVAL 1000 --from dog $TX_FLAGS

# bootstrap vaults from simulation test add liquidity transactions
make bootstrap-mocknet

# verify vault balances
curl -s localhost:1317/thorchain/vaults/asgard | jq '.[0].coins'

# watch logs for churn
make logs-mocknet

# verify active nodes
curl -s localhost:1317/thorchain/nodes | jq '[.[]|select(.status=="Active")]|length'

# disable future churns if desired
make cli-mocknet
> thornode tx thorchain mimir CHURNINTERVAL 1000000 --from dog $TX_FLAGS
```

See [build/docker/README.md](./build/docker/README.md) for more detailed documentation on the THORNode images and local mocknet environment.

### Simulation Tests

More details on simulation tests can be found in the [Simulation Test README](./test/simulation/README.md).

```bash
make test-simulation
```

### Format code

```bash
make format
```

### Test

Run tests

```bash
make test
```

By default, computationally-expensive tests like those for `go-tss` are excluded from this target. To fully test the _entire_ codebase, leverage:

```bash
make test-all
```

To tests _only_ `go-tss` run:

```bash
make test-go-tss
```

### Regression Tests

We expose a testing framework that allows the definition of test cases and suites using a DSL in YAML. Providing a regular expression to the `RUN` environment variable will match against files in `test/regression/suites` to filter tests to run.

```bash
make test-regression

# with more detailed logs
DEBUG=1 make test-regression

# with specific test filters
RUN=core make test-regression
RUN=mimir/deprecate-ilp test-regression

# overwrite export state
EXPORT=1 make test-regression
```

### Ledger CLI Support

```bash
cd cmd/thornode
go build -tags cgo,ledger
./thornode keys add ledger1 --ledger
```

=====================

# Contributions

## Devs

- Create an issue or find an existing issue on https://gitlab.com/thorchain/thornode/-/issues
- About to work on an issue? Start a conversation at #thornode-dev channel on [discord](https://discord.gg/qrnnXqnWYt)
- Assign the issue to yourself
- Create a branch using the issue id, for example if the issue you are working on is 600, then create a branch call `600-issue`, this way, GitLab will link your PR with the issue
- Raise a PR, Once your PR is ready for review, post a message in #thornode-dev channel in discord, tag `@thornode-team` for review
- If you do not have the required permissions, the pipeline will not be able to run. In this instance you will need to setup & register a [local runner](https://docs.gitlab.com/runner/register/)
- If you have completed this step (or have the required permissions to use the shared runners) make sure the pipeline completes and all is green
- Once the PR gets all required approvals by other contributors, it can be merged

Current active branch is `develop`, so when you open PR, make sure your target branch is `develop`

## ADRs

THORChain follows a Architectural Decision Record process outlined here:
https://gitlab.com/thorchain/thornode/-/blob/develop/docs/architecture/PROCESS.md?ref_type=heads

## Upgrades

The network soft-forks once a month (asynchronous upgrades), and hard-forks once a year (synchronous upgrade).

## Vulnerabilities and Bug Bounties

If you find a vulnerability in THORNode, please submit it for a bounty according to these [guidelines](bugbounty.md).

## the semantic version and release

THORNode manage changelog entry the same way like Gitlab, refer to (https://docs.gitlab.com/ee/development/changelog.html) for more detail. Once a merge request get merged into master branch,
if the merge request upgrades the [version](https://gitlab.com/thorchain/thornode/-/blob/develop/version), then a new release will be created automatically, and the repository will be tagged with
the new version by the release tool.

## New Chain Integrations

The process to integrate a new chain into THORChain is multifaceted. As it requires changes to multiple repos in multiple languages (`golang`, `python`, and `javascript`).

To learn more about how to add a new chain, follow [this doc](docs/newchain.md)

To learn more about creating your own private chain as a testing and development environment, follow [this doc](docs/private_mock_chain.md)
