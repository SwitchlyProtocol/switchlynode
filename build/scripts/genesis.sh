#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

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
  create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

  VALIDATOR=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show switchlyprotocol -a --keyring-backend file)
  NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show switchlyprotocol --pubkey --keyring-backend file | switchlynode pubkey)
  NODE_PUB_KEY_ED25519=$(printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE" | switchlynode ed25519)
  VERSION=$(fetch_version)

  NODE_IP_ADDRESS=${EXTERNAL_IP:=$(curl -s http://whatismyip.akamai.com)}
  add_node_account "$NODE_ADDRESS" "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" "$BOND_ADDRESS" "$VERSION" "$NODE_IP_ADDRESS" "$MEMBERSHIP"
  add_account "$NODE_ADDRESS" swtc 100000000000

  # disable default bank transfer, and opt to use our own custom one
  disable_bank_send

  # for mocknet, add initial balances
  echo "Using NET $NET"
  if [ "$NET" = "mocknet" ]; then
    echo "Setting up accounts"

    # smoke test accounts
    add_account swtc1z63f3mzwv3g75az80xwmhrawdqcjpaek5l3xv6 swtc 500000000000000000000 # cat
    add_account swtc1xwusttz86hqfuk5z7amcgqsg7vp6g8zhsp5lu2 swtc 500000000000000000000 # dog
    add_account swtc1ragkh3hp7wvdpw3qgxr5vchqaecmupv8ql3p5y swtc 500000000000000000000 # fox
    add_account swtc1wz78qmrkplrdhy37tw0tnvn0tkm5pqd6zdp257 swtc 500000000000000000000 # pig

    # local cluster accounts (2M SWTC)
    add_account swtc1uuds8pd92qnnq0udw0rpg0szpgcslc9p8lluej swtc 200000000000000 # cat
    add_account swtc1zf3gsk7edzwl9syyefvfhle37cjtql35h6k85m swtc 200000000000000 # dog
    add_account swtc13wrmhnh2qe98rjse30pl7u6jxszjjwl4f6yycr swtc 200000000000000 # fox
    add_account swtc1qk8c8sfrmfm0tkncs0zxeutc8v5mx3pjj07k4u swtc 200000000000000 # pig

    # simulation master
    add_account swtc1f4l5dlqhaujgkxxqmug4stfvmvt58vx2tspx4g swtc 100000000000000 # master

    # mint to reserve for mocknet
    reserve 22000000000000000

    set_bond_module # set bond module balance for invariant

    # deploy evm contracts
    deploy_evm_contracts
    
    # deploy stellar contracts
    deploy_stellar_contracts
  fi

  if [ "$NET" = "stagenet" ]; then
    if [ -z ${FAUCET+x} ]; then
      echo "env variable 'FAUCET' is not defined: should be a sswtc address"
      exit 1
    fi
    add_account "$FAUCET" swtc 40000000000000000

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
  switchlynode validate-genesis
}

fetch_version() {
  switchlynode query switchlynode version --output json | jq -r .version
}

########################################################################################
# Main
########################################################################################

# genesis on first init if we are the genesis node
if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  genesis_init
fi

# render tendermint and cosmos configuration files
switchlynode render-config

export SIGNER_NAME
export SIGNER_PASSWD
exec switchlynode start
