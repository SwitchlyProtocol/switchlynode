#!/bin/bash

set -o pipefail

. "$(dirname "$0")/core.sh"

if [ "$NET" = "mocknet" ]; then
  echo "Loading unsafe init for mocknet..."
  . "$(dirname "$0")/core-unsafe.sh"
fi

PEER="${PEER:=none}"          # the hostname of a seed node set as tendermint persistent peer
PEER_API="${PEER_API:=$PEER}" # the hostname of a seed node API if different

if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  echo "Setting SwitchlyNode as Validator node"

  create_switchly_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

  init_chain
  rm -rf ~/.switchlynode/config/genesis.json # set in switchlynode render-config

  if [ "$NET" = "mocknet" ]; then
    init_mocknet
  else
    NODE_ADDRESS=$(switchlynode keys show "$SIGNER_NAME" -a --keyring-backend test)
    echo "Your SwitchlyNode address: $NODE_ADDRESS"
    echo "Send your bond to that address"
  fi
fi

# render tendermint and cosmos configuration files
switchlynode render-config

export SIGNER_NAME
export SIGNER_PASSWD
exec switchlynode start