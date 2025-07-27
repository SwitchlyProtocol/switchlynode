#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchly}"
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
  switchlynode config set client chain-id "$CHAIN_ID" --home ~/.switchlynode
  create_switchly_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

  VALIDATOR=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)
  NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
  NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -p --keyring-backend file | switchlynode pubkey)
  VERSION=$(fetch_version)

  NODE_IP_ADDRESS=${EXTERNAL_IP:=$(curl -s http://whatismyip.akamai.com)}
  add_node_account "$NODE_ADDRESS" "$VALIDATOR" "$NODE_PUB_KEY" "$VERSION" "$NODE_ADDRESS" "$NODE_PUB_KEY_ED25519" "$NODE_IP_ADDRESS"
  add_account "$NODE_ADDRESS" switch 100000000000

  # disable default bank transfer, and opt to use our own custom one
  disable_bank_send

  # for mocknet, add initial balances
  echo "Using NET $NET"
  if [ "$NET" = "mocknet" ]; then
    echo "Setting up accounts"

    # Setup accounts in mocknet
    add_account tswitch1c9vr9yvtratlqwhyua24zfpet2zmjvrlvnspg6 switch 5000000000000
    add_account tswitch126g33q5a7f0exms8qrz46qmu6hlrwc32jxgzxp switch 25000000000100
    add_account tswitch18s83m747vmre2ljw45t7dlmju96zmt03humlvj switch 25000000000100
    add_account tswitch1tyry86qensp4ws4enudkrjrw408x559ghwe7xy switch 5090000000000

    # Liquidity Provider accounts (one for each chain)
    add_account tswitch1ugursz23jha99zeglv9qww8gaxcgz2xjq4qcp9 switch 200000000000000 # cat
    add_account tswitch1swe4u2gw9vlze677fgseyka0tu0zgmp2wkdktk switch 200000000000000 # dog
    add_account tswitch1exgvymgkqf4q7rd9epnpk0t6f3z4z9udkrzane switch 200000000000000 # fox
    add_account tswitch1pc52j8mzkhp0ywya2vtqhkatkqumpn6h9t2rec switch 200000000000000 # pig

    # User accounts for smoke tests
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
  switchlynode validate-genesis
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

  # validate genesis
  echo "Genesis validation:"
  switchlynode genesis validate ~/.switchlynode/config/genesis.json
  
  echo "DEBUG: About to exec with arguments: $@"
  echo "DEBUG: Number of arguments: $#"
  echo "DEBUG: First argument: $1"
  echo "DEBUG: Second argument: ${2:-NONE}"
  exec "$@"
