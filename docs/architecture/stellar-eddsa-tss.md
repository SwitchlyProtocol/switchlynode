# Stellar EdDSA Threshold Signing — Implementation Plan

Status: **Plan / in progress.** Stellar outbound signing currently uses an **insecure
placeholder**; this document is the concrete plan to replace it with real threshold signing, and
tracks the foundational pieces that have been started.

---

## 1. Problem

Stellar classic accounts are **ed25519**. SwitchlyProtocol vaults are secured by **secp256k1**
threshold signing (go-tss / `binance-chain/tss-lib`). The network therefore cannot, today, produce
a valid signature for a Stellar vault account.

The current code papers over this by **deriving the vault's ed25519 key from the public secp256k1
vault pubkey**:

- Address (chain side, served via `/inbound_addresses`):
  [`common/pubkey.go` `GetAddress(StellarChain)`](../../common/pubkey.go) —
  `ed25519Seed = SHA256(secp256k1_pubkey)`, `addr = strkey(ed25519.NewKeyFromSeed(seed).Public())`.
- Signing (bifrost):
  [`bifrost/pkg/chainclients/stellar/client.go` `DeriveStellarkeyFromVaultPubKey`](../../bifrost/pkg/chainclients/stellar/client.go)
  — identical `SHA256(secp256k1_pubkey)` seed, plus a hardcoded `SD…` secret key for one mocknet
  vault.

**The secp256k1 vault pubkey is public.** Anyone can recompute `SHA256(pubkey)`, obtain the ed25519
seed, and **drain the vault**. This is acceptable only for a single-process mocknet demo and must
never run on stagenet/mainnet.

### Goal

The vault holds a **real ed25519 key produced by an EdDSA threshold-keygen ceremony**; no single
node (and certainly no outside party) can reconstruct it. Stellar outbounds are signed by an EdDSA
**threshold keysign** over the transaction hash. The vault's Stellar address is derived from the
**real** ed25519 group public key.

---

## 2. Why this is multi-component

The signing curve is hardcoded to ECDSA/secp256k1 throughout:

| Layer | Today | Evidence |
|-------|-------|----------|
| TSS library | `binance-chain/tss-lib` (thorchain fork v0.1.5) — **no `eddsa` package** | `go.mod`; only `ecdsa/keygen`, `ecdsa/signing` exist |
| go-tss keygen | imports `ecdsa/keygen` | `bifrost/tss/go-tss/keygen/tss_keygen.go` |
| go-tss keysign | imports `ecdsa/signing`, `endCh` is `*signing.SignatureData`, returns `tsslibcommon.ECSignature` | `bifrost/tss/go-tss/keysign/tss_keysign.go` |
| go-tss verify | `ecdsa.Verify(...)` | `bifrost/tss/go-tss/keysign/notifier.go:55` |
| go-tss requests | no algorithm field | `keygen/request.go`, `keysign/request.go` |
| chain: Vault | single secp256k1 `pub_key` | `proto/switchly/v1/types/type_vault.proto:25` |
| chain: address | secp256k1 → SHA256 → ed25519 | `common/pubkey.go` `GetAddress(StellarChain)` |
| bifrost: sign | secp256k1 → SHA256 → ed25519 + hardcoded key | `stellar/client.go` |

So "real TSS signing for Stellar" is **not** a bifrost-only change.

---

## 3. Library decision

**RESOLVED (validated): the pinned tss-lib fork already ships EdDSA — no port or upgrade needed.**
`gitlab.com/thorchain/tss/tss-lib v0.1.5` (the replace target for `binance-chain/tss-lib`) contains
`eddsa/{keygen,signing,resharing}`, mirroring the `ecdsa/*` packages and using the Edwards25519 curve
(`github.com/decred/dcrd/dcrec/edwards/v2`). So the original "port vs upgrade" question is moot — the
work is to **wire the existing eddsa packages through go-tss** behind the `Algo` selector, keeping the
ECDSA path byte-for-byte unchanged.

A compatibility spike (`bifrost/tss/eddsacompat`, see §6) ran a real t-of-n threshold EdDSA signing
ceremony with these packages and confirmed the output is **Stellar-verifiable** (below).

