#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchly}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

. "$(dirname "$0")/core.sh"

if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  init_chain
  rm -rf ~/.switchlynode/config/genesis.json # set in switchlynode render-config
fi

# render tendermint and cosmos configuration files
switchlynode render-config

exec switchlynode start
