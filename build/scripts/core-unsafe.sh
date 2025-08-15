#!/bin/bash

set -o pipefail

deploy_evm_contracts() {
  for CHAIN in ETH AVAX BASE; do
    (
      # deploy contract and get address from output
      echo "Deploying $CHAIN contracts"
      if ! python3 scripts/evm/evm-tool.py --chain $CHAIN --rpc "$(eval echo "\$${CHAIN}_HOST")" --action deploy >/tmp/evm-tool-$CHAIN.log 2>&1; then
        cat /tmp/evm-tool-$CHAIN.log && exit 1
      fi
      cat /tmp/evm-tool-$CHAIN.log
      CONTRACT=$(grep </tmp/evm-tool-$CHAIN.log "Router Contract Address" | awk '{print $NF}')

      # add contract address to genesis
      echo "$CHAIN Contract Address: $CONTRACT"

      (
        flock -x 200
        jq --arg CHAIN "$CHAIN" --arg CONTRACT "$CONTRACT" \
          '.app_state.switchly.chain_contracts += [{"chain": $CHAIN, "router": $CONTRACT}]' \
          ~/.switchlynode/config/genesis.json >/tmp/genesis-$CHAIN.json
        mv /tmp/genesis-$CHAIN.json ~/.switchlynode/config/genesis.json
      ) 200>/tmp/genesis.lock
    ) &
  done
  wait
}

init_mocknet() {  
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)

  if [ "$PEER" = "none" ]; then
    echo "Missing PEER"
    exit 1
  fi

  # wait for peer
  until curl -s "$PEER:$PORT_RPC" 1>/dev/null 2>&1; do
    echo "Waiting for peer: $PEER:$PORT_RPC"
    sleep 3
  done

  printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchly deposit 100000000000000 SWITCH "bond:$NODE_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes

  # send bond

  sleep 2 # wait for switchly to commit a block , otherwise it get the wrong sequence number


  NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" --pubkey --keyring-backend=file | switchlynode pubkey)
  NODE_PUB_KEY_ED25519=$(printf "%s\n" "$SIGNER_PASSWD" | switchlynode ed25519)
  VALIDASWITCHLY=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)

 # set node keys
  echo "Setting node keys..."
  SET_KEYS_RESULT=$(printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchly set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDASWITCHLY" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes 2>&1)
  if echo "$SET_KEYS_RESULT" | grep -q "already has pubkey set"; then
    echo "Node keys already set, continuing..."
  elif echo "$SET_KEYS_RESULT" | grep -q "code: 0"; then
    echo "Node keys set successfully"
  else
    echo "Failed to set node keys, retrying..."
    until printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchly set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDASWITCHLY" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
      sleep 5
    done
  fi

  # add IP address
  sleep 2 # wait for switchly to commit a block

  # For mocknet cluster nodes, detect Docker container IP properly
  if [ -z "$EXTERNAL_IP" ]; then
    if command -v hostname >/dev/null 2>&1; then
      # Get Docker container's internal IP address
      DOCKER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' | head -n1)"
      if [ -n "$DOCKER_IP" ] && [ "$DOCKER_IP" != "127.0.0.1" ]; then
        EXTERNAL_IP="$DOCKER_IP"
      else
        EXTERNAL_IP="$(hostname -i 2>/dev/null || curl -s http://whatismyip.akamai.com)"
      fi
    else
      EXTERNAL_IP="$(curl -s http://whatismyip.akamai.com)"
    fi
  fi
  
  NODE_IP_ADDRESS="$EXTERNAL_IP"
  echo "Setting node IP address to: $NODE_IP_ADDRESS"
  
  # Try IP registration with better error handling  
  for i in {1..5}; do
    echo "IP registration attempt $i/5..."
    if printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchly set-ip-address "$NODE_IP_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes 2>&1; then
      echo "IP address set successfully!"
      break
    else
      echo "IP set attempt $i failed, retrying in 5 seconds..."
      sleep 5
      if [ $i -eq 5 ]; then
        echo "Error: IP registration failed after 5 attempts!"
        exit 1
      fi
    fi
  done

  sleep 2 # wait for switchly to commit a block
  # set node version
  echo "Setting node version..."
  
  # Try version registration with better error handling
  for i in {1..5}; do
    echo "Version registration attempt $i/5..."
    if printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode tx switchly set-version --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes 2>&1; then
      echo "Version set successfully!"
      break
    else
      echo "Version set attempt $i failed, retrying in 5 seconds..."
      sleep 5
      if [ $i -eq 5 ]; then
        echo "Warning: Version registration failed after 5 attempts, but continuing..."
      fi
    fi
  done
  
  echo "Node registration completed!"
}

# set external ip for mocknet environments
if [ "$NET" = "mocknet" ]; then
  # For Docker containers, get the primary internal IP address
  if command -v hostname >/dev/null 2>&1; then
    # Try hostname -I first (more reliable for Docker)
    DOCKER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' | head -n1)"
    if [ -n "$DOCKER_IP" ] && [ "$DOCKER_IP" != "127.0.0.1" ]; then
      EXTERNAL_IP="$DOCKER_IP"
    else
      # Fallback to hostname -i
      EXTERNAL_IP="$(hostname -i 2>/dev/null || echo '172.18.0.100')"
    fi
  else
    # Last resort fallback
    EXTERNAL_IP="172.18.0.100"
  fi
  echo "Setting mocknet external IP to: $EXTERNAL_IP"
  export EXTERNAL_IP
fi
