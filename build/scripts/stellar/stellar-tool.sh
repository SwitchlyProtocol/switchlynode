#!/bin/bash
# stellar-tool.sh — deploy the Soroban router + native XLM SAC to the LOCAL (standalone)
# Stellar network used by mocknet, and print the router contract id. This is the Stellar
# analog of build/scripts/evm/evm-tool.py for the EVM chains.
#
# Usage: stellar-tool.sh deploy <rpc_host>      e.g. stellar-tool.sh deploy http://stellar:8000
#
# On success it prints a line "Router Contract Address: C..." (grepped by deploy_stellar_contract).

set -o pipefail

ACTION="${1:-deploy}"
RPC_HOST="${2:-http://stellar:8000}"
WASM="${STELLAR_ROUTER_WASM:-/scripts/stellar/switchly_router.wasm}"
DEPLOYER="${STELLAR_DEPLOYER:-switchly-mock-deployer}"

SOROBAN_RPC="${RPC_HOST%/}/soroban/rpc"

log() { echo "[stellar-tool] $*" >&2; }

# Wait for the local Stellar Horizon to be up and report its network passphrase.
wait_for_stellar() {
  for i in $(seq 1 60); do
    PASSPHRASE=$(curl -fsS "$RPC_HOST" 2>/dev/null | jq -r '.network_passphrase // empty')
    if [ -n "$PASSPHRASE" ]; then
      log "stellar ready; network passphrase: $PASSPHRASE"
      return 0
    fi
    log "waiting for stellar at $RPC_HOST ($i/60)..."
    sleep 5
  done
  log "FATAL: stellar not ready at $RPC_HOST"
  return 1
}

fund() {
  # Fund an account on the local network via friendbot.
  local addr="$1"
  for i in $(seq 1 10); do
    if curl -fsS "${RPC_HOST%/}/friendbot?addr=${addr}" >/dev/null 2>&1; then
      log "funded $addr"
      return 0
    fi
    sleep 3
  done
  log "FATAL: friendbot funding failed for $addr"
  return 1
}

# Wait for the Soroban RPC to report healthy.
wait_for_soroban() {
  for i in $(seq 1 60); do
    local st
    st=$(curl -fsS -X POST "$SOROBAN_RPC" -H 'Content-Type: application/json' \
      -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}' 2>/dev/null | jq -r '.result.status // empty' 2>/dev/null)
    if [ "$st" = "healthy" ]; then
      log "soroban rpc healthy"
      return 0
    fi
    log "waiting for soroban rpc at $SOROBAN_RPC ($i/60)..."
    sleep 5
  done
  log "FATAL: soroban rpc not healthy at $SOROBAN_RPC"
  return 1
}

deploy() {
  wait_for_stellar || exit 1
  wait_for_soroban || exit 1

  # Create (or reuse) the deployer identity and fund it.
  stellar keys generate "$DEPLOYER" --overwrite >/dev/null 2>&1 || true
  DEPLOYER_ADDR=$(stellar keys address "$DEPLOYER" 2>/dev/null)
  if [ -z "$DEPLOYER_ADDR" ]; then
    log "FATAL: could not derive deployer address"
    exit 1
  fi
  log "deployer: $DEPLOYER_ADDR"
  fund "$DEPLOYER_ADDR" || exit 1

  # The standalone network applies its Soroban config-settings (the --limits upgrade) a short
  # while AFTER the RPC reports healthy; contract deploys fail until then, so retry with backoff.
  local CONTRACT=""
  for attempt in $(seq 1 24); do
    # Ensure the native XLM SAC exists (idempotent — ignore "already exists").
    stellar contract asset deploy --asset native --source "$DEPLOYER" \
      --rpc-url "$SOROBAN_RPC" --network-passphrase "$PASSPHRASE" >/dev/null 2>/tmp/sac.err || true

    CONTRACT=$(stellar contract deploy --wasm "$WASM" --source "$DEPLOYER" \
      --rpc-url "$SOROBAN_RPC" --network-passphrase "$PASSPHRASE" 2>/tmp/router.err)
    if [ -n "$CONTRACT" ]; then
      break
    fi
    log "deploy not ready (attempt $attempt/24), waiting for soroban config-settings to apply..."
    sleep 10
  done

  if [ -z "$CONTRACT" ]; then
    log "FATAL: router deploy failed after retries"
    cat /tmp/router.err >&2
    exit 1
  fi

  # Sanity check: read version().
  stellar contract invoke --id "$CONTRACT" --source "$DEPLOYER" \
    --rpc-url "$SOROBAN_RPC" --network-passphrase "$PASSPHRASE" -- version >&2 2>/dev/null || true

  # This line is parsed by deploy_stellar_contract (mirrors evm-tool's output).
  echo "Router Contract Address: $CONTRACT"
}

case "$ACTION" in
deploy) deploy ;;
*)
  log "unknown action: $ACTION"
  exit 1
  ;;
esac
