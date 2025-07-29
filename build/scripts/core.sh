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
  jq --arg IP_ADDRESS "$IP_ADDRESS" --arg VERSION "$VERSION" --arg BOND_ADDRESS "$BOND_ADDRESS" --arg VALIDATOR "$VALIDATOR" --arg NODE_ADDRESS "$NODE_ADDRESS" --arg NODE_PUB_KEY "$NODE_PUB_KEY" --arg NODE_PUB_KEY_ED25519 "$NODE_PUB_KEY_ED25519" '.app_state.switchly.node_accounts += [{"node_address": $NODE_ADDRESS, "version": $VERSION, "ip_address": $IP_ADDRESS, "status": "Active","bond":"30000000000000", "active_block_height": "0", "bond_address":$BOND_ADDRESS, "signer_membership": [], "validator_cons_pub_key":$VALIDATOR, "pub_key_set":{"secp256k1":$NODE_PUB_KEY,"ed25519":$NODE_PUB_KEY_ED25519}}]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
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
  echo "DEBUG: CHAIN_ID is '$CHAIN_ID'"
  echo "DEBUG: About to run: switchlynode init local --chain-id $CHAIN_ID"
  
  # Test the init command with explicit error handling
  if switchlynode init local --chain-id "$CHAIN_ID" 2>&1; then
    echo "DEBUG: switchlynode init successful"
  else
    echo "ERROR: switchlynode init failed with exit code $?"
    echo "DEBUG: Trying alternative init syntax..."
    
    # Try alternative init command syntax
    if switchlynode init --chain-id "$CHAIN_ID" local 2>&1; then
      echo "DEBUG: Alternative init syntax worked"
    else
      echo "ERROR: Alternative init also failed"
      echo "DEBUG: Checking switchlynode init help..."
      switchlynode init --help | head -10
      exit 1
    fi
  fi
  
  echo "DEBUG: About to list keys"
  if echo "$SIGNER_PASSWD" | switchlynode keys list --keyring-backend file 2>/dev/null; then
    echo "DEBUG: Keys list successful"
  else
    echo "DEBUG: No keys found or keys list failed (this is expected for fresh init)"
  fi
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
  NODE_PUB_KEY="$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" --pubkey --keyring-backend file | switchlynode pubkey)"
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
  # Try to get version from query, if it fails use a default
  VERSION=$(switchlynode version --output json 2>/dev/null || echo "3.7.0")
  echo "$VERSION"
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
      echo -n "$SIGNER_SEED_PHRASE" > /tmp/mnemonic.txt && printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode keys --keyring-backend file add "$SIGNER_NAME" --recover --source /tmp/mnemonic.txt
      NODE_PUB_KEY_ED25519=$(printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE" | switchlynode ed25519)
    else
      printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode keys --keyring-backend file add "$SIGNER_NAME"
      NODE_PUB_KEY_ED25519="$(printf "%s\n" "$SIGNER_PASSWD" | switchlynode ed25519)"
    fi
    export NODE_PUB_KEY_ED25519
  fi
}

set_bond_module() {
  if [ "$NET" = "mocknet" ]; then
    BOND_MODULE_ADDR="tswitch17gw75axcnr8747pkanye45pnrwk7p9c3apgv77"
  elif [ "$NET" = "stagenet" ]; then
    BOND_MODULE_ADDR="sswitch17gw75axcnr8747pkanye45pnrwk7p9c3ve0wxj"
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
      "coins": [ { "denom": "switch", "amount": "30000000000000" } ]
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

set_xlm_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchly.chain_contracts += [{"chain": "XLM", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}