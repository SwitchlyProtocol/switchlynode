# Transaction Memos

## Overview

Transactions to THORChain pass user intent with the `MEMO` field on their respective chains. THORChain inspects the transaction object and the `MEMO` in order to process the transaction, so care must be taken to ensure the `MEMO` and the transaction are both valid. If not, THORChain will automatically refund the assets. Memos are set in inbound transactions unless specified.

THORChain uses specific [asset notation](asset-notation.md) for all assets. Assets and functions can be abbreviated, and affiliate addresses and asset amounts can be shortened to [reduce memo length](memo-length-reduction.md), including through use of [scientific notation](memo-length-reduction.md#scientific-notation). Some parameters can also refer to a [THORName](../affiliate-guide/thorname-guide.md) instead of an address.

Guides have been created for [Swap](../swap-guide/quickstart-guide.md), [Savers](../saving-guide/quickstart-guide.md) and [Lending](../lending/quick-start-guide.md) to enable quoting and the automatic construction of memos for simplicity.

All memos are listed in the [relevant THORChain source code](https://gitlab.com/thorchain/thornode/-/blob/develop/x/thorchain/memo/memo.go) variable `stringToTxTypeMap`.

### Memo Size Limits

THORChain has a [memo size limit of 250 bytes](https://gitlab.com/thorchain/thornode/-/blob/develop/constants/constants.go?ref_type=heads#L32). Any inbound tx sent with a larger memo will be ignored. Additionally, memos on UTXO chains are further constrained by the `OP_RETURN` size limit, which is [80 bytes](https://developer.bitcoin.org/devguide/transactions.html#null-data), which can be extended by using the approach described [here](sending-transactions.md#memo-greater-than-80-characters).

### Dust Thresholds

THORChain has various dust thresholds (dust limits), defined on a per-chain basis. Refer to the [Dust-thresholds and transaction validation](sending-transactions.md#dust-thresholds-and-transaction-validation) for details.

## Format

All memos follow the format: `FUNCTION:PARAM1:PARAM2:PARAM3:PARAM4`

The function is invoked by a string, which in turn calls a particular handler in the state machine. The state machine parses the memo looking for the parameters which it simply decodes from human-readable strings.

Some parameters are optional. Simply leave them blank but retain the `:` separator, e.g., `FUNCTION:PARAM1:::PARAM4`.

## Permitted Functions

The following functions can be put into a memo:

1. [**SWAP**](memos.md#swap)
1. [**ADD Liquidity**](memos.md#add-liquidity)
1. [**WITHDRAW Liquidity**](memos.md#withdraw-liquidity)
1. [**CLAIM TCY**](memos.md#claim-tcy)
1. [**STAKE TCY**](memos.md#stake-tcy)
1. [**UNSTAKE TCY**](memos.md#unstake-tcy)
1. [**ADD Trade Account**](memos.md#add-trade-account)
1. [**WITHDRAW Trade Account**](memos.md#withdraw-liquidity)
1. [**ADD Secured Asset**](memos.md#add-secured-asset)
1. [**WITHDRAW Secured Asset**](memos.md#withdraw-secured-asset)
1. [**EXECUTE Smart Contract**](memos.md#execute)
1. [**SWITCH Asset**](memos.md#switch)
1. [**DEPOSIT** **Savers**](memos.md#deposit-savers)
1. [**WITHDRAW Savers**](memos.md#withdraw-savers)
1. [**OPEN** **Loan**](memos.md#open-loan)
1. [**REPAY Loan**](memos.md#repay-loan)
1. [**DEPOSIT RUNEPool**](memos.md#deposit-runepool)
1. [**WITHDRAW RUNEPool**](memos.md#withdraw-runepool)
1. [**BOND**, **UNBOND** & **LEAVE**](memos.md#bond-unbond-and-leave)
1. [**DONATE** & **RESERVE**](memos.md#donate-and-reserve)
1. [**MIGRATE**](memos.md#migrate)
1. [**NOOP**](memos.md#noop)

### Swap

Perform an asset swap.

**`SWAP:ASSET:DESTADDR:LIM/INTERVAL/QUANTITY:AFFILIATE:FEE`**

```admonish info
For the DEX aggregator-oriented variation of the `SWAP` memo, see [Aggregators Memos](../aggregators/memos.md).
```

| Parameter     | Notes                                                                                 | Conditions                                                                                                                                      |
| ------------- | ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| Payload       | Send the asset to swap.                                                               | Must be an active pool on THORChain.                                                                                                            |
| `SWAP`        | The swap handler.                                                                     | Also `s` or `=`                                                                                                                                 |
| `:ASSET`      | The [asset identifier](asset-notation.md).                                            | Can be shortened.                                                                                                                               |
| `:DESTADDR`   | The destination address to send to.                                                   | Can use THORName.                                                                                                                               |
| `/REFUNDADDR` | The destination address for a refund to be sent to.                                   | Optional. If provided, the refund will be sent to this address; otherwise, it will be sent to the originator’s address.                         |
| `:LIM`        | The trade limit, i.e., set 100000000 to get a minimum of 1 full asset, else a refund. | Optional. 1e8 or scientific notation.                                                                                                           |
| `/INTERVAL`   | Swap interval in blocks.                                                              | Optional. If 0, do not stream.                                                                                                                  |
| `/QUANTITY`   | Swap quantity. The interval value determines the frequency of swaps in blocks.        | Optional. If 0, network will determine the number of swaps.                                                                                     |
| `:AFFILIATE`  | The affiliate addresses.                                                              | Optional. Define up to [MultipleAffiliatesMaxCount](../mimir.md#swapping) (currently 5) THORNames or THOR Addresses, separated by `/`           |
| `:FEE`        | The [affiliate fees](fees.md#affiliate-fee). RUNE is sent to affiliate.               | Optional. Ranges from 0 to 1000 Basis Points. Specify one fee for all affiliates, or individual fees matching the number of affiliates defined. |

**Syntactic Examples:**

- `SWAP:ASSET:DESTADDR` &mdash; simple swap
- `SWAP:ASSET:DESTADDR/REFUNDADDR:LIM/1/0:AFFILIATE:FEE` &mdash; swap with refund address specified
- `SWAP:ASSET:DESTADDR:LIM` &mdash; swap with trade limit
- `SWAP:ASSET:DESTADDR:LIM/0/1` &mdash; swap with limit, do not stream swap
- `SWAP:ASSET:DESTADDR:LIM/3/0` &mdash; swap with limit, optimise swap amount, every 3 blocks
- `SWAP:ASSET:DESTADDR:LIM/1/0:AFFILIATE:FEE` &mdash; swap with limit, optimised and affiliate fee
- `SWAP:ASSET:DESTADDR:LIM/1/0:AFFILIATE:FEE` &mdash; swap with limit, optimised and affiliate fee
- `SWAP:ASSET:DESTADDR:LIM/1/0:AFFILIATE1/AFFILIATE2/AFFILIATE3:FEE` &mdash; swap with limit, optimised and affiliate fee where each affiliate is paid the fee.
- `SWAP:ASSET:DESTADDR:LIM/1/0:AFFILIATE1/AFFILIATE2/AFFILIATE3:FEE1/FEE2/FEE3` &mdash; swap with limit, optimised and affiliate fee where each affiliate is paid the specified fee.

**Real-world Examples:**

- `SWAP:ETH.ETH:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0` &mdash; swap to Ether, send output to the specified address
- `=:BTC.BTC:bc1q6527vxxqjpq80la2l0sw7hay3lj6dz07zs6gzl/0x7a093cebfa77403672d68e1c22d0681400a36682` &mdash; swap to Ether, send output to the specified address. If a refund, send to the other specific address. Source and refund address should be the same chain.
- `SWAP:ETH.ETH:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0:10000000` &mdash; same as above except the ETH output should be more than 0.1 ETH else refund
- `SWAP:ETH.ETH:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0:10000000/1/1` &mdash; same as above except do not stream the swap
- `SWAP:ETH.ETH:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0:10000000/3/0` &mdash; same as above except streaming the swap, every 3 blocks, and THORChain to calculate the number of swaps required to achieve optimal price efficiency
- `SWAP:ETH.ETH:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0:10000000/3/0:t:10` &mdash; same as above except sends 10 basis points from the input to affiliate `t` (THORSwap)
- `s:ETH.ETH:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0:1e6/3/0:t:10` &mdash; same as above except with a reduced memo and scientific notation trade limit
- `=:r:thor1el4ufmhll3yw7zxzszvfakrk66j7fx0tvcslym:19779138111` &mdash; swap to at least 197.79 RUNE
- `=:ETH/USDC-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48:thor15s4apx9ap7lazpsct42nmvf0t6am4r3w0r64f2:628197586176` &mdash; swap to at least 6281.9 Synthetic USDC (Synths are deprecated)
- `=:BSC.BNB:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0:544e6/2/6` &mdash; swap to at least 5.4 BNB, using streaming swaps, 6 swaps, every 2 blocks
- `=:BTC~BTC:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz:1e6/1/0:dx:10` &mdash; Swap to Bitcoin Trade Asset, using a Limit, Streaming Swaps and a 10 basis point fee to the affiliate `dx` (Asgardex)
- `=:BTC-BTC:thor1g6pnmnyeg48yc3lg796plt0uw50qpp7humfggz:1e6/1/0:dx:10` &mdash; Swap to Bitcoin Secured Asset, using a Limit, Streaming Swaps and a 10 basis point fee to the affiliate `dx` (Asgardex)
- `=:ETH.ETH:0x3021c479f7f8c9f1d5c7d8523ba5e22c0bcb5430::t1/t2/t3/t4/t5:10` &mdash; Swap to Ether, will skim 10 basis points for each of the affiliates
- `=:ETH.ETH:0x3021c479f7f8c9f1d5c7d8523ba5e22c0bcb5430::t1/dx/ss:10/20/30` &mdash; Swap to Ether, Will skim 10 basis points for `t1`, 20 basis points for `dx`, and 30 basis points for `ss`

### Add Liquidity

Add liquidity to a pool.

**`ADD:POOL:PAIREDADDR:AFFILIATE:FEE`**

There are rules for adding liquidity, see [the rules here](https://docs.thorchain.org/learn/getting-started#entering-and-leaving-a-pool).

| Parameter     | Notes                                                                                                                                                                                                                                  | Conditions                                                                  |
| ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Payload       | The asset to add liquidity with.                                                                                                                                                                                                       | Must be supported by THORChain.                                             |
| `ADD`         | The add liquidity handler.                                                                                                                                                                                                             | Also `a` or `+`                                                             |
| `:POOL`       | The pool to add liquidity to.                                                                                                                                                                                                          | Can be shortened.                                                           |
| `:PAIREDADDR` | The other address to link with. If on external chain, link to THOR address. If on THORChain, link to external address. If a paired address is found, the LP is matched and added. If none is found, the liquidity is put into pending. | Optional. If not specified, a single-sided add-liquidity action is created. |
| `:AFFILIATE`  | The affiliate address. The affiliate is added to the pool as an LP.                                                                                                                                                                    | Optional. Must be a THORName or THOR Address.                               |
| `:FEE`        | The [affiliate fee](fees.md#affiliate-fee).                                                                                                                                                                                            | Optional. Ranges from 0 to 1000 Basis Points.                               |

**Examples:**

- `ADD:BTC.BTC` &mdash; add liquidity single-sided. If this is a position's first add, liquidity can only be withdrawn to the same address
- `a:POOL:PAIREDADDR` &mdash; add on both sides (dual-sided)
- `+:POOL:PAIREDADDR:AFFILIATE:FEE` &mdash; add dual-sided with affiliate
- `+:ETH.ETH:` &mdash; add liquidity with position pending

### Withdraw Liquidity

Withdraw liquidity from a pool.

**`WITHDRAW:POOL:BASISPOINTS:ASSET`**

A withdrawal can be either dual-sided (withdrawn based on pool's price) or entirely single-sided (converted to one side and sent out).

| Parameter      | Notes                                                                                       | Conditions                                                    |
| -------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------- |
| Payload        | Send the dust threshold of the asset to cause the transaction to be picked up by THORChain. | [Dust thresholds](#dust-thresholds) must be met.              |
| `WITHDRAW`     | The withdraw liquidity handler.                                                             | Also `-` or `wd`                                              |
| `:POOL`        | The pool to withdraw liquidity from.                                                        | Can be shortened.                                             |
| `:BASISPOINTS` | Basis points.                                                                               | Range 0-10000, where 10000 = 100%.                            |
| `:ASSET`       | Single-sided withdraw to one side.                                                          | Optional. Can be shortened. Must be either RUNE or the ASSET. |

**Examples:**

- `WITHDRAW:POOL:10000` &mdash; dual-sided 100% withdraw liquidity. If a single-address position, this withdraws single-sidedly instead
- `-:POOL:1000` &mdash; dual-sided 10% withdraw liquidity
- `wd:POOL:5000:ASSET` &mdash; withdraw 50% liquidity as the asset specified while the rest stays in the pool, e.g., `w:BTC.BTC:5000:BTC.BTC`

### Claim TCY

Claim TCY tokens for THORFi debt. Claims must be sent form an address within [`tcy_claimers_mainnet`](https://gitlab.com/thorchain/thornode/-/raw/develop/common/tcyclaimlist/tcy_claimers_mainnet.json). Claimed TCY is assigned to the specified thor address and automatically staked.

**`TCY:ADDR`**

| Parameter | Notes                  | Conditions                                       |
| --------- | ---------------------- | ------------------------------------------------ |
| Payload   | From a claim address   | [Dust thresholds](#dust-thresholds) must be met. |
| `TCY`     | The TCY Claim handler. |                                                  |
| `ADDR`    | Must be a thor address | Becomes the owner of the TCY                     |

Example:

- `tcy:thor1tj237x6zdpk3l7x8g4qhutqnnjq5v6syglprz7` &mdash; claim TCY, sent to the originator's address.

### Stake TCY

**`TCY+`**

| Parameter | Notes                  | Conditions        |
| --------- | ---------------------- | ----------------- |
| Payload   | TCY to be staked       | Use `MsgDeposit`. |
| `TCY+`    | The TCY Stake handler. |                   |

Example:

- `TCY+` &mdash; Stakes TCY included in the payload.

### Unstake TCY

**`TCY-:BASISPOINTS`**

| Parameter      | Notes                    | Conditions                         |
| -------------- | ------------------------ | ---------------------------------- |
| Payload        | Unstake TCY              | Use `MsgDeposit`.                  |
| `TCY-`         | The TCY Unstake handler. |                                    |
| `:BASISPOINTS` | Basis points.            | Range 0-10000, where 10000 = 100%. |

Example:

- `TCY-:5000` &mdash; remove 50% of the staked TCY.

### Add Trade Account

**`TRADE+:ADDR`**

Adds an L1 asset to the [Trade Account](../concepts/trade-accounts.md).

| Parameter | Notes                                 | Conditions                                     |
| --------- | ------------------------------------- | ---------------------------------------------- |
| Payload   | The asset to add to the Trade Account | Must be a L1 asset and supported by THORChain. |
| `TRADE+`  | The trade account handler.            |                                                |
| `ADDR`    | Must be a thor address                | Specifies the owner                            |

**Example:** `TRADE+:thor1x2whgc2nt665y0kc44uywhynazvp0l8tp0vtu6` - Add the sent asset and amount to the Trade Account.

### Withdraw Trade Account

Withdraws an L1 asset from the Trade Account.

**`TRADE-:ADDR`**

| Parameter | Notes                                                                          | Conditions               |
| --------- | ------------------------------------------------------------------------------ | ------------------------ |
| Payload   | The [Trade Asset](./asset-notation.md#trade-assets) to be withdrawn and amount | Use `MsgDeposit`.        |
| `TRADE-`  | The trade account handler.                                                     |                          |
| `ADDR`    | L1 address to which the withdrawal will be sent                                | Cannot be a thor address |

Note: Trade Asset and Amount are determined by the `coins` within the `MsgDeposit`. Transaction fee in `RUNE` does apply.

**Example:**

- `TRADE-:bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw` - Withdraw 0.1 BTC from the Trade Account and send to `bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw`

  ```text
  {"body":{"messages":[{"":"/types.MsgDeposit","coins":[{"asset":"BTC~BTC","amount":"10000000","decimals":"0"}],"memo":"trade-:bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw","signer":"thor19phfqh3ce3nnjhh0cssn433nydq9shx7wfmk7k"}],"memo":"","timeout_height":"0","extension_options":[],"non_critical_extension_options":[]},"auth_info":{"signer_infos":[],"fee":{"amount":[],"gas_limit":"200000","payer":"","granter":""}},"signatures":[]}
  ```

### Add Secured Asset

**`SECURE+:ADDR`**

Converts a L1 asset to a [Secured Asset](../concepts/secured-assets.md).

| Parameter | Notes                               | Conditions                                     |
| --------- | ----------------------------------- | ---------------------------------------------- |
| Payload   | The asset to become a Secured Asset | Must be a L1 asset and supported by THORChain. |
| `SECURE+` | The Secured Asset handler.          |                                                |
| `ADDR`    | Must be a thor address              | Specifies the owner and desitnation            |

**Example:** `SECURE+:thor1x2whgc2nt665y0kc44uywhynazvp0l8tp0vtu6` - Converts the sent asset and amount to a Secured Asset.

### Withdraw Secured Asset

Converts a Secured Asset to a L1 Asset.

**`SECURE-:ADDR`**

| Parameter | Notes                                                                             | Conditions               |
| --------- | --------------------------------------------------------------------------------- | ------------------------ |
| Payload   | The [Secured Asset](./asset-notation.md#secured-assets) to be redeemed and amount | Use `MsgDeposit`.        |
| `SECURE-` | The Secured Asset handler.                                                        |                          |
| `ADDR`    | L1 address to which the L1 asset will be sent                                     | Cannot be a thor address |

Note: Secured Assets and amount are determined by the `coins` within the `MsgDeposit`. Transaction fee in `RUNE` does apply.

**Example:** `SECURE-:bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw` - Convert 0.1 BTC from a Secured Asset to a L1 and send to `bc1qp8278yutn09r2wu3jrc8xg2a7hgdgwv2gvsdyw`

### Execute

Execute a Smart Contract from a base layer transaction.

**`X:ADDR:PAYLOAD`**

| Parameter | Notes                                                                                                                          | Conditions                                       |
| --------- | ------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------ |
| Payload   | The asset to sent to the smart contract. L1 asset will be converted into a [Secured Asset](./asset-notation.md#secured-assets) | [Dust thresholds](#dust-thresholds) must be met. |
| `x`       | The Wasm Execute handler. Also `exec`.                                                                                         | Must be a THORChain address                      |
| `ADDR`    | The THORChain Wasm smart contract address                                                                                      | Must be a thor address                           |
| `MSG`     | Payload send as `msg` to the Smart Contract, encoded as a base64 byte string.                                                  |                                                  |

Example: `x:tthor14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sw58u9f:AA==`

The above example creates a `MsgWasmExec` object for a smart contract execution and calls the contract with the following parameters:

1. Contract Address: The address specified in the ADDR field `tthor14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sw58u9f`.
2. Sender Address: Derived from the Layer 1 transaction's sender (e.g., the Bitcoin address).
3. Msg Payload: The arguments provided in the MSG field (AA== in the example, decoded into the expected input for the smart contract).
4. Funds: The Layer 1 asset (e.g., BTC amount sent in) is converted into a Secured Asset and included in the smart contract call.
5. The smart contract will be executed via the following Cosmos SDK function: `ExecuteContract(ctx, contractAddr, senderAddr, msg.Msg, msg.Funds)`

### Switch

One way switch for external tokens to be a native THORChain asset. Supported Assets are listed in a switchMap within the `SwitchManager`.

| Parameter | Notes                              | Conditions                           |
| --------- | ---------------------------------- | ------------------------------------ |
| Payload   | The asset to become a Native Asset | Must be a L1 supported on THORChain. |
| `switch`  | The Secured Asset handler.         |                                      |
| `ADDR`    | The thor address for native asset  | Must be a thor address               |

### **Deposit Savers**

Deposit an asset into THORChain Savers.

**`ADD:POOL::AFFILIATE:FEE`**

| Parameter    | Notes                                                                  | Conditions                                      |
| ------------ | ---------------------------------------------------------------------- | ----------------------------------------------- |
| Payload      | The asset to add liquidity with.                                       | Must be supported by THORChain.                 |
| `ADD`        | The deposit handler.                                                   | Also `+`                                        |
| `:POOL`      | The pool to add liquidity to.                                          | Gas and stablecoin pools only.                  |
| `:`          | Must be empty.                                                         | Optional. Required if adding affiliate and fee. |
| `:AFFILIATE` | The affiliate address.                                                 | Optional. Must be a THORName or THOR Address.   |
| `:FEE`       | The [affiliate fee](fees.md#affiliate-fee). RUNE is sent to affiliate. | Optional. Ranges from 0 to 1000 Basis Points.   |

**Examples:**

- `ADD:ETH/ETH` &mdash; deposit into the ETH Savings Vault
- `+:BTC/BTC::t:10` &mdash; deposit into the BTC Savings Vault, with 10 basis points being sent to affiliate `t` (THORSwap)
- `a:DOGE/DOGE` &mdash; deposit into the DOGE Savings Vault

```admonish info
Depositing into Savers can also work without a memo, however memos are recommended to be explicit about the transaction intent.
```

### Withdraw Savers

Withdraw an asset from THORChain Savers.

**`WITHDRAW:POOL:BASISPOINTS`**

| Parameter      | Notes                                                                                       | Conditions                                       |
| -------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------ |
| Payload        | Send the dust threshold of the asset to cause the transaction to be picked up by THORChain. | [Dust thresholds](#dust-thresholds) must be met. |
| `WITHDRAW`     | The withdraw handler.                                                                       | Also `-` or `wd`                                 |
| `:POOL`        | The pool to withdraw liquidity from.                                                        | Gas and stablecoin pools only.                   |
| `:BASISPOINTS` | Basis points.                                                                               | Optional. Range 0-10000, where 10000 = 100%.     |

**Examples:**

- `WITHDRAW:BTC/BTC:10000` &mdash; withdraw 100% from BTC Savers
- `-:ETH/ETH:5000` &mdash; withdraw 50% from ETH Savers
- `wd:BTC/BTC:1000` &mdash; withdraw 10% from BTC Savers

```admonish info
Withdrawing from Savers can be also be done [without a memo](../saving-guide/quickstart-guide.md#basic-mechanics).
```

### **Open Loan**

Open a loan on THORChain.

**`LOAN+:ASSET:DESTADDR:MINOUT:AFFILIATE:FEE`**

| Parameter    | Notes                                                                  | Conditions                                    |
| ------------ | ---------------------------------------------------------------------- | --------------------------------------------- |
| Payload      | The collateral to open the loan with.                                  | Must be L1 supported by THORChain.            |
| `LOAN+`      | The loan open handler.                                                 | Also `$+`                                     |
| `:ASSET`     | Target debt [asset identifier](asset-notation.md).                     | Can be shortened.                             |
| `:DESTADDR`  | The destination address to send the debt to.                           | Can use THORName.                             |
| `:MINOUT`    | Minimum debt amount, else a refund. Similar to `:LIM`.                 | Optional. 1e8 format.                         |
| `:AFFILIATE` | The affiliate address.                                                 | Optional. Must be a THORName or THOR Address. |
| `:FEE`       | The [affiliate fee](fees.md#affiliate-fee). RUNE is sent to affiliate. | Optional. Ranges from 0 to 1000 Basis Points. |

**Examples:**

- `LOAN+:BSC.BUSD:0xe6a30f4f3bad978910e2cbb4d97581f5b5a0ade0` &mdash; open a loan with BUSD as the debt asset
- `$+:ETH.USDC-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48:0x1c7b17362c84287bd1184447e6dfeaf920c31bbe:10400000000` &mdash; open a loan where the debt is at least 104 USDT

### **Repay Loan**

Repay a loan on THORChain.

**`LOAN-:ASSET:DESTADDR:MINOUT`**

| Parameter   | Notes                                                            | Conditions                                                    |
| ----------- | ---------------------------------------------------------------- | ------------------------------------------------------------- |
| Payload     | The repayment for the loan.                                      | Must be L1 supported on THORChain.                            |
| `LOAN-`     | The loan repayment handler.                                      | Also `$-`                                                     |
| `:ASSET`    | Target collateral [asset identifier](asset-notation.md).         | Can be shortened.                                             |
| `:DESTADDR` | The destination address to send the collateral to.               | Can use a THORName.                                           |
| `:MINOUT`   | Minimum collateral to receive, else a refund. Similar to `:LIM`. | Optional. 1e8 format, loan needs to be fully repaid to close. |

**Examples:**

- `LOAN-:BTC.BTC:bc1qp2t4hl4jr6wjfzv28tsdyjysw7p5armf7px55w` &mdash; repay BTC loan owned by owner bc1qp2t4hl4jr6wjfzv28tsdyjysw7p5armf7px55w
- `$-:ETH.ETH:0xe9973cb51ee04446a54ffca73446d33f133d2f49:404204059` &mdash; repay ETH loan owned by `0xe9973cb51ee04446a54ffca73446d33f133d2f49` and receive at least 4.04 ETH collateral back, else refund

### Deposit RUNEPool

Deposit RUNE to the RUNEPool

**`POOL+`**

| Parameter | Notes                 | Conditions                                                               |
| --------- | --------------------- | ------------------------------------------------------------------------ |
| Payload   | THOR.RUNE             | Use `MsgDeposit`. The amount of RUNE in the tx will be added to RunePool |
| `POOL+`   | The RUNEPool handler. |                                                                          |

### Withdraw RUNEPool

**`POOL-:BASISPOINTS:AFFILIATE:FEE`**

| Parameter      | Notes                                       | Conditions                                    |
| -------------- | ------------------------------------------- | --------------------------------------------- |
| Payload        | None required.                              | Use `MsgDeposit`.                             |
| `POOL-`        | The The RUNEPool handler.                   |                                               |
| `:BASISPOINTS` | Basis points.                               | Required. Range 0-10000, where 10000 = 100%.  |
| `:AFFILIATE`   | The affiliate address.                      | Optional. Must be a THORName or THOR Address. |
| `:FEE`         | The [affiliate fee](fees.md#affiliate-fee). | Optional. Ranges from 0 to 1000 Basis Points. |

Example: `POOL-:10000:dx:10` - 100% Withdraw from RUNEPool with a 10 basis point affiliate fee. Affiliates receive the corresponding basis points of positive PnL. If user is withdrawing with a loss, the affiliate will receive no fee.

### DONATE & RESERVE

Donate to a pool.

**`DONATE:POOL`**

| Parameter | Notes                                    | Conditions                                            |
| --------- | ---------------------------------------- | ----------------------------------------------------- |
| Payload   | The asset to donate to a THORChain pool. | Must be supported by THORChain. Can be RUNE or ASSET. |
| `DONATE`  | The donate handler.                      | Also `d`                                              |
| `:POOL`   | The pool to withdraw liquidity from.     | Can be shortened.                                     |

**Examples:**

- `DONATE:ETH.ETH` &mdash; donate to the ETH pool

**`RESERVE`**

Donate to the THORChain Reserve.

| Parameter | Notes                | Conditions                                   |
| --------- | -------------------- | -------------------------------------------- |
| Payload   | THOR.RUNE            | The RUNE to credit to the THORChain Reserve. |
| `RESERVE` | The reserve handler. |                                              |

### BOND, UNBOND and LEAVE

Perform node maintenance features. Also see [Pooled Nodes](https://docs.thorchain.org/thornodes/pooled-thornodes).

**`BOND:NODEADDR:PROVIDER:FEE`**

| Parameter   | Notes                                    | Conditions                                                                             |
| ----------- | ---------------------------------------- | -------------------------------------------------------------------------------------- |
| Payload     | THOR.RUNE                                | The asset to bond to a Node.                                                           |
| `BOND`      | The bond handler.                        |                                                                                        |
| `:NODEADDR` | The node to bond with.                   |                                                                                        |
| `:PROVIDER` | Whitelist in a provider.                 | Optional. Add a provider.                                                              |
| `:FEE`      | Specify an Operator Fee in Basis Points. | Optional. Default will be the mimir value (2000 Basis Points). Can be changed anytime. |

**`UNBOND:NODEADDR:AMOUNT:PROVIDER`**

| Parameter   | Notes                    | Conditions                                                            |
| ----------- | ------------------------ | --------------------------------------------------------------------- |
| Payload     | None required.           | Use `MsgDeposit`.                                                     |
| `UNBOND`    | The unbond handler.      |                                                                       |
| `:NODEADDR` | The node to unbond from. | Must be in standby only.                                              |
| `:AMOUNT`   | The amount to unbond.    | In 1e8 format. If setting more than actual bond, then capped at bond. |
| `:PROVIDER` | Unwhitelist a provider.  | Optional. Remove a provider.                                          |

**`LEAVE:NODEADDR`**

| Parameter   | Notes                       | Conditions                                                                                               |
| ----------- | --------------------------- | -------------------------------------------------------------------------------------------------------- |
| Payload     | None required.              | Use `MsgDeposit`.                                                                                        |
| `LEAVE`     | The leave handler.          |                                                                                                          |
| `:NODEADDR` | The node to force to leave. | If in Active, request a churn out to Standby for 1 churn cycle. If in Standby, forces a permanent leave. |

**Examples:**

- `BOND:thor19m4kqulyqvya339jfja84h6qp8tkjgxuxa4n4a`
- `UNBOND:thor1x2whgc2nt665y0kc44uywhynazvp0l8tp0vtu6:750000000000`
- `LEAVE:thor1hlhdm0ngr2j4lt8tt8wuvqxz6aus58j57nxnps`

### MIGRATE

Internal memo type used to mark migration transactions between a retiring vault and a new Asgard vault during churn. Special THORChain triggered outbound tx without a related inbound tx.

**`MIGRATE:BLOCKHEIGHT`**

| Parameter      | Notes                              | Conditions                    |
| -------------- | ---------------------------------- | ----------------------------- |
| Payload        | Assets migrating.                  |                               |
| `MIGRATE`      | The migrate handler.               |                               |
| `:BLOCKHEIGHT` | THORChain block height to migrate. | Must be a valid block height. |

**Example:**

- `MIGRATE:3494355` &mdash; migrate at height 3494355. See a [real-world example on RuneScan](https://runescan.io/tx/8330CAC064370F86352D247DE3046C9AA8C3E53C78760E5D35CFC7CAA3068DC6)

### NOOP

Dev-centric functions used to fix THORChain state.

```admonish danger
May cause loss of funds if not performed correctly and at the right time.
```

**`NOOP:NOVAULT`**

| Parameter  | Notes                           | Conditions                                               |
| ---------- | ------------------------------- | -------------------------------------------------------- |
| Payload    | The asset to credit to a vault. | Must be ASSET or RUNE.                                   |
| `NOOP`     | The no-op handler.              | Adds to the vault balance, but does not add to the pool. |
| `:NOVAULT` | Do not credit the vault.        | Optional. Just fix the insolvency issue.                 |

## Refunds

The following are the conditions for refunds:

| Condition                | Conditions                                                                                                   |
| ------------------------ | ------------------------------------------------------------------------------------------------------------ |
| Invalid `MEMO`           | If the `MEMO` is incorrect the user will be refunded.                                                        |
| Invalid Assets           | If the asset for the transaction is incorrect (adding an asset into a wrong pool) the user will be refunded. |
| Invalid Transaction Type | If the user is performing a multi-send vs a send for a particular transaction, they are refunded.            |
| Exceeding Price Limit    | If the final value achieved in a trade differs to expected, they are refunded.                               |

Refunds cost fees to prevent DoS (denial-of-service) attacks. The user will pay the correct outbound fee for that chain. Refund memo is sent within a outbound transaction.

## **Other Internal Memos**

- `consolidate` &mdash; consolidate UTXO transactions
- `=>` &mdash; limit swap functions (to be implemented)
- `name` or `n` or `~` &mdash; THORName operations; see [THORName Guide](../affiliate-guide/thorname-guide.md)
- `out` &mdash; for outbound transaction, set within a outbound transaction
- `ragnarok` &mdash; used to delist pools, set within a outbound transaction
- `switch` &mdash; [killswitch](https://medium.com/thorchain/upgrading-to-native-rune-a9d48e0bf40f) operations (deprecated)
- `yggdrasil+` and `yggdrasil-` &mdash; Yggdrasil vault operations (deprecated; see [ADR002](../architecture/adr-002-removeyggvaults.md))
