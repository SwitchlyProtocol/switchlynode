#!/bin/bash

set -o pipefail

. "$(dirname "$0")/core.sh"

if [ "$NET" = "mocknet" ]; then
  echo "Loading unsafe init for mocknet..."
  . "$(dirname "$0")/core-unsafe.sh"
fi

########################################################################################
# Genesis Init
########################################################################################

genesis_init() {
  init_chain
  create_switchly_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

  VALIDASWITCHLY=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
  NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -p --keyring-backend file | switchlynode pubkey)
  VERSION=$(fetch_version)

  # For mocknet genesis node, use proper IP detection
  if [ "$NET" = "mocknet" ] && [ -n "$EXTERNAL_IP" ]; then
    NODE_IP_ADDRESS="$EXTERNAL_IP"
    echo "Using preset mocknet IP: $NODE_IP_ADDRESS"
  elif [ "$NET" = "mocknet" ]; then
    # Docker container IP detection for genesis node
    if command -v hostname >/dev/null 2>&1; then
      DOCKER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' | head -n1)"
      if [ -n "$DOCKER_IP" ] && [ "$DOCKER_IP" != "127.0.0.1" ]; then
        NODE_IP_ADDRESS="$DOCKER_IP"
      else
        NODE_IP_ADDRESS="$(hostname -i 2>/dev/null || echo '172.18.0.6')"
      fi
    else
      NODE_IP_ADDRESS="172.18.0.6"
    fi
    echo "Detected mocknet genesis IP: $NODE_IP_ADDRESS"
  else
    NODE_IP_ADDRESS=$(curl -s http://whatismyip.akamai.com)
    echo "Using external IP: $NODE_IP_ADDRESS"
  fi
  add_node_account "$NODE_ADDRESS" "$VALIDASWITCHLY" "$NODE_PUB_KEY" "$VERSION" "$NODE_ADDRESS" "$NODE_PUB_KEY_ED25519" "$NODE_IP_ADDRESS"
  add_account "$NODE_ADDRESS" switch 100000000000

  # disable default bank transfer, and opt to use our own custom one
  disable_bank_send

  # for mocknet, add initial balances
  echo "Using NET $NET"
  if [ "$NET" = "mocknet" ]; then
    echo "Setting up accounts"

    # smoke test accounts
    add_account tswitch1c9vr9yvtratlqwhyua24zfpet2zmjvrlvnspg6 switch 5000000000000
    add_account tswitch126g33q5a7f0exms8qrz46qmu6hlrwc32jxgzxp switch 25000000000100
    add_account tswitch18s83m747vmre2ljw45t7dlmju96zmt03humlvj switch 25000000000100
    add_account tswitch1tyry86qensp4ws4enudkrjrw408x559ghwe7xy switch 5090000000000

    # local cluster accounts (2M SWITCH)
    add_account tswitch1uuds8pd92qnnq0udw0rpg0szpgcslc9pxf4cw9 switch 200000000000000 # cat
    add_account tswitch1swe4u2gw9vlze677fgseyka0tu0zgmp2wkdktk switch 200000000000000 # dog
    add_account tswitch13wrmhnh2qe98rjse30pl7u6jxszjjwl4gvwq05 switch 200000000000000 # fox
    add_account tswitch1qk8c8sfrmfm0tkncs0zxeutc8v5mx3pjne5jzt switch 200000000000000 # pig

    # simulation master
    add_account tswitch1vny3kasx3nlh6h8h4w2nw92clj8hncg2v27duw switch 100000000000000 # master

    # mint to reserve for mocknet
    reserve 22000000000000000

    set_bond_module # set bond module balance for invariant

    # deploy evm contracts
    deploy_evm_contracts
  fi

  if [ "$NET" = "stagenet" ]; then
    if [ -z ${FAUCET+x} ]; then
      echo "env variable 'FAUCET' is not defined: should be a sswitch address"
      exit 1
    fi
    add_account "$FAUCET" switch 40000000000000000

    reserve 5000000000000000

    set_bond_module # set bond module balance for invariant
  fi

  if [ -n "${ETH_CONTRACT+x}" ]; then
    echo "ETH Contract Address: $ETH_CONTRACT"
    set_eth_contract "$ETH_CONTRACT"
  fi
  if [ -n "${AVAX_CONTRACT+x}" ]; then
    echo "AVAX Contract Address: $AVAX_CONTRACT"
    set_avax_contract "$AVAX_CONTRACT"
  fi
  if [ -n "${BSC_CONTRACT+x}" ]; then
    echo "BSC Contract Address: $BSC_CONTRACT"
    set_bsc_contract "$BSC_CONTRACT"
  fi
  if [ -n "${BASE_CONTRACT+x}" ]; then
    echo "BASE Contract Address: $BASE_CONTRACT"
    set_base_contract "$BASE_CONTRACT"
  fi
  if [ -n "${XLM_CONTRACT+x}" ]; then
    echo "XLM Contract Address: $XLM_CONTRACT"
    set_xlm_contract "$XLM_CONTRACT"
  fi

  echo "Genesis content"
  cat ~/.switchlynode/config/genesis.json
  echo "Validating genesis file..."
  switchlynode genesis validate --trace
  echo "Genesis validation complete"
  echo "Node accounts in genesis:"
  cat ~/.switchlynode/config/genesis.json | jq '.app_state.switchly.node_accounts | length'
}

