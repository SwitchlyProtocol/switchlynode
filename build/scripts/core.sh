#!/bin/bash

set -o pipefail

# default ulimit is set too low for switchlynode in some environments
ulimit -n 65535

PORT_P2P=26656
PORT_RPC=26657
[ "$NET" = "mainnet" ] && PORT_P2P=27146 && PORT_RPC=27147
[ "$NET" = "stagenet" ] && PORT_P2P=27146 && PORT_RPC=27147
export PORT_P2P PORT_RPC

# validate required environment
if [ -z "$SIGNER_NAME" ]; then
  echo "SIGNER_NAME must be set"
  exit 1
fi
if [ -z "$SIGNER_PASSWD" ]; then
  echo "SIGNER_PASSWD must be set"
  exit 1
fi

# adds an account node into the genesis file
add_node_account() {
  NODE_ADDRESS=$1
  VALIDATOR=$2
  NODE_PUB_KEY=$3
  VERSION=$4
  BOND_ADDRESS=$5
  NODE_PUB_KEY_ED25519=$6
  IP_ADDRESS=$7
  MEMBERSHIP=$8
  jq --arg NODE_ADDRESS "$NODE_ADDRESS" --arg VERSION "$VERSION" --arg IP_ADDRESS "$IP_ADDRESS" --arg NODE_PUB_KEY_ED25519 "$NODE_PUB_KEY_ED25519" '.app_state.switchly.node_accounts += [{"node_address": $NODE_ADDRESS, "version": $VERSION, "ip_address": $IP_ADDRESS, "status": "Active","bond":"30000000000000", "active_block_height": "0", "bond_address": $NODE_ADDRESS, "pub_key_set":{"secp256k1":$NODE_PUB_KEY_ED25519,"ed25519":$NODE_PUB_KEY_ED25519}}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
  if [ -n "$MEMBERSHIP" ]; then
    jq --arg MEMBERSHIP "$MEMBERSHIP" '.app_state.switchly.node_accounts[-1].signer_membership += [$MEMBERSHIP]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
  fi
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

reserve() {
  jq --arg RESERVE "$1" '.app_state.switchly.reserve = $RESERVE' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

disable_bank_send() {
  jq '.app_state.bank.params.default_send_enabled = false' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json

  jq '.app_state.transfer.params.send_enabled = false' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

# inits a switchly with the provided list of genesis accounts
init_chain() {
  IFS=","

  echo "Init chain"
  switchlynode init local --chain-id "$CHAIN_ID"
  echo "$SIGNER_PASSWD" | switchlynode keys list --keyring-backend file
}

fetch_node_id() {
  until curl -s "$1:$PORT_RPC" 1>/dev/null 2>&1; do
    sleep 3
  done
  curl -s "$1:$PORT_RPC/status" | jq -r .result.node_info.id
}

set_node_keys() {
  SIGNER_NAME="$1"
  SIGNER_PASSWD="$2"
  PEER="$3"
  NODE_PUB_KEY="$(echo "$SIGNER_PASSWD" | switchlynode keys show switchly --pubkey --keyring-backend file | switchlynode pubkey)"
  NODE_PUB_KEY_ED25519="$(printf "%s\n" "$SIGNER_PASSWD" | switchlynode ed25519)"
  VALIDATOR="$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)"
  echo "Setting SwitchlyNode keys"
  printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode tx switchly set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" --node "tcp://$PEER:$PORT_RPC" --from "$SIGNER_NAME" --yes
}

set_ip_address() {
  SIGNER_NAME="$1"
  SIGNER_PASSWD="$2"
  PEER="$3"
  NODE_IP_ADDRESS="${4:-$(curl -s http://whatismyip.akamai.com)}"
  echo "Setting SwitchlyNode IP address $NODE_IP_ADDRESS"
  printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode tx switchly set-ip-address "$NODE_IP_ADDRESS" --node "tcp://$PEER:$PORT_RPC" --from "$SIGNER_NAME" --yes
}

fetch_version() {
  switchlynode query switchly version --output json | jq -r .version
}

create_switchly_user() {
  SIGNER_NAME="$1"
  SIGNER_PASSWD="$2"
  SIGNER_SEED_PHRASE="$3"

  echo "Checking if SwitchlyNode Switchly '$SIGNER_NAME' account exists"
  echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" --keyring-backend file 1>/dev/null 2>&1
  # shellcheck disable=SC2181
  if [ $? -ne 0 ]; then
    echo "Creating SwitchlyNode Switchly '$SIGNER_NAME' account"
    if [ -n "$SIGNER_SEED_PHRASE" ]; then
      printf "%s\n%s\n%s\n" "$SIGNER_SEED_PHRASE" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode keys --keyring-backend file add "$SIGNER_NAME" --recover
    else
      sig_pw=$(printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD")
      RESULT=$(echo "$sig_pw" | switchlynode keys --keyring-backend file add "$SIGNER_NAME" --output json 2>&1)
      SIGNER_SEED_PHRASE=$(echo "$RESULT" | jq -r '.mnemonic')
    fi
  fi
  NODE_PUB_KEY_ED25519=$(printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE" | switchlynode ed25519)
}

set_bond_module() {
  if [ "$NET" = "mocknet" ]; then
    BOND_MODULE_ADDR="tswtc17gw75axcnr8747pkanye45pnrwk7p9c3jrajtc"
  elif [ "$NET" = "stagenet" ]; then
    BOND_MODULE_ADDR="sswtc17gw75axcnr8747pkanye45pnrwk7p9c3zds5yr"
  else
    echo "Unsupported NET: $NET"
    exit 1
  fi

  jq --arg BOND_MODULE_ADDR "$BOND_MODULE_ADDR" \
    '.app_state.state.accounts += [
    {
      "@type": "/cosmos.auth.v1beta1.ModuleAccount",
      "base_account": {
        "account_number": "0",
        "address": $BOND_MODULE_ADDR,
        "pub_key": null,
        "sequence": "0"
      },
      "name": "bond",
      "permissions": []
    }
  ]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json

  jq --arg BOND_MODULE_ADDR "$BOND_MODULE_ADDR" \
    '.app_state.bank.balances += [
    {
      "address": $BOND_MODULE_ADDR,
      "coins": [ { "denom": "rune", "amount": "30000000000000" } ]
    }
  ]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_eth_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchly.chain_contracts += [{"chain": "ETH", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_avax_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchly.chain_contracts += [{"chain": "AVAX", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_bsc_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchly.chain_contracts += [{"chain": "BSC", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_base_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchly.chain_contracts = [{"chain": "BASE", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}