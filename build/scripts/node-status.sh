#!/bin/bash

set -o pipefail
format_1e8() {
  printf "%.2f\n" "$(jq -n "$1"/100000000 2>/dev/null)" 2>/dev/null | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta'
}

format_int() {
  printf "%.0f\n" "$1" 2>/dev/null | sed ':a;s/\B[0-9]\{3\}\>/,&/;ta'
}

calc_progress() {
  if [ "$1" = "$2" ]; then
    [ "$1" = "0" ] && echo "0.000%" || echo "100.000%"
  elif [ -n "$3" ]; then
    progress="$(echo "$3 100" | awk '{printf "%.6f", $1 * $2}' 2>/dev/null)" && printf "%.3f%%" "$progress" || echo "Error"
  else
    progress="$(echo "$1 $2" | awk '{printf "%.6f", ($1/$2) * 100}' 2>/dev/null)" && printf "%.3f%%" "$progress" || echo "Error"
  fi
}

API=http://switchlynode:1317
SWITCHLYNODE_PORT="${SWITCHLYNODE_SERVICE_PORT_RPC:-27147}"
RPC=http://switchlynode:${SWITCHLYNODE_PORT}

BITCOIN_ENDPOINT="${BTC_HOST:-bitcoin-daemon:${BITCOIN_DAEMON_SERVICE_PORT_RPC:-8332}}"
LITECOIN_ENDPOINT="${LTC_HOST:-litecoin-daemon:${LITECOIN_DAEMON_SERVICE_PORT_RPC:-9332}}"
BITCOIN_CASH_ENDPOINT="${BCH_HOST:-bitcoin-cash-daemon:${BITCOIN_CASH_DAEMON_SERVICE_PORT_RPC:-8332}}"
DOGECOIN_ENDPOINT="${DOGE_HOST:-dogecoin-daemon:${DOGECOIN_DAEMON_SERVICE_PORT_RPC:-22555}}"
ETHEREUM_ENDPOINT="${ETH_HOST:-http://ethereum-daemon:${ETHEREUM_DAEMON_SERVICE_PORT_RPC:-8545}}"
# trunk-ignore(shellcheck/SC2001)
ETHEREUM_BEACON_ENDPOINT=$(echo "$ETHEREUM_ENDPOINT" | sed 's/:[0-9]*$/:3500/g')
BINANCE_SMART_ENDPOINT="${BIFROST_CHAINS_BSC_RPC_HOST:-http://binance-smart-daemon:${BINANCE_SMART_DAEMON_SERVICE_PORT_RPC:-8545}}"
BASE_ENDPOINT="${BIFROST_CHAINS_BASE_RPC_HOST:-http://base-daemon:${BASE_DAEMON_SERVICE_PORT_RPC:-8545}}"
GAIA_ENDPOINT="${GAIA_HOST:-http://gaia-daemon:26657}"
AVALANCHE_ENDPOINT="${AVAX_HOST:-http://avalanche-daemon:9650/ext/bc/C/rpc}"
XRP_ENDPOINT="${BIFROST_CHAINS_XRP_RPC_HOST:-http://xrp-daemon:${XRP_DAEMON_SERVICE_PORT_RPC:-51234}}"

ADDRESS=$(echo "$SIGNER_PASSWD" | switchlynode keys show "$SIGNER_NAME" -a --keyring-backend file)
JSON=$(curl -sL --fail -m 10 "$API/switchly/node/$ADDRESS")

IP=$(echo "$JSON" | jq -r ".ip_address")
VERSION=$(echo "$JSON" | jq -r ".version")
BOND=$(echo "$JSON" | jq -r ".total_bond")
REWARDS=$(echo "$JSON" | jq -r ".current_award")
SLASH=$(echo "$JSON" | jq -r ".slash_points")
STATUS=$(echo "$JSON" | jq -r ".status")
PREFLIGHT=$(echo "$JSON" | jq -r ".preflight_status")
PUB_KEY=$(echo "$JSON" | jq -r ".pub_key_set.secp256k1")
[ "$VALIDATOR" = "false" ] && IP=$EXTERNAL_IP