########################################################################################
# Main
########################################################################################

# genesis on first init if we are the genesis node
if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  echo "Creating genesis file as genesis node..."
  genesis_init
  echo "Genesis file created successfully"
else
  echo "Genesis file already exists, checking content..."
  echo "Node accounts in existing genesis:"
  cat ~/.switchlynode/config/genesis.json | jq '.app_state.switchly.node_accounts | length'
fi

# Ensure genesis file exists and is valid before starting
if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  echo "FATAL: Genesis file does not exist after initialization"
  exit 1
fi

echo "Final genesis validation before chain start..."
switchlynode genesis validate --trace
echo "Starting switchlynode with genesis file containing $(cat ~/.switchlynode/config/genesis.json | jq '.app_state.switchly.node_accounts | length') node accounts"

# Force complete chain reset to ensure genesis is loaded properly
# This is the nuclear option but guaranteed to work
rm -rf ~/.switchlynode/data/
rm -rf ~/.switchlynode/application.db

# Create the missing priv_validator_state.json file that CometBFT needs
mkdir -p ~/.switchlynode/data
cat > ~/.switchlynode/data/priv_validator_state.json << EOF
{
  "height": "0",
  "round": 0,
  "step": 0
}
EOF

# Force genesis file reload by updating genesis time AND chain_id
# This ensures Cosmos SDK treats this as a completely new chain
CURRENT_TIME=$(date -u +"%Y-%m-%dT%H:%M:%S.%6NZ")
CHAIN_ID="switchly-$(date +%s)"
jq --arg time "$CURRENT_TIME" --arg chain_id "$CHAIN_ID" '.genesis_time = $time | .chain_id = $chain_id' ~/.switchlynode/config/genesis.json > /tmp/genesis_fixed.json
mv /tmp/genesis_fixed.json ~/.switchlynode/config/genesis.json

echo "Updated genesis time and chain_id to force complete reload. Final node account count:"
cat ~/.switchlynode/config/genesis.json | jq '.app_state.switchly.node_accounts | length'

# Ensure we also clear any cosmos app database
rm -rf ~/.switchlynode/wasm

# Log genesis content for debugging
echo "Genesis content verification:"
cat ~/.switchlynode/config/genesis.json | jq '.app_state.switchly.node_accounts[0].status' 2>/dev/null || echo "No node accounts found"

# Ensure the genesis state is consistent across all blockchain components
echo "Backing up genesis file for reference..."
cp ~/.switchlynode/config/genesis.json /tmp/genesis_backup.json

# render tendermint and cosmos configuration files
switchlynode render-config

# Double-check that the genesis file wasn't overwritten by render-config
if ! diff -q ~/.switchlynode/config/genesis.json /tmp/genesis_backup.json > /dev/null; then
  echo "WARNING: render-config modified genesis file, restoring..."
  cp /tmp/genesis_backup.json ~/.switchlynode/config/genesis.json
fi

echo "Final verification - starting with genesis containing:"
cat ~/.switchlynode/config/genesis.json | jq '.app_state.switchly.node_accounts | length'

export SIGNER_NAME
export SIGNER_PASSWD
exec switchlynode start --home /root/.switchlynode
