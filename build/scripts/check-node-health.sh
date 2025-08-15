#!/bin/bash

# Node health check script to verify IP and version registration

set -o pipefail

PEER="${1:-localhost}"
PORT="${2:-1317}"

echo "=== Node Health Check ==="
echo "Checking nodes via $PEER:$PORT"
echo

# Get all nodes
NODES_RESPONSE=$(curl -s "http://$PEER:$PORT/switchly/nodes" || echo "[]")

if [ "$NODES_RESPONSE" = "[]" ] || [ -z "$NODES_RESPONSE" ]; then
  echo "❌ ERROR: Unable to fetch nodes from $PEER:$PORT"
  exit 1
fi

echo "$NODES_RESPONSE" | jq -r '.[] | 
  "Node: " + .node_address + 
  "\n  Status: " + .status + 
  "\n  IP: " + (.ip_address // "EMPTY") + 
  "\n  Version: " + (.version // "EMPTY") + 
  "\n  Issues: " + (
    if .ip_address == "" then "Missing IP " else "" end +
    if .version == "0.0.0" or .version == "" then "Missing Version " else "" end +
    if .status == "Unknown" then "Unknown Status " else "" end
  ) + 
  "\n"'

echo
echo "=== Summary ==="

# Count issues
TOTAL_NODES=$(echo "$NODES_RESPONSE" | jq '. | length')
MISSING_IP=$(echo "$NODES_RESPONSE" | jq '[.[] | select(.ip_address == "")] | length')
MISSING_VERSION=$(echo "$NODES_RESPONSE" | jq '[.[] | select(.version == "0.0.0" or .version == "")] | length')
UNKNOWN_STATUS=$(echo "$NODES_RESPONSE" | jq '[.[] | select(.status == "Unknown")] | length')

echo "Total nodes: $TOTAL_NODES"
echo "Missing IP addresses: $MISSING_IP"
echo "Missing versions: $MISSING_VERSION" 
echo "Unknown status: $UNKNOWN_STATUS"

if [ "$MISSING_IP" -eq 0 ] && [ "$MISSING_VERSION" -eq 0 ] && [ "$UNKNOWN_STATUS" -eq 0 ]; then
  echo "✅ All nodes properly configured!"
  exit 0
else
  echo "⚠️  Some nodes need attention"
  exit 1
fi