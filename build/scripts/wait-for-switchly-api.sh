#!/bin/bash

# https://docs.docker.com/compose/startup-order/

set -e

echo "Waiting for Switchly API..."

until curl -s "$1/switchly/ping" >/dev/null; do
  # echo "Rest server is unavailable - sleeping"
  sleep 1
done

echo "Switchly API ready"

# Additional check for lastblock endpoint to be ready for Bifrost
echo "Waiting for Switchly lastblock to be initialized (extended timeout)..."
# Wait up to 20 minutes for lastblock to be non-null to allow TSS coordination
for i in {1..1200}; do
  if response=$(curl -s "$1/switchly/lastblock" 2>/dev/null) && [ "$response" != "null" ] && [ -n "$response" ]; then
    echo "Switchly lastblock ready: $response"
    break
  fi
  if [ $i -eq 1200 ]; then
    echo "Warning: lastblock still null after 20 minutes, proceeding anyway..."
    break
  fi
  sleep 1
done