if [ "$VALIDATOR" = "true" ]; then
  # calculate BTC chain sync progress
  BTC_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://switchly:password@"$BITCOIN_ENDPOINT")
  BTC_HEIGHT=$(echo "$BTC_RESULT" | jq -r ".result.headers")
  BTC_SYNC_HEIGHT=$(echo "$BTC_RESULT" | jq -r ".result.blocks")
  BTC_PROGRESS=$(echo "$BTC_RESULT" | jq -r ".result.verificationprogress")
  BTC_PROGRESS=$(calc_progress "$BTC_SYNC_HEIGHT" "$BTC_HEIGHT" "$BTC_PROGRESS")

  # calculate LTC chain sync progress
  LTC_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://switchly:password@"$LITECOIN_ENDPOINT")
  LTC_HEIGHT=$(echo "$LTC_RESULT" | jq -r ".result.headers")
  LTC_SYNC_HEIGHT=$(echo "$LTC_RESULT" | jq -r ".result.blocks")
  LTC_PROGRESS=$(echo "$LTC_RESULT" | jq -r ".result.verificationprogress")
  LTC_PROGRESS=$(calc_progress "$LTC_SYNC_HEIGHT" "$LTC_HEIGHT" "$LTC_PROGRESS")

  ETH_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' -H 'content-type: application/json' "$ETHEREUM_ENDPOINT")
  if [ "$ETH_RESULT" = '{"jsonrpc":"2.0","id":1,"result":false}' ]; then
    ETH_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "$ETHEREUM_ENDPOINT")
    ETH_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result")")
    ETH_SYNC_HEIGHT=$ETH_HEIGHT
    ETH_PROGRESS=$(calc_progress "$ETH_SYNC_HEIGHT" "$ETH_HEIGHT")
  elif [ -n "$ETH_RESULT" ]; then
    ETH_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result.highestBlock")")
    ETH_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$ETH_RESULT" | jq -r ".result.currentBlock")")
  else
    ETH_PROGRESS=Error
  fi

  # calculate ETH chain sync progress
  ETH_BEACON_RESULT=$(curl -sL --fail -m 10 "$ETHEREUM_BEACON_ENDPOINT/eth/v1/node/syncing")
  if [ -n "$ETH_BEACON_RESULT" ]; then
    ETH_BEACON_HEIGHT=$(echo "$ETH_BEACON_RESULT" | jq -r "(.data.head_slot|tonumber)+(.data.sync_distance|tonumber)")
    ETH_BEACON_SYNC_HEIGHT=$(echo "$ETH_BEACON_RESULT" | jq -r ".data.head_slot|tonumber")
    ETH_BEACON_PROGRESS=$(calc_progress "$ETH_BEACON_SYNC_HEIGHT" "$ETH_BEACON_HEIGHT")
  else
    ETH_BEACON_PROGRESS=Error
  fi

  # calculate BCH chain sync progress
  BCH_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://switchly:password@"$BITCOIN_CASH_ENDPOINT")
  BCH_HEIGHT=$(echo "$BCH_RESULT" | jq -r ".result.headers")
  BCH_SYNC_HEIGHT=$(echo "$BCH_RESULT" | jq -r ".result.blocks")
  BCH_PROGRESS=$(echo "$BCH_RESULT" | jq -r ".result.verificationprogress")
  BCH_PROGRESS=$(calc_progress "$BCH_SYNC_HEIGHT" "$BCH_HEIGHT" "$BCH_PROGRESS")

  # calculate DOGE chain sync progress
  DOGE_RESULT=$(curl -sL --fail -m 10 --data-binary '{"jsonrpc": "1.0", "id": "node-status", "method": "getblockchaininfo", "params": []}' -H 'content-type: text/plain;' http://switchly:password@"$DOGECOIN_ENDPOINT")
  DOGE_HEIGHT=$(echo "$DOGE_RESULT" | jq -r ".result.headers")
  DOGE_SYNC_HEIGHT=$(echo "$DOGE_RESULT" | jq -r ".result.blocks")
  DOGE_PROGRESS=$(echo "$DOGE_RESULT" | jq -r ".result.verificationprogress")
  DOGE_PROGRESS=$(calc_progress "$DOGE_SYNC_HEIGHT" "$DOGE_HEIGHT" "$DOGE_PROGRESS")

  # calculate Gaia chain sync progress
  GAIA_HEIGHT=$(curl -sL --fail -m 10 https://gaia.ninerealms.com/status | jq -r ".result.sync_info.latest_block_height")
  GAIA_SYNC_HEIGHT=$(curl -sL --fail -m 10 "$GAIA_ENDPOINT/status" | jq -r ".result.sync_info.latest_block_height")
  GAIA_PROGRESS=$(calc_progress "$GAIA_SYNC_HEIGHT" "$GAIA_HEIGHT")

  # calculate AVAX chain sync progress
  AVAX_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "https://api.avax.network/ext/bc/C/rpc")
  AVAX_SYNC_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "$AVALANCHE_ENDPOINT")
  AVAX_HEIGHT=$(printf "%.0f" "$(echo "$AVAX_HEIGHT_RESULT" | jq -r ".result")")
  if [ -n "$AVAX_SYNC_HEIGHT_RESULT" ]; then
    AVAX_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$AVAX_SYNC_HEIGHT_RESULT" | jq -r ".result")")
  else
    AVAX_SYNC_HEIGHT=0
  fi
  AVAX_PROGRESS=$(calc_progress "$AVAX_SYNC_HEIGHT" "$AVAX_HEIGHT")

  # calculate BSC chain sync progress
  BSC_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "https://bsc.nodereal.io")
  BSC_SYNC_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "$BINANCE_SMART_ENDPOINT")
  BSC_HEIGHT=$(printf "%.0f" "$(echo "$BSC_HEIGHT_RESULT" | jq -r ".result")")
  if [ -n "$BSC_SYNC_HEIGHT_RESULT" ]; then
    BSC_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$BSC_SYNC_HEIGHT_RESULT" | jq -r ".result")")
  else
    BSC_SYNC_HEIGHT=0
  fi
  BSC_PROGRESS=$(calc_progress "$BSC_SYNC_HEIGHT" "$BSC_HEIGHT")

  # calculate BASE chain sync progress
  BASE_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "https://base.llamarpc.com")
  BASE_SYNC_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type: application/json' "$BASE_ENDPOINT")
  BASE_HEIGHT=$(printf "%.0f" "$(echo "$BASE_HEIGHT_RESULT" | jq -r ".result")")
  if [ -n "$BASE_SYNC_HEIGHT_RESULT" ]; then
    BASE_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$BASE_SYNC_HEIGHT_RESULT" | jq -r ".result")")
  else
    BASE_SYNC_HEIGHT=0
  fi
  BASE_PROGRESS=$(calc_progress "$BASE_SYNC_HEIGHT" "$BASE_HEIGHT")

  # calculate XRP chain sync progress
  if [ "$BIFROST_CHAINS_XRP_DISABLED" = "false" ]; then
    XRP_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"ledger","params":[{"ledger_index":"validated"}],"id":1}' -H 'content-type: application/json' "https://s1.ripple.com:51234")
    XRP_SYNC_HEIGHT_RESULT=$(curl -X POST -sL --fail -m 10 --data '{"jsonrpc":"2.0","method":"ledger","params":[{"ledger_index":"validated"}],"id":1}' -H 'content-type: application/json' "$XRP_ENDPOINT")
    XRP_HEIGHT=$(printf "%.0f" "$(echo "$XRP_HEIGHT_RESULT" | jq -r ".result.ledger.ledger_index")")
    if [ -n "$XRP_SYNC_HEIGHT_RESULT" ]; then
      XRP_SYNC_HEIGHT=$(printf "%.0f" "$(echo "$XRP_SYNC_HEIGHT_RESULT" | jq -r ".result.ledger.ledger_index")")
    else
      XRP_SYNC_HEIGHT=0
    fi
    XRP_PROGRESS=$(calc_progress "$XRP_SYNC_HEIGHT" "$XRP_HEIGHT")
  fi
