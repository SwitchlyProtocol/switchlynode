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
          '.app_state.thorchain.chain_contracts += [{"chain": $CHAIN, "router": $CONTRACT}]' \
          ~/.thornode/config/genesis.json >/tmp/genesis-$CHAIN.json
        mv /tmp/genesis-$CHAIN.json ~/.thornode/config/genesis.json
      ) 200>/tmp/genesis.lock
    ) &
  done
  wait
}

deploy_stellar_contracts() {
  # Only add Stellar contracts if XLM_HOST is set
  if [ -n "${XLM_HOST+x}" ]; then
    echo "Setting up Stellar router contracts"
    
    # Get the router address from the router upgrade info (same as EVM approach for mainnet/stagenet)
    # The router upgrade info files contain the actual deployed contract addresses
    case "$NET" in
      "mocknet")
        # For mocknet, use the mocknet router address
        CONTRACT="CAVLZEEZWWDBBROYGYLL7IXVUFTSLIXV5EOVU6KR467U43SOBZ36UZTY"
        ;;
      "stagenet")
        # For stagenet, use the stagenet router address  
        CONTRACT="CAVLZEEZWWDBBROYGYLL7IXVUFTSLIXV5EOVU6KR467U43SOBZ36UZTY"
        ;;
      "mainnet")
        # For mainnet, router address would be set when deployed
        # Currently empty in router_upgrade_info.go
        CONTRACT=""
        ;;
      *)
        echo "Unknown network: $NET"
        CONTRACT=""
        ;;
    esac
    
    # Only add to genesis if we have a valid contract address
    if [ -n "$CONTRACT" ]; then
      echo "XLM Contract Address: $CONTRACT"
      
      (
        flock -x 200
        jq --arg CHAIN "XLM" --arg CONTRACT "$CONTRACT" \
          '.app_state.thorchain.chain_contracts += [{"chain": $CHAIN, "router": $CONTRACT}]' \
          ~/.thornode/config/genesis.json >/tmp/genesis-XLM.json
        mv /tmp/genesis-XLM.json ~/.thornode/config/genesis.json
      ) 200>/tmp/genesis.lock
      
      echo "Stellar router configuration complete"
      echo "Using router: $CONTRACT"
      echo "Network: $NET"
    else
      echo "No Stellar router contract configured for network: $NET"
      echo "Skipping Stellar router setup"
    fi
  else
    echo "XLM_HOST not set, skipping Stellar router setup"
  fi
}

init_mocknet() {
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | thornode keys show "$SIGNER_NAME" -a --keyring-backend file)

  if [ "$PEER" = "none" ]; then
    echo "Missing PEER"
    exit 1
  fi

  # wait for peer
  until curl -s "$PEER:$PORT_RPC" 1>/dev/null 2>&1; do
    echo "Waiting for peer: $PEER:$PORT_RPC"
    sleep 3
  done

  printf "%s\n" "$SIGNER_PASSWD" | thornode tx thorchain deposit 100000000000000 RUNE "bond:$NODE_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes

  # send bond

  sleep 2 # wait for thorchain to commit a block , otherwise it get the wrong sequence number

  NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | thornode keys show thorchain --pubkey --keyring-backend=file | thornode pubkey)
  VALIDATOR=$(thornode tendermint show-validator | thornode pubkey --bech cons)

  # set node keys
  until printf "%s\n" "$SIGNER_PASSWD" | thornode tx thorchain set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
    sleep 5
  done

  # add IP address
  sleep 2 # wait for thorchain to commit a block

  NODE_IP_ADDRESS=${EXTERNAL_IP:=$(curl -s http://whatismyip.akamai.com)}
  until printf "%s\n" "$SIGNER_PASSWD" | thornode tx thorchain set-ip-address "$NODE_IP_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
    sleep 5
  done

  sleep 2 # wait for thorchain to commit a block
  # set node version
  until printf "%s\n" "$SIGNER_PASSWD" | thornode tx thorchain set-version --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
    sleep 5
  done
}

# set external ip to localhost in mocknet
if [ "$NET" = "mocknet" ]; then
  EXTERNAL_IP="$(hostname -i)"
  export EXTERNAL_IP
fi
