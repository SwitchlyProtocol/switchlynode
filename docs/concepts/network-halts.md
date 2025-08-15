# Network Halts

```admonish warning
If the network is halted, do not send funds. The easiest check to do is if `halted = true` on the inbound addresses endpoint.
```

```admonish info
In most cases funds won't be lost if they are sent when halted, but they may be significantly delayed.
```

```admonish danger
In the worse case if SWITCHLYChain suffers a consensus halt the `inbound_addresses` endpoint will freeze with `halted = false` but the network is actually hard-halted. In this case running a fullnode is beneficial, because the last block will become stale after 6 seconds and interfaces can detect this.
```

Interfaces that provide LP management can provide more feedback to the user what specifically is paused.

There are levels of granularity the network has to control itself and chains in the event of issues. Interfaces need to monitor these settings and apply appropriate controls in their interfaces, inform users and prevent unsupported actions.

All activity is controlled within [Mimir](https://switchlynode.ninerealms.com/switchly/mimir) and needs to be observed by interfaces and acted upon. Also, see a description of [Constants and Mimir](../mimir.md).

Halt flags are Boolean. For clarity `0` = false, no issues and `> 0` = true (usually 1), halt in effect.

## Halt/ Pause Management

Each chain has granular control allowing each chain to be halted or resumed on a specific chain as required. Network-level halting is also possible.

1. **Specific Chain Signing Halt** - Allows inbound transactions but stops the signing of outbound transactions. Outbound transactions are [queued](https://switchlynode.ninerealms.com/switchly/queue). This is the least impactful halt.
   1. Mimir setting is `HALTSIGNING[Chain]`, e.g., `HALTSIGNINGETH`
2. **Specific Chain Liquidity Provider Pause -** addition and withdrawal of liquidity are suspended but swaps and other transactions are processed.
   1. Mimir setting is `PAUSELP[Chain]`, e.g., `PAUSELPBCH` for BCH
3. **Specific Chain Trading Halt** - Transactions on external chains are observed but not processed, only [refunds](memos.md#refunds) are given. SWITCHLYNode's Bifrost is running, nodes are synced to the tip therefore trading resumption can happen very quickly.
   1. Mimir setting is `HALT[Chain]TRADING`, e.g., `HALTBCHTRADING` for BCH
4. **Specific Chain Halt** - Serious halt where transitions on that chain are no longer observed and SWITCHLYNodes will not be synced to the chain tip, usually their Bifrost offline. Resumption will require a majority of nodes syncing to the tip before trading can commence.
   1. Mimir setting is `HALT[Chain]CHAIN`, e.g., `HALTBCHCHAIN` for BCH.
5. **Specific Pool Liquidity Provider Pause** - suspends deposits into a specific Liquidity Pool
   1. Mimir setting is `PAUSELPDEPOSIT-<Asset>`, e.g., `PAUSELPDEPOSIT-BTC-BTC` for BTC pool

```admonish warning
Chain specific halts do occur and need to be monitored and reacted to when they occur. Users should not be able to send transactions via an interface when a halt is in effect.
```

### **Network Level Halts**

- **Network Pause LP** `PAUSELP = 1` Addition and withdrawal of liquidity are suspended for all pools but swaps and other transactions are processed.
- **Network Pause Lending** `PAUSELOANS = 1` Opening and closing of loans is paused for all loans.
- **Network Trading Halt** `HALTTRADING = 1` Will stop all trading for every connected chain. The SWITCHLYChain blockchain will continue and native SWITCH transactions will be processed.

There is no Network level chain halt setting as the SWITCHLYChain Blockchain continually needs to produce blocks.

A chain halt is possible in which case Mimir or Midgard will not return data. This can happen if the chain suffers consensus failure or more than 1/3 of nodes are switched off. If this occurs the Dev Discord Server `#interface-alerts` will issue alerts.

```admonish warning
While very rare, a network level halt is possible and should be monitored for.
```

### Secured Asset Halt Management

1. **Global Secured Asset Halt** - Disables deposits and withdrawals of all secured assets across base and App Layers.
   1. Mimir setting is `HaltSecuredGlobal`, set to `1` to disable all operations.
2. **Specific Secured Asset Deposit Halt** - Disables deposits of secured assets in base and App Layer for the specified chain.
   1. Mimir setting is `HaltSecuredDeposit-<Chain>`, e.g., `HaltSecuredDeposit-ETH = 1` disabled deposits for ETH-ETH and all ERC20 secured assets.
3. **Specific Secured Asset Withdrawal Halt** - Same as `HaltSecuredDeposit-<Chain>` except for Secured Asset Withdrawal.
   1. Mimir setting is `HaltSecuredWithdraw-<Chain>`, e.g., `HaltSecuredWithdraw-ETH =``, set to `1`to disable;`0` to enable.

### Smart Contract Halt Management

Smart contract halts, introduced in v3.2.0 (February 2025), control CosmWasm contract execution in the App Layer. These halts pause specific or all contract activities during vulnerabilities, ensuring network security. Interfaces must monitor these settings and prevent unsupported actions.

1. **Global Smart Contract Halt** - Disables all CosmWasm contract executions in the App Layer. Used for critical vulnerabilities or network-wide issues.
   1. Mimir setting is `HaltWasmGlobal`, set to `1` to disable all contracts; `0` to enable.
2. **Contract Code Halt** - Disables all instances of a specific CosmWasm contract by its code checksum. Targets contract code with known issues.
   1. Mimir setting is `HaltWasmCs-<checksum>`, e.g., `HaltWasmCs-4UMPB3SYCM6Z5WRT5DINB66N462U5VVQVDOIFKMP5G55WKRR7VDA` disabled smart code with the checksum `4UMPB3SYCM6Z5WRT5DINB66N462U5VVQVDOIFKMP5G55WKRR7VDA`.
3. **Specific Contract Halt** - Disables a single CosmWasm contract instance by its contract address address suffix (last 6 characters). Isolates a specific contract instance.
   1. Mimir setting is `HaltWasmContract-<address suffix>`, e.g., `HaltWasmContract-w58u9f`, disabled the smart contract with a contract address ending in `w58u9f`.

### SWCY Management

Claiming and Staking of $SWCY can be enabled and disabled using flags.

- `SWCYClaimingSwapHalt`: Enables/disables SWITCH-to-SWCY swaps in the claiming module (default: 1, halted).
- `SWCYStakeDistributionHalt`: Enables/disables distribution of SWITCH revenue to SWCY stakers (default: 1, halted).
- `SWCYStakingHalt`: Enables/disables staking of SWCY tokens (default: 1, halted).
- `SWCYUnstakingHalt`: Enables/disables unstaking of SWCY tokens (default: 1, halted).
- `SWCYClaimingHalt`: Enables/disables claiming of SWCY tokens for SWITCHLYFi deposits (default: 1, halted).

### Trade Accounts

**Trade Accounts Pause** `TradeAccountsEnabled = 1` - Adding to and withdrawing from the Trade Account is enabled.

## Monitoring Mimir Keys

```bash
curl https://switchlynode.switchly.org/switchly/mimir
```

- **Integration**: App Layer interfaces must poll Mimir settings to detect halts and adjust functionality
- **Alerts**: Subscribe to Dev Discord `#interface-alerts` channel for updates

## Quick Reference Table

| Key                           | Description                             | Scope         | Effect                                             |
| ----------------------------- | --------------------------------------- | ------------- | -------------------------------------------------- |
| `HALTSIGNING[Chain]`          | Stop outbound tx signing on chain       | Chain         | Outbound txs queued, no tx broadcast               |
| `PAUSELP[Chain]`              | Pause LP add/withdraw on chain          | Chain         | LPs cannot add/remove liquidity                    |
| `HALT[Chain]TRADING`          | Halt trading on a specific chain        | Chain         | Only refunds allowed, no new swaps                 |
| `HALT[Chain]CHAIN`            | Full halt on a chain                    | Chain         | Chain not observed, Bifrost offline                |
| `PAUSELPDEPOSIT-<Asset>`      | Pause LP deposit on specific asset pool | Pool          | Deposits disabled for the asset pool               |
| `PAUSELP`                     | Pause LP actions globally               | Global        | All pools pause LP adds/removals                   |
| `HALTTRADING`                 | Halt all trading                        | Global        | No swaps across any chain                          |
| `HaltSecuredGlobal`           | Disable all secured asset txs           | Global        | Disable deposits/withdrawals of all secured assets |
| `HaltSecuredDeposit-<Chain>`  | Disable secured deposits on chain       | Chain         | No secured asset deposits allowed                  |
| `HaltSecuredWithdraw-<Chain>` | Disable secured withdrawals on chain    | Chain         | No secured asset withdrawals allowed               |
| `HaltWasmGlobal`              | Halt all CosmWasm contracts             | Global        | Disable contract execution                         |
| `HaltWasmCs-<checksum>`       | Halt contract code by checksum          | Contract Code | Disable all instances of matching contract code    |
| `HaltWasmContract-<address>`  | Halt specific contract instance         | Contract      | Disable one specific contract                      |
| `SWCYClaimingSwapHalt`         | Halt SWCY claiming swap                  | Global        | Prevent SWITCH→SWCY swaps                             |
| `SWCYStakeDistributionHalt`    | Halt SWCY staking rewards                | Global        | Stops SWCY revenue distribution                     |
| `SWCYStakingHalt`              | Halt staking of SWCY                     | Global        | Disable SWCY staking                                |
| `SWCYUnstakingHalt`            | Halt unstaking of SWCY                   | Global        | Disable SWCY unstaking                              |
| `SWCYClaimingHalt`             | Halt SWCY claiming                       | Global        | Prevent SWCY claims from SWITCHLYFi deposits            |
| `TradeAccountsEnabled`        | Enable/disable Trade Accounts           | Global        | Allow/disallow Trade Account usage                 |
