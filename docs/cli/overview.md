# Overview

## Install (macOS)

### Prerequisites

1. `xcode-select xcode-select --install`
2. Homebrew: [https://brew.sh](https://brew.sh)

### GoLang

Install Go: [https://go.dev/dl](https://go.dev/dl)

```shell
# Set Go PATH
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOROOT:$GOPATH:$GOBIN
```

### Protobuf

```shell
# Install Protobuf
brew install protobuf
brew install protoc-gen-go
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

### GNU Utils

```shell
# Install GNU utils
brew install coreutils binutils diffutils findutils gnu-tar gnu-sed gawk grep make

# Set GNU flags
export LDFLAGS=-L$(brew --prefix)/opt/binutils/lib
export CPPFLAGS=-I$(brew --prefix)/opt/binutils/include

# Set GNU PATH
export PATH=$(brew --prefix)/opt/binutils/bin:$PATH
export PATH=$(brew --prefix)/opt/findutils/libexec/gnubin:$PATH
export PATH=$(brew --prefix)/opt/gnu-tar/libexec/gnubin:$PATH
export PATH=$(brew --prefix)/opt/gnu-sed/libexec/gnubin:$PATH
export PATH=$(brew --prefix)/opt/gawk/libexec/gnubin:$PATH
export PATH=$(brew --prefix)/opt/grep/libexec/gnubin:$PATH
export PATH=$(brew --prefix)/opt/make/libexec/gnubin:$PATH
```

### Docker

```shell
# Install docker
brew install homebrew/cask/docker
```

### SWITCHLYNode

```shell
# Clone repo and install dependencies
git clone https://gitlab.com/switchly/switchlynode
# Docker must be started...
make openapi
make proto-gen
make install
```

Build mainnet binary with Ledger support:

```shell
go build -ldflags '-X github.com/cosmos/cosmos-sdk/version.Name=SWITCHLYChain -X github.com/cosmos/cosmos-sdk/version.AppName=switchlynode -X github.com/cosmos/cosmos-sdk/version.BuildTags=mainnet,ledger' -tags "mainnet ledger" -o ./cmd/switchlynode ./cmd/switchlynode
```

## Commands

`switchlynode --help`

```text
SWITCHLYChain Network

Usage:
  SWITCHLYChain [command]

Available Commands:
  add-genesis-account Add a genesis account to genesis.json
  collect-gentxs      Collect genesis txs and output a genesis.json file
  compact             force leveldb compaction
  debug               Tool for helping with debugging your application
  ed25519             Generate an ed25519 keys
  export              Export state to JSON
  gentx               Generate a genesis tx carrying a self delegation
  help                Help about any command
  init                Initialize private validator, p2p, genesis, and application configuration files
  keys                Manage your application's keys
  migrate             Migrate genesis to a specified target version
  pubkey              Convert Proto3 JSON encoded pubkey to bech32 format
  query               Querying subcommands
  render-config       renders tendermint and cosmos config from switchlynode base config
  rollback            rollback cosmos-sdk and tendermint state by one height
  start               Run the full node
  status              Query remote node for status
  tendermint          Tendermint subcommands
  tx                  Transactions subcommands
  util                Utility commands for the SWITCHLYChain module
  validate-genesis    validates the genesis file at the default location or at the location passed as an arg
  version             Print the application binary version information

Flags:
  -h, --help                help for SWITCHLYChain
      --home string         directory for config and data (default "/Users/dev/.switchlynode")
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic) (default "info")
      --trace               print out full stack trace on errors

Use "SWITCHLYChain [command] --help" for more information about a command.
```

### Popular Commands

#### Add new account

```text
switchlynode keys add {accountName}
```

#### Add existing account (via mnemonic)

```text
switchlynode keys add {accountName} --recover
```

#### List all accounts

```text
switchlynode keys list
```

## Send Transaction

### Create Transaction

```text
# Sender: thor1505gp5h48zd24uexrfgka70fg8ccedafsnj0e3
# Receiver: thor1gutjhrw4xlu3n3p3k3r0vexl2xknq3nv8ux9fy
# Amount: 1 SWITCH (in 1e8 notation)
switchlynode tx bank send thor1505gp5h48zd24uexrfgka70fg8ccedafsnj0e3 thor1gutjhrw4xlu3n3p3k3r0vexl2xknq3nv8ux9fy 100000000rune --chain-id switchly-1 --node https://rpc.ninerealms.com:443 --gas 3000000 --generate-only > tx_raw.json
```

This will output a file called `tx_raw.json`. Edit this file and change the `@type` field from `/cosmos.bank.v1beta1.MsgSend` to `/types.MsgSend`.

The `tx_raw.json` transaction should look like this:

```json
{
  "body": {
    "messages": [
      {
        "@type": "/types.MsgSend",
        "from_address": "thor1505gp5h48zd24uexrfgka70fg8ccedafsnj0e3",
        "to_address": "thor1gutjhrw4xlu3n3p3k3r0vexl2xknq3nv8ux9fy",
        "amount": [{ "denom": "rune", "amount": "100000000" }]
      }
    ],
    "memo": "",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [],
    "fee": { "amount": [], "gas_limit": "3000000", "payer": "", "granter": "" }
  },
  "signatures": []
}
```

### Sign Transaction

```text
switchlynode tx sign tx_raw.json --from {accountName} --sign-mode amino-json --chain-id switchly-1 --node https://rpc.ninerealms.com:443 > tx.json
```

This will output a file called `tx.json`.

### Broadcast Transaction

```text
switchlynode tx broadcast tx.json --chain-id switchly-1 --node https://rpc.ninerealms.com:443 --gas auto
```
