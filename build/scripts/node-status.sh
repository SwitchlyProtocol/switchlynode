#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlyprotocol}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

. "$(dirname "$0")/core.sh"

# set defaults
CHAIN_API="${CHAIN_API:=127.0.0.1:1317}"
CHAIN_RPC="${CHAIN_RPC:=127.0.0.1:26657}"
PEER="${PEER:=127.0.0.1}"
FAUCET="${FAUCET:=swtc1v8vkkymvhe2sf7gd2092ujc6hweta38xjc5jt6}"

API=http://switchlyprotocol:1317
SWITCHLYNODE_PORT="${SWITCHLYNODE_SERVICE_PORT_RPC:-27147}"
RPC=http://switchlyprotocol:${SWITCHLYNODE_PORT}

# wait for switchlynode to start
sleep 10

# create the user if it doesn't exist
create_thor_user "$SIGNER_NAME" "$SIGNER_PASSWD" "$SIGNER_SEED_PHRASE"

# get our node account
ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
JSON=$(curl -s "$API/cosmos/auth/v1beta1/accounts/$ADDRESS")
if [ ".switchlynode" = "$(echo "$JSON" | jq -r .node_address)" ]; then
  echo "switchlynode address not found"
  exit 1
fi

IP=$(echo "$JSON" | jq -r .ip_address)
VERSION=$(echo "$JSON" | jq -r .version)
BOND=$(echo "$JSON" | jq -r .bond)
REWARDS=$(echo "$JSON" | jq -r .current_award)
SLASH=$(echo "$JSON" | jq -r .slash_points)
STATUS=$(echo "$JSON" | jq -r .status)
PREFLIGHT=$(echo "$JSON" | jq -r .preflight_status)

# get BTC block height
BITCOIN_ENDPOINT="${BITCOIN_ENDPOINT:=bitcoin:18443}"
if [ -n "$BITCOIN_ENDPOINT" ]; then
  BTC_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockcount", "params": []}' -H 'content-type: text/plain;' http://switchlynode:password@"$BITCOIN_ENDPOINT" | jq -r .result)
fi

# get LTC block height
LITECOIN_ENDPOINT="${LITECOIN_ENDPOINT:=litecoin:38443}"
if [ -n "$LITECOIN_ENDPOINT" ]; then
  LTC_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockcount", "params": []}' -H 'content-type: text/plain;' http://switchlynode:password@"$LITECOIN_ENDPOINT" | jq -r .result)
fi

# get ETH block height
ETH_HOST="${ETH_HOST:=http://ethereum:8545}"
if [ -n "$ETH_HOST" ]; then
  ETH_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "2.0", "id": "node-status", "method": "eth_blockNumber", "params": []}' -H 'content-type: application/json;' "$ETH_HOST" | jq -r .result)
  ETH_HEIGHT=$((ETH_HEIGHT))
fi

# get AVAX block height
AVAX_HOST="${AVAX_HOST:=http://avalanche:9650/ext/bc/C/rpc}"
if [ -n "$AVAX_HOST" ]; then
  AVAX_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "2.0", "id": "node-status", "method": "eth_blockNumber", "params": []}' -H 'content-type: application/json;' "$AVAX_HOST" | jq -r .result)
  AVAX_HEIGHT=$((AVAX_HEIGHT))
fi

# get BSC block height
BSC_HOST="${BSC_HOST:=http://binance-smart:8545}"
if [ -n "$BSC_HOST" ]; then
  BSC_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "2.0", "id": "node-status", "method": "eth_blockNumber", "params": []}' -H 'content-type: application/json;' "$BSC_HOST" | jq -r .result)
  BSC_HEIGHT=$((BSC_HEIGHT))
fi

# get BCH block height
BITCOIN_CASH_ENDPOINT="${BITCOIN_CASH_ENDPOINT:=bitcoin-cash:28443}"
if [ -n "$BITCOIN_CASH_ENDPOINT" ]; then
  BCH_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockcount", "params": []}' -H 'content-type: text/plain;' http://switchlynode:password@"$BITCOIN_CASH_ENDPOINT" | jq -r .result)
