#!/bin/bash

set -euo pipefail

# set node CLI configuration
switchlynode config chain-id "$CHAIN_ID"
switchlynode config keyring-backend file

# set defaults
PEER="${PEER:=none}"
if [[ "$PEER" == "none" ]]; then
    echo "Missing PEER"
    exit 1
fi

switchlynode render-config

exec switchlynode start
