# SWITCHName Guide

## Summary

[SWITCHNames](https://docs.switchly.org/how-it-works/switchly-name-service) are SWITCHLYChain's vanity address system that allows affiliates to collect fees and track their user's transactions. SWITCHNames exist on the SWITCHLYChain L1, so you will need a SWITCHLYChain address and $SWITCH to create and manage a SWITCHName.

SWITCHNames have the following properties:

- **Name:** The SWITCHName's string. Between 1-30 hexadecimal characters and `-_+` special characters.
- **Owner**: This is the SWITCHLYChain address that owns the SWITCHName
- **Aliases**: SWITCHNames can have an alias address for any external chain supported by SWITCHLYChain, and can have an alias for the SWITCHLYChain L1 that is different than the owner.
- **Expiry:** SWITCHLYChain Block-height at which the SWITCHName expires.
- **Preferred Asset:** The asset to pay out affiliate fees in. This can be any asset supported by SWITCHLYChain.

## Create a SWITCHName

SWITCHNames are created by posting a `MsgDeposit` to the SWITCHLYChain network with the appropriate [memo](../concepts/memos.md) and enough $SWITCH to cover the registration fee and to pay for the amount of blocks the SWITCHName should be registered for.

- **Registration fee**: `tns_register_fee_rune` on the [Network endpoint](https://switchlynode.ninerealms.com/switchly/network). This value is in 1e8, so `100000000 = 1 $SWITCH`
- **Per block fee**: `tns_fee_per_block_rune` on the same endpoint, also in 1e8.

For example, for a new SWITCHName to be registered for 10 years the amount paid would be:

`amt = tns_register_fee_rune + tns_fee_per_block_rune * 10 * 5256000`

`5256000 = avg # of blocks per year`

The expiration of the SWITCHName will automatically be set to the number of blocks in the future that was paid for minus the registration fee.

**Memo Format:**

Memo template is: `~:name:chain:address:?owner:?preferredAsset:?expiry`

- **name**: Your SWITCHName. Must be unique, between 1-30 characters, hexadecimal and `-_+` special characters.
- **chain:** The chain of the alias to set.
- **address**: The alias address. Must be an address of chain.
- **owner**: SWITCHLYChain address of owner (optional).
- **preferredAsset:** Asset to receive fees in. Must be supported be an active pool on SWITCHLYChain. Value should be `asset` property from the [Pools endpoint](https://switchlynode.ninerealms.com/switchly/pools).

```admonish info
Example: `~:ODIN:BTC:bc1Address:thorAddress:BTC.BTC`
```

This will register a new SWITCHName called `ODIN` with a Bitcoin alias of `bc1Address` owner of `thorAddress` and preferred asset of BTC.BTC.

```admonish info
You can use [Asgardex](https://github.com/asgardex/asgardex-desktop) to post a MsgDeposit with a custom memo. Load your wallet, then open your SWITCHLYChain wallet page > Deposit > Custom.
```

```admonish info
View your SWITCHName's configuration at the SWITCHName endpoint:

e.g. [https://switchlynode.ninerealms.com/switchly/switchlyname/](https://switchlynode.ninerealms.com/switchly/switchlyname/ac-test){name}
```

## Renewing your SWITCHName

All SWITCHName's have a expiration represented by a SWITCHLYChain block-height. Once the expiration block-height has passed, another SWITCHLYChain address can claim the SWITCHName and any associated balance in the Affiliate Fee Collector Module (Read [#preferred-asset-for-affiliate-fees](switchlyname-guide.md#preferred-asset-for-affiliate-fees "mention")), so it's important to monitor this and renew your SWITCHName as needed.

To keep your SWITCHName registered you can extend the registration period (move back the expiration block height), by posting a `MsgDeposit` with the correct SWITCHName memo and $SWITCH amount.

**Memo:**

`~:ODIN:SWITCHLY:<thor-alias-address>`

_(Chain and alias address are required, so just use current values to keep alias unchanged)._

**$SWITCH Amount:**

`rune_amt = num_blocks_to_extend * tns_fee_per_block`

_(Remember this value will be in 1e8, so adjust accordingly for your transaction)._

## Preferred Asset for Affiliate Fees

Affiliates can collect their fees in the asset of their choice (choosing from the assets that have a pool on SWITCHLYChain). In order to collect fees in a preferred asset, affiliates must use a [SWITCHName](https://docs.switchly.org/how-it-works/switchly-name-service) in their swap memos.

### Configuring a Preferred Asset for a SWITCHName

1. [**Register a SWITCHName**](../affiliate-guide/switchlyname-guide.md#create-a-switchlyname) if not done already. This is done with a `MsgDeposit` posted to the SWITCHLYChain network.
2. Set your preferred asset's chain alias (the address you'll be paid out to), and your preferred asset. _Note: your preferred asset must be currently supported by SWITCHLYChain._

For example, if you wanted to be paid out in USDC you would:

1. Grab the full USDC name from the [Pools](https://switchlynode.ninerealms.com/switchly/pools) endpoint: `ETH.USDC-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48`
2. Post a `MsgDeposit` to the SWITCHLYChain network with the appropriate memo to register your SWITCHName, set your preferred asset as USDC, and set your Ethereum network address alias. Assuming the following info:

   1. SWITCHLYChain address: `thor1dl7un46w7l7f3ewrnrm6nq58nerjtp0dradjtd`
   2. SWITCHName: `ac-test`
   3. ETH payout address: `0x6621d872f17109d6601c49edba526ebcfd332d5d`

   The full memo would look like:

   > `~:ac-test:ETH:0x6621d872f17109d6601c49edba526ebcfd332d5d:thor1dl7un46w7l7f3ewrnrm6nq58nerjtp0dradjtd:ETH.USDC-0XA0B86991C6218B36C1D19D4A2E9EB0CE3606EB48`

```admonish info
You can use [Asgardex](https://github.com/asgardex/asgardex-desktop) to post a MsgDeposit with a custom memo. Load your wallet, then open your SWITCHLYChain wallet page > Deposit > Custom.
```

```admonish info
You will also need a SWITCHLY alias set to collect affiliate fees. Use another MsgDeposit with memo: `~:<switchlyname>:SWITCHLY:<switchly-address>` to set your SWITCHLY alias. Your SWITCHLY alias address can be the same as your owner address, but won't be used for anything if a preferred asset is set.
```

Once you successfully post your MsgDeposit you can verify that your SWITCHName is configured properly. View your SWITCHName info from SWITCHLYNode at the following endpoint:\
[https://switchlynode.ninerealms.com/switchly/switchlyname/ac-test](https://switchlynode.ninerealms.com/switchly/switchlyname/ac-test)

The response should look like:

```json
{
  "name": "ac-test",
  "expire_block_height": 28061405,
  "owner": "thor19phfqh3ce3nnjhh0cssn433nydq9shx7wfmk7k",
  "preferred_asset": "BNB.BUSD-BD1",
  "affiliate_collector_rune": "0",
  "aliases": [
    {
      "chain": "ETH",
      "address": "0x3021c479f7f8c9f1d5c7d8523ba5e22c0bcb5430"
    },
    {
      "chain": "SWITCHLY",
      "address": "thor19phfqh3ce3nnjhh0cssn433nydq9shx7wfmk7k"
    },
    {
      "chain": "BNB",
      "address": "bnb1laxspje9u0faauqh7j07p9x6ds8lg4ychhg5qh"
    }
  ],
  "preferred_asset_swap_threshold_rune": "0"
}
```

Your SWITCHName is now properly configured and any affiliate fees will begin accruing in the AffiliateCollector module. You can verify that fees are being collected by checking the `affiliate_collector_rune` value of the above endpoint.
