#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

. "$(dirname "$0")/core.sh"

if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  init_chain
  rm -rf ~/.switchlynode/config/genesis.json # set in thornode render-config
fi

# render tendermint and cosmos configuration files
thornode render-config

exec switchlynode start