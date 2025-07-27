#!/bin/bash

# https://docs.docker.com/compose/startup-order/

set -o pipefail

# set defaults
PEER="${PEER:=127.0.0.1}"
PORT="${PORT:=1317}"

echo "Waiting for SwitchlyNode API to be ready..."

until curl -s "$PEER:$PORT/switchly/ping" > /dev/null; do
  echo "SwitchlyNode API not ready, waiting..."
  sleep 5
done

echo "SwitchlyNode API is ready!"
