# Switchly Stellar (Soroban) Router Contract

A stateless Soroban router that forwards inbound assets directly to Switchly vaults and
emits events carrying the **full** Switchly memo — working around Stellar's 28-byte
`MEMO_TEXT` limit. Source: [`contracts/src/lib.rs`](contracts/src/lib.rs).

Exported functions: `deposit`, `deposit_with_expiry`, `transfer_out`,
`return_vault_assets`, `version` (`"3.0.0-stateless"`).

## Layout

```
chain/stellar/
├── Cargo.toml            # workspace root (profiles live here)
├── contracts/            # the single router crate (package: switchly-router)
│   └── src/{lib.rs,test.rs}
├── .stellar/             # CLI record of deployed contract ids (per network)
└── Makefile              # build / deploy helpers
```

> The duplicate top-level package manifest was removed; this is now a single-crate
> Cargo workspace. Build and deploy from **this** directory.

## Prerequisites

1. **Rust + wasm target** (via [rustup](https://rustup.rs)):
   ```bash
   rustup target add wasm32v1-none
   ```
2. **Stellar CLI** (Soroban):
   ```bash
   brew install stellar-cli      # macOS; or: cargo install --locked stellar-cli
   ```
   If installed via Homebrew's `rustup`, ensure the cargo shims are on PATH:
   `export PATH="/opt/homebrew/opt/rustup/bin:$PATH"`.

stellar-cli ≥ 23 stores identities and contract aliases in the **global** config
(`~/.config/stellar`), not the repo-local `.stellar/`. The files under `.stellar/` are
kept as a record of the deployed ids.

## Build

```bash
make build      # -> target/wasm32v1-none/release/switchly_router.wasm
make test       # contract unit tests
```

## Deploy to testnet

```bash
make identity                 # one-time: create + friendbot-fund the deployer key
make deploy NETWORK=testnet   # build + deploy, registers the `switchly_router` alias
make version                  # sanity check: prints "3.0.0-stateless"
```

`make deploy` prints the new contract id. After deploying, update the references that the
node/bifrost actually read:

- `x/switchly/router_upgrade_info_stagenet.go` → `xlmNewRouter`
- `tools/stagenet/docker-compose.yml` → `XLM_CONTRACT`
- `.stellar/contract-ids/switchly_router.json` (record)

## Currently deployed (testnet)

| Network | Contract ID |
|---------|-------------|
| Test SDF Network ; September 2015 | `CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH` |

Wasm hash: `463934840e885b5c473860d5414b4cd49796ae2dbca3dc0b01c829b98272dad7`
