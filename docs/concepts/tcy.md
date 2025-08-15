# SWITCHLYChain SWCY Token Technical Overview

## Introduction

The **SWCY** token is a native asset introduced by SWITCHLYChain to address approximately $210 million debt accumulated by SWITCHLYFi’s savings and lending services. Approved through community governance via **Proposal 6**, SWCY converts defaulted debt into equity, transforming creditors into stakeholders. See more information in this [SWITCHLYFi Unwind Medium Article](https://medium.com/switchly/thorfi-unwind-96b46dff72c0).
This page provides wallet developers with technical details to integrate SWCY, covering token mechanics, revenue distribution, and pool interaction.

## Token Purpose and Background

SWCY restructures approximately $210 million in unserviceable debt accumulated by SWITCHLYFi, which suspended lending and savings services on **January 23, 2025**. By issuing SWCY, SWITCHLYChain converts debt into equity at a 1:1 ratio ($1 of debt = 1 SWCY), providing creditors with a stake in the protocol’s future revenue. This approach aims to stabilise the ecosystem, maintain trust, and align creditor interests with SWITCHLYChain’s long-term success.

## Key Technical Specifications

- **Token Type:** Native token on SWITCHLYChain’s blockchain.
- **Total Supply:** Fixed at 210 million SWCY tokens, corresponding to $210 million in defaulted debt.
- **Purpose:** Equity-like asset representing a share of SWITCHLYChain’s revenue.
- **Distribution:** 1 SWCY per $1 of defaulted debt, allocated to SWITCHLYFi creditors.
- **Revenue Share:** SWCY stakers receive 10% (set by [`SWCYStakeSystemIncomeBps`](../mimir.md)) of system income, allocated per block to the SWCY fund module. SWITCH is distributed directly to stakers proportional to their staked position. When the SWCY Fund’s balance reaches at least 2100 SWITCH, set by [`MinSWITCHForSWCYStakeDistribution`](../mimir.md#tcy-management). Stakers must stake at least 0.001 SWCY (100000 / 1e8 and set by [`MinSWCYForSWCYStakeDistribution`](../mimir.md#tcy-management)) to be eligible.
- **Initial Pricing:** SWCY starts trading at $0.10 per token in the SWITCH/SWCY liquidity pool.
- **Claim:** Claims must be greater than zero, with amounts defined in 1e8 precision within [`tcy_claimers_mainnet`](https://gitlab.com/switchly/switchlynode/-/raw/develop/common/tcyclaimlist/tcy_claimers_mainnet.json). Claims are automatically staked.
- **Security:** SWCY is secured by SWITCHLYChain’s proof-of-stake consensus with node-bonded SWITCH.

## Liquidity and Trading

To ensure market accessibility and price stability, SWITCHLYChain has established a SWITCH/SWCY liquidity pool with the following details:

- **Initial Liquidity:** $500,000, funded by a $5 million treasury allocation to seed the SWITCH/SWCY pool.
- **Treasury Support:** $5 million deployed to purchase $500,000 of SWCY weekly for 10 weeks.
- **Trading Mechanism:** SWCY is tradable in the SWITCH/SWCY pool, a Continuous Liquidity Pool (CLP) contributing to total pooled liquidity, operating under the [Incentive Pendulum](../concepts/incentive-pendulum.md).
- **Recovery Timeline:** SWCY has inherent value based on perpetual SWITCH yield however full debt recovery (reaching $1 per SWCY) is market dependent. Full debt recovery is not guaranteed.

## Revenue Distribution Mechanism

- **Revenue Source:** Swap fees and block emissions.
- **SWCY Allocation:** 10% of system income per block is allocated to the SWCY fund.
- **Distribution Frequency:** Each block (~6 seconds), if the SWCY fund’s balance is at least 2100 SWITCH, SWITCH is distributed directly to stakers with at least 0.001 SWCY, proportional to their staked SWCY, in multiples of 2100 SWITCH. The balance accumulates over multiple blocks based on income.

### Distribution Example

For example, if SWITCHLYChain generates $100,000 in system income daily at $1.50 per SWITCH with 10% of the system income being distributed to SWCY stakers, and ~ 6 second block time, the following would occur:

- **System Income Split:** 6,666.667 SWITCH daily is sent to the SWCY Fund, which is ~0.462963 SWITCH per block (6,666.667 ÷ 14,400 blocks).
- **Distribution Cycle:** To accumulate 2100 SWITCH at ~0.462963 SWITCH/block, it takes ~4,536 blocks (2100 ÷ 0.462963). This equals ~7.56 hours or ~0.315 days. Each block thereafter, if the fund’s balance is ≥2100 SWITCH, a distribution of 2100 SWITCH occurs, with ~3.1746 cycles per day (14,400 ÷ 4,536, approximate).
- **24-Hour Period:** The SWCY fund distributes ~6,666.66 SWITCH daily to stakers (2100 SWITCH/cycle × ~3.1746 cycles/day), assuming consistent income.
- **Staker Distribution:** A user staking 1% of total SWCY (2.1 million SWCY, given 210 million SWCY total supply) would receive 1% of 2100 SWITCH = 21 SWITCH per cycle, or ~66.6666 SWITCH daily (~21 × 3.1746), assuming consistent income and stable SWITCH value. Actual distributions vary based on system income and SWITCH price.
- **Disclaimer:** The above example is hypothetical and for illustrative purposes only. Actual SWITCH distributions are not guaranteed and depend on variable factors, including SWITCHLYChain’s system income, SWITCH market price volatility, SWCY staking participation, and network conditions. Stakers should be aware of the risks outlined in the [User Interaction](#user-interaction) section.

## User Interaction

- **Claiming SWCY**: Creditors will need to claim SWCY using the [claim memo](../concepts/memos.md#claim-tcy). SWCY is automatically staked during the claim process. Only valid claims are honored, detailed in [`tcy_claimers_mainnet`](https://gitlab.com/switchly/switchlynode/-/raw/develop/common/tcyclaimlist/tcy_claimers_mainnet.json).
- **Staking SWCY**: SWCY is automatically staked when claimed to earn SWITCH distributions (requires ≥0.001 SWCY), as described in the [Revenue Distribution Mechanism](#revenue-distribution-mechanism) section. SWCY can be unstaked and held in supported wallets, but unstaked SWCY does not earn SWITCH distributions.
- **Trading SWCY**: Trade SWCY in the SWITCH/SWCY pool via [SWITCHLYChain-integrated DEXs](https://docs.switchly.org/ecosystem#exchanges-only) (e.g., SWITCHLYSwap, Asgardex).
- **Risks**:
  - **Market Volatility**: SWCY’s market price may fluctuate, starting at $0.10 but potentially rising or falling based on SWITCH performance and protocol revenue.
  - **Recovery Uncertainty**: Full debt recovery ($1 per SWCY) is not guaranteed.
  - **SWITCH Dependency**: Revenue is paid in SWITCH, exposing SWCY stakers to SWITCH price volatility.
  - **No Governance Rights**: SWCY stakers and holders do not have governance rights, unlike SWITCH holders.

## Query SWCY

### Claims

The `/switchly/tcy_claimers` endpoint returns information on all SWCY claims for SWITCHLYFi creditors, including the asset, L1 address, and claim amount. The `/switchly/tcy_claimer/{address}` endpoint returns all claims for a specific L1 address, which may include multiple claims for chains like EVM.

The `/switchly/tcy_claimers` endpoint example output:

```json
[
  {
    "asset": "avax.avax",
    "l1_address": "0x00112c24ebee9c96d177a3aa2ff55dcb93a53c80",
    "tcy_claim": 335869573367
  },
  {
    "asset": "eth.eth",
    "l1_address": "0x7e4a8391c728fed9069b2962699ab416628b19fa",
    "tcy_claim": 150000000000
  },
  {
    "asset": "btc.btc",
    "l1_address": "12Fxnarf9wmPnGnFhe9SGk745dd6bSvKdi",
    "tcy_claim": 78780988965
  },
  {
    "asset": "eth.usdc-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
    "l1_address": "0x453e85ac0f598cfc1cecc2ecbfb663f8c41c3a97",
    "tcy_claim": 1212647033705
  }
]
```

The `/switchly/tcy_claimer/{address}` endpoint example output, where address = `0x00112c24ebee9c96d177a3aa2ff55dcb93a53c80`:

```json
[
  {
    "asset": "avax.avax",
    "l1_address": "0x00112c24ebee9c96d177a3aa2ff55dcb93a53c80",
    "tcy_claim": 335869573367
  },
  {
    "asset": "bsc.bnb",
    "l1_address": "0x00112c24ebee9c96d177a3aa2ff55dcb93a53c80",
    "tcy_claim": 345729506846
  }
]
```

### SWCY Staking

The `/switchly/tcy_staker/{address}` endpoint returns the staked SWCY position where address is a thor address, e.g. `thor1230hd4mtzgxqvrjf73cjzu9mmy5gfr625eezu7`.

```json
{
  "address": "thor1230hd4mtzgxqvrjf73cjzu9mmy5gfr625eezu7",
  "amount": "10000000000000"
}
```

### SWCY Balance

SWCY is a bank token and can be accessed via the `/bank/balances/{address}` endpoint where address is a thor address, e.g. `thor1230hd4mtzgxqvrjf73cjzu9mmy5gfr625eezu7`. Use the `tcy_staker` endpoint to see the staking balance.

```json
{
  "result": [
    {
      "denom": "rune",
      "amount": "20000"
    },
    {
      "denom": "tcy",
      "amount": "100000000000"
    }
  ]
}
```

## References

- [SWITCHLYFi Unwind Medium Article](https://medium.com/switchly/thorfi-unwind-96b46dff72c0)
- [Add SWCY Merge Request](https://gitlab.com/switchly/switchlynode/-/merge_requests/3988)
- [Proposal 6 - Convert defaulted debt to $SWCY](https://gitlab.com/-/snippets/4801556)
