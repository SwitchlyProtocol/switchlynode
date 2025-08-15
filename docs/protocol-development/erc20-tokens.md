# ERC20 Tokens

To minimise the attack surface for ERC20 tokens, SWITCHLYChain's EVM implementation whitelists ERC20 contracts. The whitelist is managed by 1INCH:

{{#embed https://tokenlists.org/token-list?url=tokens.1inch.eth }}

If the token is not found on the list, it can be added by a Pull Request to SWITCHLYNode. Example:

{{#embed https://gitlab.com/switchly/switchlynode/-/merge_requests/2085/diffs }}
