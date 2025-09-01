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

  if [ "$NET" = "mocknet" ]; then
    # Fetch genesis for mocknet cluster nodes
    echo "Fetching genesis from peer $PEER..."
    until curl -s "http://$PEER:26657/genesis" | jq '.result.genesis' > ~/.switchlynode/config/genesis.json; do
      echo "Waiting for peer genesis..."
      sleep 3
    done
    echo "Genesis fetched successfully"
  else
    NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
    echo "Your SwitchlyNode address: $NODE_ADDRESS"
    echo "Send your bond to that address"
    # Remove genesis for non-mocknet (will be fetched from network)
    rm -rf ~/.switchlynode/config/genesis.json
  fi
fi

# For mocknet cluster nodes, always check registration status and register if needed
if [ "$NET" = "mocknet" ] && [ "$PEER" != "none" ]; then
  # Ensure we have the user created
  create_switchly_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"
  
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
  echo "Node address: $NODE_ADDRESS"
  
  # Wait for peer to be available
  echo "Waiting for peer $PEER to be available..."
  until curl -s "http://$PEER:1317/switchly/nodes" >/dev/null 2>&1; do
    echo "Waiting for peer: $PEER:1317"
    sleep 3
  done
  
  # Check if this node is already FULLY configured on the network
  NODE_INFO=$(curl -s "http://$PEER:1317/switchly/node/$NODE_ADDRESS" || true)
  NODE_STATUS=$(echo "$NODE_INFO" | jq -r '.status // "Unknown"')
  NODE_IP=$(echo "$NODE_INFO" | jq -r '.ip_address // ""')
  NODE_VERSION=$(echo "$NODE_INFO" | jq -r '.version // ""')
  
  echo "Node check - Status: $NODE_STATUS, IP: $NODE_IP, Version: $NODE_VERSION"
  
  # Check if node is FULLY configured (has proper IP and version)
  if [ "$NODE_STATUS" = "Active" ] && [ "$NODE_IP" != "" ] && [ "$NODE_VERSION" != "0.0.0" ] && [ "$NODE_VERSION" != "" ]; then
    echo "Node fully registered with status $NODE_STATUS, IP $NODE_IP, version $NODE_VERSION - joining as validator..."
  else
    echo "Node needs registration (status: $NODE_STATUS, IP: '$NODE_IP', version: '$NODE_VERSION') - bonding and registering..."
    init_mocknet
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