fi

# get DOGE block height
DOGECOIN_ENDPOINT="${DOGECOIN_ENDPOINT:=dogecoin:18332}"
if [ -n "$DOGECOIN_ENDPOINT" ]; then
  DOGE_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockcount", "params": []}' -H 'content-type: text/plain;' http://switchlynode:password@"$DOGECOIN_ENDPOINT" | jq -r .result)
fi

# get GAIA block height
GAIA_HOST="${GAIA_HOST:=http://gaia:26657}"
if [ -n "$GAIA_HOST" ]; then
  GAIA_HEIGHT=$(curl -sL --fail -m 10 "$GAIA_HOST/status" | jq -r .result.sync_info.latest_block_height)
fi

# get BASE block height
BASE_HOST="${BASE_HOST:=http://base:8545}"
if [ -n "$BASE_HOST" ]; then
  BASE_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "2.0", "id": "node-status", "method": "eth_blockNumber", "params": []}' -H 'content-type: application/json;' "$BASE_HOST" | jq -r .result)
  BASE_HEIGHT=$((BASE_HEIGHT))
fi

# get XRP block height
XRP_HOST="${XRP_HOST:=http://xrp:5005}"
if [ -n "$XRP_HOST" ]; then
  XRP_HEIGHT=$(curl -sL --fail -m 10 --data-binary '{"method": "ledger", "params": [{"ledger_index": "validated"}]}' -H 'content-type: application/json;' "$XRP_HOST" | jq -r .result.ledger.ledger_index)
fi

# get XLM block height
XLM_HOST="${XLM_HOST:=http://stellar:8000}"
if [ -n "$XLM_HOST" ]; then
  XLM_HEIGHT=$(curl -sL --fail -m 10 "$XLM_HOST/ledgers?order=desc&limit=1" | jq -r ._embedded.records[0].sequence)
fi

# get switchlynode block height
if [ "$PEER" = "127.0.0.1" ]; then
  SWITCHLYNODE_SYNC_HEIGHT=$(curl -sL --fail -m 10 switchlynode:"$SWITCHLYNODE_PORT"/status | jq -r ".result.sync_info.latest_block_height")
else
  # get peer height
  SWITCHLYNODE_HEIGHT=$(curl -sL --fail -m 10 "$PEER:$SWITCHLYNODE_PORT/status" | jq -r ".result.sync_info.latest_block_height")
  
  # get our height
  while true; do
    SWITCHLYNODE_HEIGHT=$(curl -sL --fail -m 10 "$PEER:$SWITCHLYNODE_PORT/status" | jq -r ".result.sync_info.latest_block_height") || continue
    break
  done
fi

