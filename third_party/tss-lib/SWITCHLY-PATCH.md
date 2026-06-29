# Switchly patch of thorchain's tss-lib fork

Vendored copy of `gitlab.com/thorchain/tss/tss-lib@v0.1.5` (module path
`github.com/binance-chain/tss-lib`), wired in via a `replace` directive in the repo root `go.mod`.

## Why

A single binary cannot link both `ecdsa/keygen` and `eddsa/keygen` from the upstream fork: their
protobuf messages (`KGRound1Message`, `KGRound2Message1`, `KGRound2Message2`, …) were declared with no
proto `package`, so the message full-names collided and `google.golang.org/protobuf` panicked at init
(`name conflict over KGRound2Message2`). bifrost links ECDSA for every chain and needs EdDSA for
Stellar, so both must coexist. See docs/architecture/stellar-eddsa-tss.md.

## The only change vs upstream

- `protob/eddsa-{keygen,signing,signature,resharing}.proto`: added `package eddsa;` so the EdDSA
  message descriptors are namespaced (`eddsa.KGRound2Message2`) and no longer collide with ECDSA.
- Regenerated the 4 `eddsa/**/**.pb.go` from those protos (only those four files changed; the ECDSA
  and hand-written Go are byte-identical to upstream). The Go API is unchanged — only the registered
  proto descriptor names differ.
- `go.mod`: `go 1.17` -> `go 1.21` (the regenerated code uses `any`).

## Regenerate

    cd third_party/tss-lib
    PATH="$(go env GOPATH)/bin:$PATH" \
      buf generate --path protob/eddsa-keygen.proto --path protob/eddsa-signing.proto \
                   --path protob/eddsa-signature.proto --path protob/eddsa-resharing.proto

(needs `buf` + `protoc-gen-go` matching the repo's `google.golang.org/protobuf` version, both
go-installable; no system protoc required.)
