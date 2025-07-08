#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

. "$(dirname "$0")/core.sh"

########################################################################################
# Mocknet Init
########################################################################################

init_mocknet() {
  echo "Creating mocknet genesis..."
  
  # Add genesis accounts
  add_account swtc1z63f3mzwv3g75az80xwmhrawdqcjpaek5l3xv6 swtc 500000000000000000000 # cat
  add_account swtc1xwusttz86hqfuk5z7amcgqsg7vp6g8zhsp5lu2 swtc 500000000000000000000 # dog
  add_account swtc1ragkh3hp7wvdpw3qgxr5vchqaecmupv8ql3p5y swtc 500000000000000000000 # fox
  add_account swtc1wz78qmrkplrdhy37tw0tnvn0tkm5pqd6zdp257 swtc 500000000000000000000 # pig
  
  # Reserve funds
  reserve 22000000000000000
  
  # Disable bank send
  disable_bank_send
  
  # Set up node account
  set_node_account
  
  # Set up contracts
  set_eth_contract "0x0000000000000000000000000000000000000000"
  set_avax_contract "0x0000000000000000000000000000000000000000"
  set_bsc_contract "0x0000000000000000000000000000000000000000"
  set_base_contract "0x0000000000000000000000000000000000000000"
  
  # Validate genesis
  switchlynode validate-genesis
}

# SWTC deposit functions
deposit_swtc() {
  AMOUNT=${1:=100000000000000000000}
  TO_ADDRESS=${2:=swtc1z63f3mzwv3g75az80xwmhrawdqcjpaek5l3xv6}
  
  echo "Depositing $AMOUNT SWTC to $TO_ADDRESS"
  switchlynode tx bank send "$SIGNER_NAME" "$TO_ADDRESS" "$AMOUNT"swtc \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

# Transaction functions
create_pool() {
  ASSET=${1:=BTC.BTC}
  echo "Creating pool for $ASSET"
  switchlynode tx switchlyprotocol deposit 100000000000000000000 swtc "ADD:$ASSET" \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

set_mimir() {
  KEY=${1:=HALTCHAINGLOBAL}
  VALUE=${2:=1}
  echo "Setting mimir $KEY to $VALUE"
  switchlynode tx switchlyprotocol mimir "$KEY" "$VALUE" \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

if [ "$NET" = "mocknet" ]; then
  echo "Mocknet environment detected"
fi

add_contract() {
  CHAIN=$1
  CONTRACT=$2
  jq --arg CHAIN "$CHAIN" --arg CONTRACT "$CONTRACT" \
    '.app_state.switchlyprotocol.chain_contracts += [{"chain": $CHAIN, "router": $CONTRACT}]' \
    ~/.switchlynode/config/genesis.json >/tmp/genesis-$CHAIN.json
  mv /tmp/genesis-$CHAIN.json ~/.switchlynode/config/genesis.json
}

add_account() {
  jq --arg ADDRESS "$1" --arg ASSET "$2" --arg AMOUNT "$3" '.app_state.auth.accounts += [{
        "@type": "/cosmos.auth.v1beta1.BaseAccount",
        "address": $ADDRESS,
        "pub_key": null,
        "account_number": "0",
        "sequence": "0"
    }]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json

  jq --arg ADDRESS "$1" --arg ASSET "$2" --arg AMOUNT "$3" '.app_state.bank.balances += [{
        "address": $ADDRESS,
        "coins": [ { "denom": $ASSET, "amount": $AMOUNT } ],
    }]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

init_chain() {
  echo "Init chain"
  switchlynode init local --chain-id "$CHAIN_ID"
  echo "$SIGNER_PASSWD" | switchlynode keys list --keyring-backend file
}

set_xlm_contract() {
  CHAIN=XLM
  CONTRACT=$1
  jq --arg CHAIN "$CHAIN" --arg CONTRACT "$CONTRACT" \
    '.app_state.switchlyprotocol.chain_contracts += [{"chain": $CHAIN, "router": $CONTRACT}]' \
    ~/.switchlynode/config/genesis.json >/tmp/genesis-XLM.json
  mv /tmp/genesis-XLM.json ~/.switchlynode/config/genesis.json
}

# genesis on first init if we are the genesis node
if [ "$SEED" = "switchlyprotocol" ]; then
  if [ ! -f ~/.switchlynode/config/genesis.json ]; then
    init_chain
    create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

    # get our node account
    NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
    add_account "$NODE_ADDRESS" swtc 100000000000

    # reserve
    jq --arg RESERVE "22000000000000000" '.app_state.switchlyprotocol.reserve = $RESERVE' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.switchlynode/config/genesis.json

    # disable bank send
    jq '.app_state.bank.params.default_send_enabled = false' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.switchlynode/config/genesis.json

    jq '.app_state.transfer.params.send_enabled = false' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.switchlynode/config/genesis.json

    # render tendermint and cosmos configuration files
    switchlynode render-config
  fi

  # genesis on first init if we are the genesis node
  if [ ! -f ~/.switchlynode/config/genesis.json ]; then
    init_chain
    create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

    # get our node account
    NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
    printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchlyprotocol deposit 100000000000000 SWTC "bond:$NODE_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes

    # wait for switchlynode to commit a block , otherwise it get the wrong sequence number
    sleep 2 # wait for switchlynode to commit a block , otherwise it get the wrong sequence number

    NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show switchlyprotocol --pubkey --keyring-backend=file | switchlynode pubkey)
    VALIDATOR=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)
    NODE_PUB_KEY_ED25519=$(printf "%s\n" "$SIGNER_PASSWD" | switchlynode ed25519)

    until printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchlyprotocol set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
      echo "Failed to set node keys, retrying in 3 seconds..."
      sleep 3
    done
    sleep 2 # wait for switchlynode to commit a block

    NODE_IP_ADDRESS=${EXTERNAL_IP:=$(curl -s http://whatismyip.akamai.com)}
    until printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchlyprotocol set-ip-address "$NODE_IP_ADDRESS" --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
      echo "Failed to set IP address, retrying in 3 seconds..."
      sleep 3
    done
    sleep 2 # wait for switchlynode to commit a block

    until printf "%s\n" "$SIGNER_PASSWD" | switchlynode tx switchlyprotocol set-version --node tcp://"$PEER":26657 --from "$SIGNER_NAME" --keyring-backend=file --chain-id "$CHAIN_ID" --yes; do
      echo "Failed to set version, retrying in 3 seconds..."
      sleep 3
    done
  fi
fi

exec "$@"
