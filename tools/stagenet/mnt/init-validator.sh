#!/bin/sh

# set version
printf "%s\n%s\n" "password" "password" | switchlynode tx switchly set-version --from switchly --keyring-backend file --chain-id "$CHAIN_ID" --node http://localhost:27147 --yes

# set node keys
NODE_PUB_KEY=$(echo "password" | switchlynode keys show switchly --pubkey --keyring-backend file | switchlynode pubkey)
NODE_PUB_KEY_ED25519=$(echo "password" | switchlynode ed25519)
VALIDATOR=$(switchlynode tendermint show-validator | switchlynode pubkey --bech cons)
printf "%s\n%s\n" "password" "password" | switchlynode tx switchly set-node-keys "$NODE_PUB_KEY" "$NODE_PUB_KEY_ED25519" "$VALIDATOR" --from switchly --keyring-backend file --chain-id "$CHAIN_ID" --node http://localhost:27147 --yes

# set node ip
printf "%s\n%s\n" "password" "password" | thornode tx thorchain set-ip-address "$(hostname -i)" --from thorchain --keyring-backend file --chain-id "$CHAIN_ID" --node http://localhost:27147 --yes
