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

  create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

  init_chain

  if [ "$NET" = "mocknet" ]; then
    # For cluster nodes, check if we already have pub keys set before trying to init
    NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
    echo "Node address: $NODE_ADDRESS"
    
    # Wait for peer to be available
    echo "Waiting for peer $PEER to be available..."
    until curl -s "http://$PEER:1317/switchly/nodes" >/dev/null 2>&1; do
      echo "Waiting for peer: $PEER:1317"
      sleep 3
    done
    
    # Check if this node is already fully registered on the network (status != Unknown)
    NODE_INFO=$(curl -s "http://$PEER:1317/switchly/node/$NODE_ADDRESS" || true)
    NODE_STATUS=$(echo "$NODE_INFO" | jq -r '.status // "Unknown"')
    if [ "$NODE_STATUS" = "Active" ] || [ "$NODE_STATUS" = "Standby" ]; then
      echo "Node already registered on network with status $NODE_STATUS, joining as validator..."
      # Fetch genesis from peer to ensure proper validator set
      echo "Fetching genesis from peer $PEER..."
      until curl -s "http://$PEER:26657/genesis" | jq '.result.genesis' > ~/.switchlynode/config/genesis.json; do
        echo "Waiting for peer genesis..."
        sleep 3
      done
      echo "Genesis fetched successfully"
    else
      echo "Node status $NODE_STATUS – bonding and registering keys/IP..."
      # Fetch genesis from peer to ensure proper validator set
      echo "Fetching genesis from peer $PEER..."
      until curl -s "http://$PEER:26657/genesis" | jq '.result.genesis' > ~/.switchlynode/config/genesis.json; do
        echo "Waiting for peer genesis..."
        sleep 3
      done
      echo "Genesis fetched successfully"
      init_mocknet
    fi
  else
    NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
    echo "Your SwitchlyNode address: $NODE_ADDRESS"
    echo "Send your bond to that address"
    # Remove genesis for non-mocknet (will be fetched from network)
    rm -rf ~/.switchlynode/config/genesis.json
  fi
fi

# render tendermint and cosmos configuration files
switchlynode render-config

# For mocknet cluster nodes, ensure we use the correct genesis from the main node
if [ "$NET" = "mocknet" ] && [ "$PEER" != "none" ]; then
  echo "Fetching genesis from main node after render-config..."
  until curl -s "http://$PEER:26657/genesis" | jq '.result.genesis' > ~/.switchlynode/config/genesis.json; do
    echo "Waiting for peer genesis..."
    sleep 3
  done
  echo "Genesis updated successfully"
fi

export SIGNER_NAME
export SIGNER_PASSWD
exec switchlynode start