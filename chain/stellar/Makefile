# Switchly Stellar (Soroban) router contract — build & deploy.
#
# Prereqs (see README.md): rustup + `wasm32v1-none` target, and the `stellar` CLI.
# Run all targets from this directory (chain/stellar) so the contract workspace and
# the deployed contract alias line up.
#
# Common overrides:
#   make deploy NETWORK=testnet IDENTITY=switchly-deployer
NETWORK  ?= testnet
IDENTITY ?= switchly-deployer
ALIAS    ?= switchly_router
WASM     := target/wasm32v1-none/release/switchly_router.wasm

.PHONY: build test identity fund deploy version clean

## build: compile + optimize the contract to wasm
build:
	stellar contract build

## test: run the contract's Rust unit tests
test:
	cargo test

## identity: generate the deployer key and fund it via friendbot (testnet only)
identity:
	stellar keys generate $(IDENTITY) --network $(NETWORK) --fund
	stellar keys address $(IDENTITY)

## fund: (re)fund the deployer identity via friendbot (testnet only)
fund:
	stellar keys fund $(IDENTITY) --network $(NETWORK)

## deploy: build then deploy to $(NETWORK); records the id under the $(ALIAS) alias
deploy: build
	stellar contract deploy --wasm $(WASM) --source $(IDENTITY) --network $(NETWORK) --alias $(ALIAS)

## version: read the deployed contract's version() (sanity check)
version:
	stellar contract invoke --id $(ALIAS) --source $(IDENTITY) --network $(NETWORK) -- version

## clean: remove build artifacts
clean:
	cargo clean