**Key constraint — the curve is a process global.** This fork selects the curve via `tss.SetCurve` /
`tss.EC()` (not a per-`Parameters` curve). ECDSA uses secp256k1; EdDSA uses Edwards. They therefore
**cannot run concurrently**: every EdDSA keygen/keysign must set the global to Edwards and be
serialized against the ECDSA path (and restore it after). go-tss already runs one ceremony at a time,
but the curve switch must be made explicit and guarded (it is a sharp footgun — a concurrent ECDSA
ceremony with the global left on Edwards would be corrupted). This is the main wiring hazard.

> Either way, **TSS protocol changes are consensus/ceremony-critical**: all validators must run the
> same library version. This ships behind a network upgrade/hard-fork, not a rolling bifrost update.

---

## 4. Design

### 4.1 Algorithm selection (go-tss)

Introduce an algorithm type and thread it through (default `ECDSA`, preserving today's behavior):

```go
// bifrost/tss/go-tss/common (new)
type Algo string
const ( ECDSA Algo = "ecdsa"; EdDSA Algo = "eddsa" )
```

- Add `Algo Algo` to `keygen.Request` and `keysign.Request` (omitempty → defaults to ECDSA on the
  wire for back-compat).
- `keygen/tss_keygen.go`, `keysign/tss_keysign.go`: branch on `Algo` to construct either the
  `ecdsa/*` or `eddsa/*` party, `endCh` type, and result conversion. Curve = `tss.S256()` vs
  `tss.Edwards()`.
- `keysign/notifier.go`: verify with `ecdsa.Verify` or `ed25519.Verify` per algo.
- `blame`: EdDSA keygen/keysign have different round counts/names than ECDSA; add EdDSA round tables.

### 4.2 bifrost keymanager

- `bifrost/tss/tss_signer.go` `KeySign.RemoteSign(msg, poolPubKey, algo)` (or a sibling
  `RemoteSignEdDSA`) that sets `Request.Algo = EdDSA`.
- The signature returned for EdDSA is the raw 64-byte `R||S`; no secp256k1 recovery/`V`.

### 4.3 Chain: vault key material

- **Proto**: add `string ed25519_pub_key = N;` to `Vault` (bech32 ed25519, or strkey). Keep the
  existing secp256k1 `pub_key` as the vault identity/index.
- **Keygen ceremony** (`x/switchly` keygen handler + `bifrost/observer`/signer keygen path): when a
  churn keygen runs, perform a **second, EdDSA keygen** over the same membership and report the
  ed25519 group pubkey alongside the secp256k1 one. Store it on the `Vault`.
- **Migration/churn**: funds move from old → new vault as today; the new vault's ed25519 key is
  generated in the same ceremony. Old placeholder vaults cannot be migrated to (no recoverable real
  key) — see §6 rollout.

### 4.4 Address derivation

- `common/pubkey.go` `GetAddress(StellarChain)`: derive from the **real ed25519 vault pubkey**, not
  `SHA256(secp256k1)`. Because `common.PubKey` is the secp256k1 key, the cleanest approach is a
  dedicated path that takes the ed25519 pubkey (the chain looks it up on the `Vault`). Introduce:
  ```go
  // common (new, pure, unit-tested) — see foundational pieces
  func Ed25519PubKeyToStellarAddress(ed25519Pub []byte) (Address, error)
  ```
  and have the Stellar inbound-address code resolve the vault's `ed25519_pub_key` and call it.

### 4.5 bifrost Stellar client

- `stellar/client.go`: remove `DeriveStellarkeyFromVaultPubKey` (SHA256 + hardcoded key) and
  `signTransactionLocally`. `SignTx` builds the unsigned tx, computes the Stellar tx **signature
  base/hash**, calls `RemoteSign(hash, vaultPubKey, EdDSA)`, and attaches the 64-byte ed25519
  signature as a `DecoratedSignature` with the vault's ed25519 hint.
- The signing key's public half = the vault `ed25519_pub_key` (fetched from the chain), so the
  source-account signature satisfies the router's `vault.require_auth()`.

---

## 5. Foundational pieces (done)

Low-risk, independently testable building blocks that don't change the running secp256k1 path:

1. ✅ **`common.Ed25519PubKeyToStellarAddress`** — pure strkey encoding of a 32-byte ed25519 pubkey to
   a `G…` address, with unit tests (known all-zero strkey vector + length check + `GetAddress`
   consistency). This is the seam that §4.4 and §4.5 both consume.
2. ✅ **go-tss `Algo` type** (`bifrost/tss/go-tss/common/algo.go`) — the `ecdsa`/`eddsa` enum +
   `NormalizeAlgo` (defaults to `ecdsa` for wire/back-compat), the selector §4.1 builds on.
3. ✅ **Safe gate on the placeholder** — the insecure placeholder signing is now permitted on
   **mocknet builds only** via a compile-time flag (`placeholderStellarSigningAllowed`, build-tag pair
   `signing_gate.go` / `signing_gate_mocknet.go`). `DeriveStellarkeyFromVaultPubKey` hard-fails on
   non-mocknet builds, so a stagenet/mainnet binary physically cannot sign with the recoverable
   placeholder key. The hardcoded key and dead signing helpers were removed.
4. ✅ **EdDSA→Stellar compatibility spike** (`bifrost/tss/eddsacompat`) — see §6.
5. ✅ **`common.WithCurveForAlgo`** (`bifrost/tss/go-tss/common/curve.go`) — serializes and switches
   tss-lib's process-global curve (secp256k1 ↔ Edwards) for the duration of a ceremony, restoring it
   afterward (the §3 hazard), with a test.
6. ✅ **`Algo` request plumbing + keyshare schema** — `Algo` field on `keygen.Request` /
   `keysign.Request` (omitempty → ECDSA), and `storage.KeygenLocalState` gains `Algo` +
   `EdDSALocalData` (opaque `json.RawMessage`, per §6.1) with a round-trip test. No eddsa proto is
   pulled into bifrost's import graph, so this is a safe, ECDSA-unchanged foundation.

---

## 6. Validated: threshold EdDSA is Stellar-verifiable

The make-or-break unknown was whether a *threshold* EdDSA signature (different nonce derivation than a
single-key signer) verifies under the **standard `crypto/ed25519`** verifier that Stellar uses.

`bifrost/tss/eddsacompat` runs a real **t-of-n** (4-of-6) in-process threshold signing ceremony using
the fork's `eddsa/{keygen,signing}` fixtures and asserts:
- the signature verifies under tss-lib's own decred/edwards verifier (sanity), **and**
- the signature verifies under **`crypto/ed25519.Verify`** (the Stellar path) — ✅ **passes**, and
- the group public key maps to a valid Stellar `G…` address via `common.Ed25519PubKeyToStellarAddress`.

So the cryptographic foundation is proven with the library already in the repo; the remaining work is
plumbing, not crypto. Three wiring hazards surfaced:
- **Global curve** (see §3): EdDSA ceremonies must `tss.SetCurve(edwards.Edwards())` and be serialized
  against ECDSA, restoring afterward. Implemented as `common.WithCurveForAlgo` (§5.5).
- **Message encoding**: tss-lib signs `msg.(*big.Int).Bytes()`, which drops leading zero bytes. Stellar
  tx hashes are fixed 32-byte values that may have leading zeros, so the keysign wrapper must encode
  the hash so signer and verifier agree on the exact bytes (e.g. fixed-width, or document the
  invariant) — otherwise verification fails ~1/256 of the time.

### 6.1 BLOCKER (prerequisite for Layer 2): protobuf message-name conflict

**A binary cannot link both `ecdsa/keygen` and `eddsa/keygen` from this tss-lib fork.** Both register
protobuf messages (`KGRound1Message`, `KGRound2Message1`, `KGRound2Message2`, …) under the **same
`protob` proto package**, so the modern `google.golang.org/protobuf` runtime panics at init:

```
panic: proto: file "protob/eddsa-keygen.proto" has a name conflict over KGRound2Message2
```

The `eddsacompat` spike avoids this only because its test binary links eddsa **without** ecdsa.
bifrost links ecdsa for every chain, so the moment any package in bifrost's (non-test) import graph
pulls in `eddsa/{keygen,signing}`, **bifrost panics on startup**. This gates the entire go-tss EdDSA
wiring (Layer 2): the parties, save data, and signature types all live in those packages.

Resolution options that were considered:
1. **Patch the tss-lib fork** to give the eddsa protos a distinct proto package and regenerate.
2. **`GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn`** env var — verified to load, but every node process
   must set it (a missed env var = startup panic) and it lets duplicate descriptors coexist. Fragile.
3. **Upgrade to `bnb-chain/tss-lib/v2`** (eddsa protos already namespaced, curve per-`Parameters`).

**DECISION: option 1 (fork-patch).** Option 3 (v2) was evaluated end-to-end (see git history /
reverted §6.2–§6.3): it does solve both this conflict and the global-curve hazard, **but** it forces
`btcd ≥ v0.23`, which dropped the root `btcd/btcec` package, cascading into a repo-wide
`btcec → btcec/v2` migration *and* an old-`btcsuite/btcutil → btcd/btcutil` migration across the
consensus-critical bitcoin/UTXO signing stack (3 parts; ~⅔ done before pivoting). That blast radius is
far larger than this proto conflict warrants, so the v2 work was **reverted** and we take the
fork-patch, which touches **no dependencies** and leaves the ECDSA/UTXO signing path untouched.

Fork-patch — **DONE** (commit "fork-patch tss-lib to namespace eddsa protos"):
1. ✅ Vendored `gitlab.com/thorchain/tss/tss-lib@v0.1.5` into `third_party/tss-lib` with a `go.mod`
   `replace`; no dependency versions change; repo builds. See `third_party/tss-lib/SWITCHLY-PATCH.md`.
2. ✅ Added `package eddsa;` to `protob/eddsa-*.proto`.
3. ✅ Regenerated the 4 eddsa `.pb.go` with `buf` + `protoc-gen-go v1.35.1` (ONLY those 4 files differ
   from upstream; ECDSA + all hand-written Go byte-identical; go directive bumped 1.17→1.21 for `any`).
4. ✅ Proven: `bifrost/tss/eddsacompat` links **both** ecdsa+eddsa with no init panic, and the
   threshold EdDSA sig still verifies under `crypto/ed25519` → valid Stellar address.

The proto conflict is resolved; bifrost can now link EdDSA.

## 9. Layer 2 — wire EdDSA through go-tss (remaining)

Reuses the kept Layer-1 foundations (the `Algo` selector, `WithCurveForAlgo`, the `KeygenLocalState`
schema). Work items:
- **Pubkey encoder** ✅ `conversion.GetTssPubKeyEdDSA` — Edwards point → hex ed25519 group key (tested).
- **keygen** ✅ `GenerateNewKeyEdDSA` (`eddsa/keygen.NewLocalParty`, no Paillier pre-params, own
  `endCh`/save-data, returns `EDDSAPub`); server `Keygen` branches on `req.Algo` + reports via
  `GetTssPubKeyEdDSA`. A gated churn hook (`BIFROST_EDDSA_KEYGEN_VALIDATION`) runs it on mocknet.
- **keysign** ✅ `SignMessageEdDSA` (`eddsa/signing`, keyshare from `EdDSALocalData`), result is the
  64-byte ed25519 sig; server `generateSignature` branches on `req.Algo`.
- **notifier** ✅ `verifySignatureEdDSA` uses `crypto/ed25519.Verify` with the hex 32-byte pool key and
  the 64-byte sig (tested: `TestNotifierEdDSA`). Algo threaded through `WaitForSignature`/`NewNotifier`.
- **bifrost RemoteSign** ✅ `KeySign.RemoteSignEdDSA` returns the canonical 64-byte ed25519 sig (carried
  via `keysign.Signature.EncodedSignature`, since tss-lib's eddsa R/S are stripped big-endian bytes).
- **blame**: EdDSA keygen/keysign reuse the shared timeout-blame helpers; round-count tables are the
  ECDSA ones today (acceptable for keygen; revisit if EdDSA-specific blame granularity is needed).

**Live-cluster status (2026-06-29).** With the gated hook on, the EdDSA keygen path *executes* across
the 4 separate bifrost processes on the docker `mocknet-cluster` — it builds the eddsa party and
attempts joinParty. A *successful* group key was not yet produced because the cluster's p2p does not
form (CometBFT validators can't stay synced with the seed; bifrost TSS bootstrap peers don't connect)
— and this blocks the **ECDSA** keygen identically. So the EdDSA wiring is validated to the same degree
as ECDSA on this cluster; closing the loop needs the cluster p2p repaired (separate, non-consensus).

**Hard constraint — the global curve and concurrency.** The fork selects the curve via a process
global (`tss.SetCurve`). A per-ceremony mutex around the curve (`WithCurveForAlgo`) is correct for a
**production node** (one process = one party; serialize its ECDSA vs EdDSA ceremonies) — but it would
**deadlock the in-process 4-node test** (`tss_4nodes_test.go`), because there the 4 interdependent
parties of one ceremony run in a single process and would each block on the same curve mutex. So:
- production server `Keygen`/`KeySign` set the curve per-ceremony under the existing per-op lockers
  (`tssKeyGenLocker` etc.); concurrent ECDSA-keysign + EdDSA-keysign in one process is the residual
  hazard to guard (rare for EdDSA; document/serialize). This is the cost of the fork's global curve
  (v2's per-`Parameters` curve would have removed it — but at the btcec/btcutil cascade cost).
- the 4-node EdDSA test sets the global curve to Edwards once for the all-EdDSA block (no per-party
  mutex), and validates an EdDSA keygen→keysign round (sig verifies under `crypto/ed25519`).

### 9.1 Remaining (consensus-critical — coordinated upgrade)

These are the only pieces left for end-to-end EdDSA, and both are gated on a network upgrade:

1. **Carry the ed25519 group key from keygen to the chain.** `bifrost/tss/keygen.go GenerateNewKey`
   must run *both* the ECDSA and EdDSA keygen at churn (the gated hook already does the second one) and
   return both keys; `common.NewPubKeySet(secp, eddsa)` instead of `(cpk, cpk)`.
2. **`MsgTssPool` + vault storage.** `MsgTssPool` carries a single secp256k1 `PoolPubKey`; add an
   **optional** `ed25519_pub_key` field (additive; empty == today's behavior, so ECDSA-only churns are
   byte-identical) and store it into the vault `PubKeySet.Ed25519` in `handler_tss_keysign.go`. The
   vault query/`inbound_addresses` then exposes it. *Proto + handler = state-machine change → version
   gate + coordinated upgrade; every validator must run the EdDSA build before the first EdDSA churn.*
3. **Stellar `SignTx`.** Replace the mocknet placeholder: fetch the vault `ed25519_pub_key`, compute the
   Stellar tx **signature base** (hash; §6 leading-zero caveat), call `KeySign.RemoteSignEdDSA`, and
   attach a `DecoratedSignature{Hint: lastByte(pubkey), Signature: sig}`. Remove the placeholder once
   real keys exist (the §5.3 gate stays until then).

`EdDSALocalData` can also become the typed `eddsa/keygen.LocalPartySaveData` now the proto conflict is
gone (it is `json.RawMessage` today only to keep eddsa protos out of bifrost's graph pre-patch).

### 9.2 Validation prerequisite (non-consensus)

Before enabling on any network, the cluster must actually complete a multi-node keygen+keysign. The
mocknet-cluster p2p needs repair (CometBFT validator peer-sync / PEX, and bifrost TSS `bootstrap_peers`
from `PEER`). This is devops/config, not consensus, and currently blocks ECDSA churn too.

---

## 7. Rollout / safety

- EdDSA TSS is a **consensus-critical** change: version-gate the keygen/keysign algorithm and ship via
  a coordinated network upgrade; all validators must run the EdDSA-capable build before the first
  EdDSA churn.
- Placeholder vaults have **no real key** and must never hold mainnet/stagenet funds. The §5.3 gate
  enforces this at compile time until a churn has produced a real EdDSA vault.
- Add cross-impl test vectors (keygen group key, keysign over a fixed message) and a localnet
  multi-node keygen+keysign integration test before enabling on any public network.

---

## 8. Work breakdown

1. ✅ Library: the fork already ships `eddsa/{keygen,signing,resharing}` — no port needed; spike proves
   Stellar-verifiability (§6). Remaining: manage the global-curve switch (§3).
2. ✅ go-tss: `Algo` plumbing + the eddsa keygen/keysign party branches + server curve switch +
   notifier `ed25519.Verify` (unblocked by §6.1 fork-patch). *(consensus-critical; ECDSA byte-unchanged)*
3. Chain: `MsgTssPool` optional `ed25519_pub_key` + store into vault `PubKeySet.Ed25519` + churn runs
   both keygens (§9.1). **Remaining — consensus-critical, coordinated upgrade.**
4. ✅ common: ed25519→Stellar address seam (§5.1). Remaining: resolve the vault's real ed25519 pubkey
   for `GetAddress(StellarChain)` and remove the SHA256 placeholder (depends on #3).
5. ✅ bifrost EdDSA `RemoteSignEdDSA` primitive. Remaining: Stellar `SignTx` via threshold ed25519
   (§9.1.3; depends on #3 for the vault ed25519 key); remove the placeholder once real keys exist.
6. **Validation gap:** the mocknet-cluster p2p must be repaired (§9.2) to run a real multi-node
   keygen+keysign; then add cross-impl test vectors + version gate before any public network.
