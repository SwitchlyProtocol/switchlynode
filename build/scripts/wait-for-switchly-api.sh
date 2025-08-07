#!/bin/bash

# https://docs.docker.com/compose/startup-order/

set -e

echo "Waiting for Switchly API..."

until curl -s "$1/switchly/ping" >/dev/null; do
  # echo "Rest server is unavailable - sleeping"
  sleep 1
done

echo "Switchly API ready"
