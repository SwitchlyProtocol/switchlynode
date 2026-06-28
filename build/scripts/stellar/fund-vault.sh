#!/bin/bash
# fund-vault.sh — create + fund the Asgard XLM vault account on the local (standalone) Stellar
# network used by mocknet. The vault address is only known AFTER TSS keygen, so this polls the
# node's /inbound_addresses until the XLM vault appears, then friendbot-funds it once (a Stellar
# account must exist before it can hold/move assets). Runs as a one-shot mocknet sidecar.

set -o pipefail

API="${SWITCHLYNODE_API:-http://switchlynode:1317}"
STELLAR_HOST="${STELLAR_HOST:-http://stellar:8000}"

log() { echo "[fund-vault] $*"; }

while true; do
  ADDR=$(curl -fsS "$API/switchly/inbound_addresses" 2>/dev/null \
    | jq -r '.[] | select(.chain=="XLM") | .address' 2>/dev/null | head -n1)

  if [ -n "$ADDR" ] && [ "$ADDR" != "null" ]; then
    if curl -fsS "${STELLAR_HOST%/}/accounts/$ADDR" >/dev/null 2>&1; then
      log "XLM vault $ADDR already exists on the local network"
      break
    fi
    log "funding XLM vault $ADDR via friendbot"
    if curl -fsS "${STELLAR_HOST%/}/friendbot?addr=$ADDR" >/dev/null 2>&1; then
      log "XLM vault funded"
      break
    fi
    log "friendbot funding failed, retrying..."
  else
    log "waiting for XLM vault to appear in /inbound_addresses (post-keygen)..."
  fi
  sleep 10
done

log "done"
