# Trade Accounts

More capital-efficient trading and arbitration than Synthetics.

Trade Accounts provide professional traders (mostly arbitrage bots) a method to execute instant trades on SWITCHLYChain without involving Layer1 transactions on external blockchains. Trade Accounts create a new type of asset, backed by the network security rather than the liquidity in a pool ([Synthetics](https://docs.switchly.org/switchly-finance/synthetic-asset-model)), or by the SWITCH asset (Derived Assets).

Arbitrage bots can arbitrage the pools faster and with more capital efficiency than Synthetics can. This is because Synthetics adds or removes from one side of the pool depth but not the other, causing the pool to move only half the distance in terms of price. For example, a $100 SWITCH --> BTC swap requires $200 of Synthetic BTC to be burned to correct the price. Trade Accounts have twice the efficiency, so a $100 SWITCH --> BTC swap would require $100 from Trade Accounts to correct the price. This allows arbitrageurs to quickly restore big deviations using less capital.

## How it Works

1. Traders deposit Layer1 assets into the network, minting a [Trade Asset](./asset-notation.md#trade-assets) in a 1:1 ratio within a Network Trade module held by the network, not the user's wallet. These assets are held separately from the Liquidity Pools.
1. Trader receives accredited shares of this module relative to their deposit versus module depth.
1. Trader can swap/trade assets <> SWITCH (or other trade asset) to and from the trade module. Because this occurs completely within SWITCHLYNode, execution times are fast and efficient. Swap fees are the same as any other Layer1 swap.
1. Trader can withdraw some or all of their balance from their Trade Account. [Outbound delay](./delays.md) applies when they withdraw.

SWITCH and Synthetics cannot be added to the Trade Account.

## Security

As assets within the Trade Account are not held in the pools, the combined pool and trade account value (combined Layer1 asset value) could exceed the total bonded. To ensure this does not occur:

1. The calculation of the [Incentive Pendulum](https://docs.switchly.org/how-it-works/incentive-pendulum) now operates based on Layer1 assets versus bonds, rather than solely on pool depths versus bonds. This ensures there is always "space" for arbitrageurs to exist in the network and be able to arbitrage pools effectively (versus synths hitting caps).

## Using Trade Accounts

Trade Accounts can be used by creating transaction memos for [adding](./memos.md#add-trade-account), [swapping](./memos.md#swap) and [withdrawing](./memos.md#withdraw-trade-account).

### Add to the Trade Account

Send Layer1 Asset to the [Inbound Address](./querying-switchly.md#getting-the-asgard-vault) with the memo:
**`TRADE+:SWITCHLYADD`**.

**Example:**

`TRADE+:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz` - Add the sent asset and amount to the Trade Account.

The Layer1 asset is converted 1:1 to a trade asset updating the Trade Account balance.

```admonish info
The owner's SWITCHLYChain Address must be specified.
```

### Swapping Trade Assets

The [swap memo](./memos.md#swap) is used when swapping to and from trade assets.

**Examples:**

- `=:ETH~ETH:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz` &mdash; Swap (from SWITCH) to Ether Trade Asset
- `=:BTC~BTC:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz` &mdash; Swap (from ETH~ETH) to Bitcoin Trade Asset
- `=:SWITCHLY.SWITCH:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz` &mdash; Swap (from ETH~ETH) to SWITCH Trade Asset
- `=:BTC~BTC:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz:1e6/1/0:dx:10` &mdash; - Swap to Bitcoin Trade Asset, using a Limit, Streaming Swaps and a 10 basis point fee to the affiliate `dx` (Asgardex)

```admonish info
The destination/receiving address of the Trade Assets MUST be a SWITCHLYChain Address!
```

### Withdrawing from the Trade Account

Send a SWITCHLYChain MsgDeposit with the memo **`TRADE-:ADDR`**.

**Example:**

`TRADE-:bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw` - Withdraw 0.1 BTC from the Trade Account and send to `bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw`. The amount in the coins array defines the withdrawal amount.

```json
{
  "body": {
    "messages": [
      {
        "": "/types.MsgDeposit",
        "coins": [
          {
            "asset": "BTC~BTC",
            "amount": "10000000",
            "decimals": "0"
          }
        ],
        "memo": "trade-:bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw",
        "signer": "thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz"
      }
    ],
    "memo": "",
    "timeout_height": "0",
    "extension_options": [],
    "non_critical_extension_options": []
  },
  "auth_info": {
    "signer_infos": [],
    "fee": {
      "amount": [],
      "gas_limit": "200000",
      "payer": "",
      "granter": ""
    }
  },
  "signatures": []
}
```

### Verify Trade Account Balances

Balances can be verified using the Owner's SWITCHLYChain Address via the `trade/account/` [endpoint](./connecting-to-switchly.md#switchlynode).

**Example:**

<https://switchlynode.ninerealms.com/switchly/trade/account/thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz>:

```json
[
  {
    "asset": "BTC~BTC",
    "units": "49853",
    "owner": "thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz",
    "last_add_height": 13082526,
    "last_withdraw_height": 0
  },
  {
    "asset": "ETH~ETH",
    "units": "1000000",
    "owner": "thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz",
    "last_add_height": 13082126,
    "last_withdraw_height": 13082526
  }
]
```