# get node IP address
if [ -z "$IP" ]; then
  IP=$(curl -sL --fail -m 10 http://whatismyip.akamai.com)
fi

# get node IP address
if [ -z "$IP" ]; then
  IP="127.0.0.1"
fi

# get node IP address
if [ -z "$VERSION" ]; then
  VERSION="unknown"
fi

# get node IP address
if [ -z "$BOND" ]; then
  BOND="0"
fi

# get node IP address
if [ -z "$REWARDS" ]; then
  REWARDS="0"
fi

# get node IP address
if [ -z "$SLASH" ]; then
  SLASH="0"
fi

# get node IP address
if [ -z "$STATUS" ]; then
  STATUS="unknown"
fi

# get node IP address
if [ -z "$PREFLIGHT" ]; then
  PREFLIGHT="unknown"
fi

# print node status
echo "API         http://$IP:1317/switchlynode/doc/"
echo "RPC         http://$IP:$SWITCHLYNODE_PORT"
echo "ADDRESS     $ADDRESS"
echo "IP          $IP"
echo "VERSION     $VERSION"
echo "STATUS      $STATUS"
echo "BOND        $BOND"
echo "REWARDS     $REWARDS"
echo "SLASH       $SLASH"
echo "PREFLIGHT   $PREFLIGHT"
echo ""

# print chain heights
echo "CHAIN HEIGHTS"
echo "BTC         $BTC_HEIGHT"
echo "LTC         $LTC_HEIGHT"
echo "ETH         $ETH_HEIGHT"
echo "AVAX        $AVAX_HEIGHT"
echo "BSC         $BSC_HEIGHT"
echo "BCH         $BCH_HEIGHT"
echo "DOGE        $DOGE_HEIGHT"
echo "GAIA        $GAIA_HEIGHT"
echo "BASE        $BASE_HEIGHT"
echo "XRP         $XRP_HEIGHT"
echo "XLM         $XLM_HEIGHT"
echo "SWITCHLYNODE $SWITCHLYNODE_HEIGHT"
echo ""

# print node status
echo "NODE STATUS"
echo "ADDRESS     $ADDRESS"
echo "IP          $IP"
echo "VERSION     $VERSION"
echo "STATUS      $STATUS"
echo "BOND        $BOND"
echo "REWARDS     $REWARDS"
echo "SLASH       $SLASH"
echo "PREFLIGHT   $PREFLIGHT"
echo ""

# print chain heights
echo "CHAIN HEIGHTS"
echo "BTC         $BTC_HEIGHT"
echo "LTC         $LTC_HEIGHT"
echo "ETH         $ETH_HEIGHT"
echo "AVAX        $AVAX_HEIGHT"
echo "BSC         $BSC_HEIGHT"
echo "BCH         $BCH_HEIGHT"
echo "DOGE        $DOGE_HEIGHT"
echo "GAIA        $GAIA_HEIGHT"
echo "BASE        $BASE_HEIGHT"
echo "XRP         $XRP_HEIGHT"
echo "XLM         $XLM_HEIGHT"
echo "SWITCHLYNODE $SWITCHLYNODE_HEIGHT"
echo ""

# print mimir and constants
MIMIR=$(curl -sL --fail -m 10 "$API/switchlynode/mimir")
CONSTANTS=$(curl -sL --fail -m 10 "$API/switchlynode/constants")
VAULTS=$(curl -sL --fail -m 10 "$API/switchlynode/vaults/asgard")

echo "MIMIR"
echo "$MIMIR" | jq -r 'to_entries[] | "\(.key) = \(.value)"'
echo ""

echo "CONSTANTS"
echo "$CONSTANTS" | jq -r 'to_entries[] | "\(.key) = \(.value)"'
echo ""

echo "VAULTS"
echo "$VAULTS" | jq -r '.[] | "Status: \(.status), Membership: \(.membership), Addresses: \(.addresses)"'
echo ""

# print pool info
POOLS=$(curl -sL --fail -m 10 "$API/switchlynode/pools")
echo "POOLS"
echo "$POOLS" | jq -r '.[] | "Asset: \(.asset), Status: \(.status), Balance SWTC: \(.balance_rune), Balance Asset: \(.balance_asset)"'
echo ""

# print queue info
QUEUE=$(curl -sL --fail -m 10 "$API/switchlynode/queue")
echo "QUEUE"
echo "$QUEUE" | jq -r '.swap + .outbound | length'
echo ""

# print last block info
LASTBLOCK=$(curl -sL --fail -m 10 "$API/switchlynode/lastblock")
echo "LASTBLOCK"
echo "$LASTBLOCK" | jq -r '.[] | "Chain: \(.chain), Height: \(.last_observed_in)"'
echo ""

# print network info
NETWORK=$(curl -sL --fail -m 10 "$API/switchlynode/network")
echo "NETWORK"
echo "$NETWORK" | jq -r '"Total Reserve: \(.total_reserve), Bond Reward SWTC: \(.bond_reward_rune), Total Bond Units: \(.total_bond_units)"'
echo ""

# print explorer links
if [ "$NET" = "mainnet" ]; then
  EXPLORER="https://switchlynode.net"
elif [ "$NET" = "stagenet" ]; then
  EXPLORER="https://stagenet.switchlynode.net"
fi

if [ -n "$EXPLORER" ]; then
  echo "EXPLORER"
  echo "$EXPLORER/address/$ADDRESS"
  echo ""
fi
