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
  NODE_ADDRESS=$(switchlynode keys show "$SIGNER_NAME" -a --keyring-backend test)

  if [ "$PEER" = "none" ]; then
    echo "Missing PEER"
    exit 1
  fi

  # wait for peer
  until curl -s "$PEER:$PORT_RPC" 1>/dev/null 2>&1; do
    echo "Waiting for peer: $PEER:$PORT_RPC"
    sleep 3
  done

  switchlynode tx switchly deposit 100000000000000 RUNE "bond:$NODE_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=test --chain-id "$CHAIN_ID" --yes

  # send bond

  sleep 2 # wait for switchly to commit a block , otherwise it get the wrong sequence number

  NODE_PUB_KEY=$(switchlynode keys show switchly --pubkey --keyring-backend=test | switchlynode pubkey)
  # Generate ed25519 key from mnemonic - following THORChain's approach
  NODE_PUB_KEY_ED25519=$(printf "%s\npassword\n" "$SIGNER_SEED_PHRASE" | switchlynode ed25519)
  VALIDATOR=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)

  # set node keys
  until switchlynode tx switchly set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=test --chain-id "$CHAIN_ID" --yes; do
    sleep 5
  done

  # add IP address
  sleep 2 # wait for switchlyprotocol to commit a block

  NODE_IP_ADDRESS=${EXTERNAL_IP:=$(curl -s http://whatismyip.akamai.com)}
  until switchlynode tx switchly set-ip-address "$NODE_IP_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=test --chain-id "$CHAIN_ID" --yes; do
    sleep 5
  done

  sleep 2 # wait for switchlyprotocol to commit a block
  # set node version
  until switchlynode tx switchly set-version --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=test --chain-id "$CHAIN_ID" --yes; do
    sleep 5
  done
}

# set external ip to localhost in mocknet
if [ "$NET" = "mocknet" ]; then
  EXTERNAL_IP="$(hostname -i)"
  export EXTERNAL_IP
fi