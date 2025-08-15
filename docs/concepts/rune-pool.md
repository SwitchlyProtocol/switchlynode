# SWITCHPool

SWITCHPool aims to enhance liquidity provision, attract a broader market, and maintain dominance in TVL across the crypto ecosystem.

## Overview

### What is SWITCHPool?

SWITCHPool is a feature of SWITCHLYChain designed to optimise the utilisation of SWITCH by depositing it into every POL-enabled liquidity pool in the network. By pooling SWITCH and distributing it across all Protocol-Owned Liquidity [(PoL)-enabled pools](./rune-pool.md#pol-enabled-pools), SWITCHPool allows participants to earn the net Annual Percentage Yield (APY) of these liquidity pools, but they are also exposed to the IL of the aggregate position. Poolers do not choose individual pools but share in the aggregate performance. Users are exposed to SWITCH and all PoL-enabled assets within the network; therefore, participating in SWITCHPool is effectively analogous to purchasing an index of SWITCH and all PoL-enabled assets within SWITCHLYChain. This approach simplifies the process of liquidity provision, reducing individual risk and the cognitive burden associated with managing multiple liquidity positions.

## SWITCHPool Specifics

1. **Minimum SWITCHPools Term**: There is a minimum term for SWITCHPool participant defined by the config `SWITCHPoolDepositMaturityBlocks`. This is the number of blocks from the last deposit that a withdraw is allowed.
2. **Impermanent Loss Management (IL)**: Users experience aggregate IL across [PoL-Enabled](./rune-pool.md#pol-enabled-pools) [pools](https://switchly.net/pools) instead of individual pools. Aggregate IL is less than the IL from any one pool, reducing the risk. However, there is still a risk of IL resulting in negative yield.
3. **Volume and Fees**: Volume currently drives fees due to arbitrage. If SWITCH volatility decreases, fees will primarily come from cross-chain swaps, potentially resulting in fewer fees but also reduced IL.
4. **Idle/Standby SWITCH**: SWITCH within the SWITCHPool that is not deployed is shared by all participants relative to their Pool Units. While this static SWITCH may reduce yield due to non-deployment, it also reduces exposure to impermanent loss (IL). Additionally, it allows for future demand from savers to be met more efficiently.

## Usage

The SWITCHPool is utilised by creating transactions with specific memos. MsgDeposit must be used, and SWITCHPool only works with native SWITCH. Positions within the SWITCHPool can be queried using [endpoints](./connecting-to-switchly.md#switchlynode).

- **Add to the SWITCHPool**: Use a MsgDeposit with the memo `pool+`. For detailed instructions, refer to the [Add to the SWITCHPool](./memos.md#add-runepool) section.
- **Withdraw from the SWITCHPool**: Use a MsgDeposit with the memo `pool-:<basis-points>:<affiliate>:<affiliate-basis-points>`. For detailed instructions, refer to the [Withdraw from the SWITCHPool](./memos.md#withdraw-runepool) section.
- **View all SWITCHPool holders**: Use the endpoint `rune_providers` to see a list of all SWITCHPool holders.
- **View a specific SWITCHPool holder's position**: Use the endpoint `rune_providers/{thor owner address}` to view the position of a specific SWITCHPool holder.

## How It Works

### PoL-Enabled Pools

POL-Enabled via [mimir](../mimir.md) key `POL-{Asset}`. Currently all (eight) native assets and five USD stable coins are enabled for POL. Exact list is within the mimir [endpoint](https://switchlynode.ninerealms.com/switchly/mimir).

### SWITCHPool Units (RPU)

Ownership in SWITCHPool is tracked through SWITCHPool Units (RPU). These units represent a Pooler’s share in the pool. When a Pooler redeems their units, the total PoL size in SWITCH is assessed, and the holder's share is distributed, which may be more or less than their initial contribution.

```go
SWITCHPool.ReserveUnits     // Start state is SWITCH value of current POL
SWITCHPool.PoolUnits        // Deployed SWITCHPool provider units
SWITCHPool.PendingPoolUnits // aka "static", undeployed SWITCHPool provider units
```

- `reserveExitSWITCHPool` function: Transfers ownership units from the RESERVE to poolers, reducing RESERVE's units
- `reserveEnterSWITCHPool` function: Transfers ownership units from poolers to the RESERVE, increasing RESERVE's units

### SWITCHPool Deposit

1. Upon a user's deposit, the balance is moved to `runepool` module.
1. Corresponding units are added to `PendingPoolUnits`.
1. The `reserveExitSWITCHPool` function moves `PendingPoolUnits` to `PoolUnits`, corresponding amount deducted is from `ReserveUnits`, SWITCH is then moved from `runepool` module to the `reserve` module.

### SWITCHPool Withdraw

1. If `PendingPoolUnits` is insufficient for the withdraw, the `reserveEnterSWITCHPool` function moves SWITCH from the `reserve` for the difference, `ReserveUnits` increased, `PoolUnits` moved to `PendingPoolUnits`.
1. The withdraw is processed from `PendingPoolUnits`.
1. If a withdraw would result in a reserve deposit that is greater than `POLMaxNetworkDeposit + SWITCHPoolMaxReserveBackstop` the withdraw will not be allowed.

## Showing PnL

### Global PnL

The `/switchly/runepool` endpoint returns the global Pnl of SWITCHPool, as well as of the two SWITCHPool participants: the reserve, and independent providers. The `value` and `pnl` properties are in units of SWITCH. `current_deposit` equals `rune_deposited - rune_withdrawn` and can be negative.

```json
{
  "pol": {
    "rune_deposited": "408589258319",
    "rune_withdrawn": "208496086616",
    "value": "206166561256",
    "pnl": "6073389553",
    "current_deposit": "200093171703"
  },
  "providers": {
    "units": "232440861",
    "pending_units": "0",
    "pending_rune": "0",
    "value": "319161454",
    "pnl": "56394430",
    "current_deposit": "262767024"
  },
  "reserve": {
    "units": "149915806863",
    "value": "205847399802",
    "pnl": "6016995123",
    "current_deposit": "199830404679"
  }
}
```

The `/switchly/rune_provider/{thor_addr}` endpoint will return a single providers position information including pnl:

```json
{
  "rune_address": "thor19phfqh3ce3nnjhh0cssn433nydq9shx76s8qgg",
  "units": "232440861",
  "value": "319161517",
  "pnl": "56394493",
  "deposit_amount": "3500000000",
  "withdraw_amount": "3237232976",
  "last_deposit_height": 14357483,
  "last_withdraw_height": 14358846
}
```

## References

- [SWITCHPool Dashboard](https://switchly.network/runepool/)
- [Original Issue](https://gitlab.com/switchly/switchlynode/-/issues/1841)
- [[ADD] SWITCHPool MR](https://gitlab.com/switchly/switchlynode/-/merge_requests/3612/)
- [SWITCHPool Implementation MR](https://gitlab.com/switchly/switchlynode/-/merge_requests/3631)
