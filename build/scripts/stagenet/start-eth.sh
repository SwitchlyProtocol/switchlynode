#!/bin/bash
geth --sepolia \
  --http --http.api eth,net,web3 \
  --http.addr 0.0.0.0 \
  --http.port 8545 \
  --http.corsdomain "*" \
  --ws --ws.api eth,net,web3 \
  --ws.addr 0.0.0.0 \
  --ws.port 8546 \
  --authrpc.port 8551 \
  --datadir ~/sepolia-data
