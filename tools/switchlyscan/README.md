# Thorscan

Thorscan provides:

1. Python `switchlyscan` library and CLI for data wrangling of blocks, events, transactions, and messages.
2. Golang package with a `Scan` function to easily scan blocks from a Golang channel.

## Python

### Installation

Simply run the following from this directory:

```bash
pip3 install .
```

### Docker

You can alternatively leverage Docker for running this utility in a pre-built container:

```bash
alias switchlyscan="docker run -it --rm registry.gitlab.com/switchly/switchlynode:switchlyscan"
```

### Examples

You can use one liners in the CLI:

```bash
# all swap events
switchlyscan 'events(lambda b,tx,e: e, types={"swap"}), start=-1'

# gas used
switchlyscan 'transactions(lambda b,tx: (tx["hash"],tx["result"]["gas_used"])), start=-1'

# failed transactions
switchlyscan 'transactions(lambda b,tx: (tx["hash"],tx["result"]["code"]), failed=True), start=-1'

# slash and leave events
switchlyscan 'events(lambda b,tx,e: e, types={"slash", "validator_request_leave"}), start=-1'

# bond slash events
switchlyscan 'events(lambda b,tx,e: e if e["bond_type"] == "\u0003" else None, types={"bond"}), start=-1'

# observed outbounds
switchlyscan 'messages(lambda b,tx,m: tx, types={"MsgObservedTxOut"}), start=-1'
```

Alternatively import the library to create more complex listener functions:

```python
# count outbound observations by chain
import collections, json, switchlyscan

counts = collections.defaultdict(lambda: 0)

def listen(height, tx, msg):
    global counts
    for tx in msg["txs"]:
        counts[tx["tx"]["chain"]] += 1

switchlyscan.scan(switchlyscan.messages(listen, types={"MsgObservedTxOut"}), start=-100, stop=-1)

print(json.dumps(counts, indent=2))
```

## Golang

```golang
package main

import (
	"gitlab.com/switchly/switchlynode/tools/switchlyscan"
)

func main() {
	for block := range switchlyscan.Scan(-200, -100) {
		println(block.Header.Height, "has", len(block.Txs), "txs")
	}
}
```

## Advanced

Override the following default config values with the Golang or Python packages via the corresponding environment variables:

```text
API_ENDPOINT = https://switchlynode-v1.ninerealms.com
RPC_ENDPOINT = https://rpc-v1.ninerealms.com
PARALLELISM  = 4
```