fi

# calculate SWITCHLY chain sync progress
SWITCHLY_SYNC_HEIGHT=$(curl -sL --fail -m 10 switchlynode:"$SWITCHLYNODE_PORT"/status | jq -r ".result.sync_info.latest_block_height")
if [ "$PEER" != "" ]; then
  SWITCHLY_HEIGHT=$(curl -sL --fail -m 10 "$PEER:$SWITCHLYNODE_PORT/status" | jq -r ".result.sync_info.latest_block_height")
elif [ "$SEEDS" != "" ]; then
  OLD_IFS=$IFS
  IFS=","
  for PEER in $SEEDS; do
    SWITCHLY_HEIGHT=$(curl -sL --fail -m 10 "$PEER:$SWITCHLYNODE_PORT/status" | jq -r ".result.sync_info.latest_block_height") || continue
    break
  done
  IFS=$OLD_IFS
elif [ "$NET" = "mainnet" ]; then
  SWITCHLY_HEIGHT=$(curl -sL --fail -m 10 https://rpc.switchly.com/status | jq -r ".result.sync_info.latest_block_height")
elif [ "$NET" = "stagenet" ]; then
  SWITCHLY_HEIGHT=$(curl -sL --fail -m 10 https://stagenet-rpc.switchly.com/status | jq -r ".result.sync_info.latest_block_height")
else
  SWITCHLY_HEIGHT=$SWITCHLY_SYNC_HEIGHT
fi
SWITCHLY_PROGRESS=$(printf "%.3f%%" "$(jq -n "$SWITCHLY_SYNC_HEIGHT"/"$SWITCHLY_HEIGHT"*100 2>/dev/null)" 2>/dev/null) || SWITCHLY_PROGRESS=Error

cat <<"EOF"
   _____         _ __       __    __     ____             __                  __
  / ___/      __(_) /______/ /_  / /_  _/ __ \_________  / /_____  ________  / /
  \__ \ | /| / / / __/ ___/ __ \/ / / / / /_/ / ___/ __ \/ __/ __ \/ ___/ __ \/ / 
 ___/ / |/ |/ / / /_/ /__/ / / / / /_/ / ____/ /  / /_/ / /_/ /_/ / /__/ /_/ / /  
/____/|__/|__/_/\__/\___/_/ /_/_/\__, /_/   /_/   \____/\__/\____/\___/\____/_/   
                                /____/                                            
EOF
echo

if [ "$VALIDATOR" = "true" ]; then
  echo "ADDRESS     $ADDRESS"
  echo "IP          $IP"
  echo "VERSION     $VERSION"
  echo "STATUS      $STATUS"
  echo "BOND        $(format_1e8 "$BOND")"
  echo "REWARDS     $(format_1e8 "$REWARDS")"
  echo "SLASH       $(format_int "$SLASH")"
  echo "PREFLIGHT   $PREFLIGHT"
fi

echo
echo "API         http://$IP:1317/switchly/doc/"
echo "RPC         http://$IP:$SWITCHLYNODE_PORT"
echo "MIDGARD     http://$IP:8080/v2/doc"

# set defaults to avoid failures in math below
SWITCHLY_HEIGHT=${SWITCHLY_HEIGHT:=0}
SWITCHLY_SYNC_HEIGHT=${SWITCHLY_SYNC_HEIGHT:=0}
BTC_HEIGHT=${BTC_HEIGHT:=0}
BTC_SYNC_HEIGHT=${BTC_SYNC_HEIGHT:=0}
ETH_HEIGHT=${ETH_HEIGHT:=0}
ETH_SYNC_HEIGHT=${ETH_SYNC_HEIGHT:=0}
LTC_HEIGHT=${LTC_HEIGHT:=0}
LTC_SYNC_HEIGHT=${LTC_SYNC_HEIGHT:=0}
BCH_HEIGHT=${BCH_HEIGHT:=0}
BCH_SYNC_HEIGHT=${BCH_SYNC_HEIGHT:=0}
DOGE_HEIGHT=${DOGE_HEIGHT:=0}
DOGE_SYNC_HEIGHT=${DOGE_SYNC_HEIGHT:=0}
GAIA_HEIGHT=${GAIA_HEIGHT:=0}
GAIA_SYNC_HEIGHT=${GAIA_SYNC_HEIGHT:=0}
BASE_HEIGHT=${BASE_HEIGHT:=0}
BASE_SYNC_HEIGHT=${BASE_SYNC_HEIGHT:=0}

echo
printf "%-18s %-10s %-14s %-10s\n" CHAIN SYNC BEHIND TIP
printf "%-18s %-10s %-14s %-10s\n" SWITCHLY "$SWITCHLY_PROGRESS" "$(format_int $((SWITCHLY_SYNC_HEIGHT - SWITCHLY_HEIGHT)))" "$(format_int "$SWITCHLY_HEIGHT")"

if [ "$VALIDATOR" = "true" ]; then
  printf "%-18s %-10s %-14s %-10s\n" BTC "$BTC_PROGRESS" "$(format_int $((BTC_SYNC_HEIGHT - BTC_HEIGHT)))" "$(format_int "$BTC_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" "ETH" "$ETH_PROGRESS" "$(format_int $((ETH_SYNC_HEIGHT - ETH_HEIGHT)))" "$(format_int "$ETH_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" "ETH (beacon slot)" "$ETH_BEACON_PROGRESS" "$(format_int $((ETH_BEACON_SYNC_HEIGHT - ETH_BEACON_HEIGHT)))" "$(format_int "$ETH_BEACON_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" LTC "$LTC_PROGRESS" "$(format_int $((LTC_SYNC_HEIGHT - LTC_HEIGHT)))" "$(format_int "$LTC_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" BCH "$BCH_PROGRESS" "$(format_int $((BCH_SYNC_HEIGHT - BCH_HEIGHT)))" "$(format_int "$BCH_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" DOGE "$DOGE_PROGRESS" "$(format_int $((DOGE_SYNC_HEIGHT - DOGE_HEIGHT)))" "$(format_int "$DOGE_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" GAIA "$GAIA_PROGRESS" "$(format_int $((GAIA_SYNC_HEIGHT - GAIA_HEIGHT)))" "$(format_int "$GAIA_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" AVAX "$AVAX_PROGRESS" "$(format_int $((AVAX_SYNC_HEIGHT - AVAX_HEIGHT)))" "$(format_int "$AVAX_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" BSC "$BSC_PROGRESS" "$(format_int $((BSC_SYNC_HEIGHT - BSC_HEIGHT)))" "$(format_int "$BSC_HEIGHT")"
  printf "%-18s %-10s %-14s %-10s\n" BASE "$BASE_PROGRESS" "$(format_int $((BASE_SYNC_HEIGHT - BASE_HEIGHT)))" "$(format_int "$BASE_HEIGHT")"
  if [ "$BIFROST_CHAINS_XRP_DISABLED" = "false" ]; then
    printf "%-18s %-10s %-14s %-10s\n" XRP "$XRP_PROGRESS" "$(format_int $((XRP_SYNC_HEIGHT - XRP_HEIGHT)))" "$(format_int "$XRP_HEIGHT")"
  fi
fi

# skip unbond eta calculation if not within 100 blocks of tip
if [ "$((SWITCHLY_SYNC_HEIGHT - SWITCHLY_HEIGHT))" -gt 100 ]; then
  exit 0
fi

# get values for churn calculations
MIMIR=$(curl -sL --fail -m 10 "$API/switchly/mimir")
CONSTANTS=$(curl -sL --fail -m 10 "$API/switchly/constants")
VAULTS=$(curl -sL --fail -m 10 "$API/switchly/vaults/asgard")
CHURN_MIGRATE_ROUNDS=$(echo "$MIMIR" | jq -r ".CHURNMIGRATEROUNDS // empty")
CHURN_MIGRATE_ROUNDS=${CHURN_MIGRATE_ROUNDS:-$(echo "$CONSTANTS" | jq -r ".int_64_values.ChurnMigrateRounds")}
FUND_MIGRATION_INTERVAL=$(echo "$MIMIR" | jq -r ".FUNDMIGRATIONINTERVAL")
CHURN_INTERVAL=$(echo "$MIMIR" | jq -r ".CHURNINTERVAL")

# fallback to constants
[ "$FUND_MIGRATION_INTERVAL" = "null" ] && FUND_MIGRATION_INTERVAL=$(echo "$CONSTANTS" | jq -r ".int_64_values.FundMigrationInterval")
[ "$CHURN_INTERVAL" = "null" ] && CHURN_INTERVAL=$(echo "$CONSTANTS" | jq -r ".int_64_values.ChurnInterval")

# calculate churn height
CHURN_START=$(echo "$VAULTS" | jq -r '[.[]|.block_height]|max')
CHURNING=$(echo "$VAULTS" | jq '[.[]|select(.status=="RetiringVault")]|length > 0')
if [ "$CHURNING" = "false" ]; then
  # set churn height to last churn plus interval
  CHURN_START=$((CHURN_START + CHURN_INTERVAL))

  # calculate start churn eta
  PAST=$(curl -sL --fail -m 10 "${RPC}"/block?height=$((SWITCHLY_HEIGHT + SWITCHLY_HEIGHT - CHURN_START)) | jq '.result.block.header.time|split(".")[0]+"Z"|fromdateiso8601')
  NOW=$(date +%s)
  SECONDS_LEFT=$((NOW - PAST))
  DAYS_LEFT=$((SECONDS_LEFT / 86400))
  START_ETA=$(date -d @${SECONDS_LEFT} -u +"${DAYS_LEFT}d %-Hh %-Mm %-Ss")
else
  START_ETA="in progress"
fi

# calculate finish churn eta
CHURN_FINISH=$((CHURN_START + ((CHURN_MIGRATE_ROUNDS + 1) * FUND_MIGRATION_INTERVAL) + 1))
BLOCKS_LEFT=$((CHURN_FINISH - SWITCHLY_HEIGHT))
if [ "$BLOCKS_LEFT" -lt 0 ]; then
  FINISH_ETA="final migration pending - check for stuck migration or insolvency"
else
  PAST=$(curl -sL --fail -m 10 "${RPC}"/block?height=$((SWITCHLY_HEIGHT - BLOCKS_LEFT)) | jq '.result.block.header.time|split(".")[0]+"Z"|fromdateiso8601')
  NOW=$(date +%s)
  SECONDS_LEFT=$((NOW - PAST))
  DAYS_LEFT=$((SECONDS_LEFT / 86400))
  FINISH_ETA=$(date -d @${SECONDS_LEFT} -u +"${DAYS_LEFT}d %-Hh %-Mm %-Ss")
fi

echo
echo
echo "CHURN ESTIMATES - failed keygens, halted chains, and stuck migrations not considered"
echo
echo "    Churn Interval: $CHURN_INTERVAL"
echo "Migration Interval: $FUND_MIGRATION_INTERVAL"
echo "  Migration Rounds: $CHURN_MIGRATE_ROUNDS (plus one additional round expected for dust)"
echo
echo "      Start Height: $CHURN_START"
echo "   ETA Churn Start: $START_ETA"
echo
echo "     Finish Height: $CHURN_FINISH"
echo "  ETA Churn Finish: $FINISH_ETA"

# find retiring and active member vaults
if [ "$PUB_KEY" != "null" ]; then
  echo

  MEMBER_ACTIVE=$(echo "$VAULTS" | jq -r ".[] | select(.status == \"ActiveVault\") | select([.membership[] == \"$PUB_KEY\"] | any) | .addresses[] | select(.chain == \"SWITCHLY\") | .address")
  MEMBER_RETIRING=$(echo "$VAULTS" | jq -r ".[] | select(.status == \"RetiringVault\") | select([.membership[] == \"$PUB_KEY\"] | any) | .addresses[] | select(.chain == \"SWITCHLY\") | .address")
  UNBOND_ALLOWED="true"

  EXPLORER="https://switchly.com"
  if [ "$NET" = "stagenet" ]; then
    EXPLORER="https://stagenet.switchly.com"
  fi
  if [ -n "$MEMBER_ACTIVE" ]; then
    echo "  Active Member: $EXPLORER/address/$MEMBER_ACTIVE"
    UNBOND_ALLOWED="false"
  fi
  if [ -n "$MEMBER_RETIRING" ]; then
    echo "Retiring Member: $EXPLORER/address/$MEMBER_RETIRING"
    UNBOND_ALLOWED="false"
  fi

  echo " Unbond Allowed: $UNBOND_ALLOWED"
fi