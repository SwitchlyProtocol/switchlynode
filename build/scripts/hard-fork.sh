#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

. "$(dirname "$0")/core.sh"

# set defaults
FORK_HEIGHT="${FORK_HEIGHT:=1}"
EXPORT_HEIGHT="${EXPORT_HEIGHT:=$((FORK_HEIGHT - 1))}"

echo "Exporting state at height $EXPORT_HEIGHT"
switchlynode export --height "$EXPORT_HEIGHT" --for-zero-height --jail-allowed-addrs "" > ~/.switchlynode/config/exported_genesis.json

echo "Resetting state"
switchlynode tendermint unsafe-reset-all

echo "Copying exported genesis to genesis.json"
cp ~/.switchlynode/config/exported_genesis.json ~/.switchlynode/config/genesis.json

echo "Validating genesis"
switchlynode genesis validate --trace

echo "Starting node"
exec switchlynode start
