#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

. "$(dirname "$0")/core.sh"

if [ "$NET" = "mocknet" ]; then
  echo "Loading unsafe init for mocknet..."
  . "$(dirname "$0")/core-unsafe.sh"
fi

init_chain() {
  echo "switchlynode init"
  switchlynode init local --chain-id "$CHAIN_ID" --default-denom swtc 2>&1 | grep -v "already initialized"
  echo "switchlynode render-config"
  switchlynode render-config
}

if [ ! -f ~/.switchlynode/config/genesis.json ]; then
  init_chain
fi

create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

NODE_ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
NODE_PUB_KEY=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -p --keyring-backend file)
VALIDATOR=$(switchlynode tendermint show-validator)

echo "Setting node keys..."
printf "%s\n%s\n" "$SIGNER_PASSWD" "$SIGNER_PASSWD" | switchlynode keys add switchlyprotocol --keyring-backend file --recover <<< "$SIGNER_SEED_PHRASE"

echo "Creating validator..."
if [ "$NET" = "mocknet" ]; then
  switchlynode tx staking create-validator \
    --amount 100000000000000000000swtc \
    --commission-max-change-rate "0.05" \
    --commission-max-rate "0.10" \
    --commission-rate "0.05" \
    --min-self-delegation "1" \
    --details "SwitchlyProtocol Validator" \
    --pubkey "$VALIDATOR" \
    --moniker "$SIGNER_NAME" \
    --chain-id "$CHAIN_ID" \
    --gas auto \
    --gas-adjustment 1.5 \
    --gas-prices 0.02swtc \
    --from "$SIGNER_NAME" \
    --keyring-backend file \
    --yes
fi

exec switchlynode start
