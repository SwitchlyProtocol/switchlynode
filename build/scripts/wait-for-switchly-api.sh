#!/bin/bash

# https://docs.docker.com/compose/startup-order/

set -o pipefail

# set defaults
API_HOST="${1:-${PEER:-127.0.0.1:1317}}"

echo "Waiting for SwitchlyNode API to be ready at $API_HOST..."

until curl -s "$API_HOST/switchly/ping" > /dev/null; do
  echo "SwitchlyNode API not ready, waiting..."
  sleep 5
done

echo "SwitchlyNode API is ready!"
