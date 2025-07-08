#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"
export SIGNER_SEED_PHRASE="${SIGNER_SEED_PHRASE:=dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog dog fossil}"

# set chain defaults
export CHAIN_ID="${CHAIN_ID:=switchlynode}"
export CHAIN_HOME_FOLDER="${CHAIN_HOME_FOLDER:=.switchlynode}"

# default ulimit is set too low for switchlyprotocol in some environments
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
  jq --arg IP_ADDRESS "$IP_ADDRESS" --arg VERSION "$VERSION" --arg BOND_ADDRESS "$BOND_ADDRESS" --arg VALIDATOR "$VALIDATOR" --arg NODE_ADDRESS "$NODE_ADDRESS" --arg NODE_PUB_KEY "$NODE_PUB_KEY" --arg NODE_PUB_KEY_ED25519 "$NODE_PUB_KEY_ED25519" '.app_state.switchlyprotocol.node_accounts += [{"node_address": $NODE_ADDRESS, "version": $VERSION, "ip_address": $IP_ADDRESS, "status": "Active","bond":"30000000000000", "active_block_height": "0", "bond_address":$BOND_ADDRESS, "signer_membership": [], "validator_cons_pub_key":$VALIDATOR, "pub_key_set":{"secp256k1":$NODE_PUB_KEY,"ed25519":$NODE_PUB_KEY_ED25519}}]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
  if [ -n "$MEMBERSHIP" ]; then
    jq --arg MEMBERSHIP "$MEMBERSHIP" '.app_state.switchlyprotocol.node_accounts[-1].signer_membership += [$MEMBERSHIP]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
    mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
  fi
}

add_account() {
  ADDRESS=${1:=swtc1z63f3mzwv3g75az80xwmhrawdqcjpaek5l3xv6}
  AMOUNT=${2:=100000000000000000000}
  echo "Adding account: $ADDRESS with $AMOUNT swtc"
  switchlynode add-genesis-account "$ADDRESS" "$AMOUNT"swtc
}

reserve() {
  RESERVE_AMOUNT=${1:=22000000000000000}
  
  echo "Reserving $RESERVE_AMOUNT swtc..."
  switchlynode tx bank send "$SIGNER_NAME" swtc1dheycdevq39qlkxs2a6wuuzyn4aqxhve4qxtxt "$RESERVE_AMOUNT"swtc \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

disable_bank_send() {
  echo "Disabling bank send..."
  switchlynode tx bank disable-send \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

# inits a switchlyprotocol with the provided list of genesis accounts
init_chain() {
  echo "switchlynode init"
  switchlynode init local --chain-id "$CHAIN_ID" --default-denom swtc 2>&1 | grep -v "already initialized"
  echo "switchlynode render-config"
  switchlynode render-config
}

fetch_node_id() {
  until curl -s "$1:$PORT_RPC" 1>/dev/null 2>&1; do
    sleep 3
  done
  curl -s "$1:$PORT_RPC/status" | jq -r .result.node_info.id
}

set_node_keys() {
  echo "Setting node keys..."
  printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode keys add switchlyprotocol --keyring-backend file --recover <<< "$SIGNER_SEED_PHRASE"
}

set_ip_address() {
  echo "Setting IP address..."
  switchlynode tx switchlynode set-ip-address $(curl -s http://whatismyip.akamai.com) \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

set_node_account() {
  echo "Setting node account..."
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
  NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -p --keyring-backend file)
  VALIDATOR=$(switchlynode tendermint show-validator)
  
  switchlynode tx switchlynode set-node-account "$NODE_ADDRESS" "$NODE_PUB_KEY" "$VALIDATOR" \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

set_asgard_address() {
  POOL_ADDRESS=${1:=swtc1g98cy3n9mmjrpn0sxmn63lztelera37nrytwp2}
  echo "Setting asgard address: $POOL_ADDRESS"
  switchlynode tx switchlynode set-asgard-address "$POOL_ADDRESS" \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

ban_address() {
  BAN_ADDRESS=${1:=swtc1wz78qmrkplrdhy37tw0tnvn0tkm5pqd6zdp257}
  echo "Banning address: $BAN_ADDRESS"
  switchlynode tx switchlynode ban "$BAN_ADDRESS" \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

set_version() {
  echo "Setting version..."
  switchlynode tx switchlynode set-version \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

create_thor_user() {
  USER=${1:=$SIGNER_NAME}
  PASS=${2:=$SIGNER_PASSWD}
  SEED_PHRASE=${3:=$SIGNER_SEED_PHRASE}
  
  echo "Creating user: $USER"
  printf "%s\n%s\n" "$PASS" "$PASS" | switchlynode keys add "$USER" --keyring-backend file --recover <<< "$SEED_PHRASE" 2>&1 | grep -v "already exists"
}

set_bond_module() {
  if [ "$NET" = "mocknet" ]; then
    BOND_MODULE_ADDR="swtc17gw75axcnr8747pkanye45pnrwk7p9c3uhzgff"
  elif [ "$NET" = "stagenet" ]; then
    BOND_MODULE_ADDR="sswtc17gw75axcnr8747pkanye45pnrwk7p9c3ve0wxj"
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
      "coins": [ { "denom": "swtc", "amount": "30000000000000" } ]
    }
  ]' <~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_eth_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchlyprotocol.chain_contracts += [{"chain": "ETH", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_avax_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchlyprotocol.chain_contracts += [{"chain": "AVAX", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_bsc_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchlyprotocol.chain_contracts += [{"chain": "BSC", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_base_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchlyprotocol.chain_contracts = [{"chain": "BASE", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_xlm_contract() {
  jq --arg CONTRACT "$1" '.app_state.switchlyprotocol.chain_contracts += [{"chain": "XLM", "router": $CONTRACT}]' ~/.switchlynode/config/genesis.json >/tmp/genesis.json
  mv /tmp/genesis.json ~/.switchlynode/config/genesis.json
}

set_gas_config() {
  echo "Setting gas config..."
  switchlynode tx switchlynode set-gas-config \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --chain-id "$CHAIN_ID" \
    --yes
}

if [ "$NET" = "mocknet" ]; then
  echo "Loading unsafe init for mocknet..."
  . "$(dirname "$0")/core-unsafe.sh"
fi
