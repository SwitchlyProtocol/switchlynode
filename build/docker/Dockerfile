########################################################################################
# Build
########################################################################################

FROM golang:1.23.4-bullseye AS build

ENV GOBIN=/go/bin
ENV GOPATH=/go
ENV GOOS=linux

WORKDIR /app
COPY go.mod go.sum ./
# go.mod replaces github.com/binance-chain/tss-lib with the vendored fork in ./third_party/tss-lib
# (EdDSA-capable, proto-namespaced). It must be present before `go mod download` can resolve modules.
COPY third_party ./third_party
RUN go mod download

COPY . .

ARG TAG=mainnet
ARG COMMIT=unknown

RUN make install

########################################################################################
# Deploy
########################################################################################

# bookworm (glibc 2.36, openssl 3, libdbus) — required by the prebuilt stellar-cli used for the
# mocknet Soroban deploy; the older bullseye base (glibc 2.31) cannot run modern stellar-cli.
FROM debian:bookworm-slim@sha256:60eac759739651111db372c07be67863818726f754804b8707c90979bda511df

RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  jq \
  libffi-dev \
  libdbus-1-3 \
  libssl-dev \
  openssl \
  procps \
  python3 \
  python3-pip && \
  update-ca-certificates && \
  rm -rf /var/lib/apt/lists/* && \
  pip3 install --no-cache-dir --break-system-packages requests==2.32.3 web3==7.5.0 retry==0.9.2

# Stellar CLI (prebuilt) for deploying the local Soroban router + native SAC in mocknet,
# mirroring python3/web3 used by evm-tool.py for the EVM chains.
ARG STELLAR_CLI_VERSION=27.0.0
ARG TARGETARCH
RUN case "${TARGETARCH:-amd64}" in \
  arm64) SARCH=aarch64 ;; \
  *) SARCH=x86_64 ;; \
  esac && \
  curl -fsSL "https://github.com/stellar/stellar-cli/releases/download/v${STELLAR_CLI_VERSION}/stellar-cli-${STELLAR_CLI_VERSION}-${SARCH}-unknown-linux-gnu.tar.gz" \
  | tar -xz -C /usr/local/bin stellar && \
  chmod +x /usr/local/bin/stellar && \
  stellar --version

# Copy the compiled binaries over.
COPY --from=build /go/bin/switchlynode /go/bin/bifrost /go/bin/recover-keyshare-backup /usr/bin/
COPY --from=build /go/pkg/mod/github.com/!cosm!wasm/wasmvm/v2@v2.1.2/internal/api/libwasmvm.*.so /usr/lib

COPY build/scripts /scripts

# default to mainnet
ARG TAG=mainnet
ENV NET=$TAG

# default to fullnode
CMD ["/scripts/fullnode.sh"]
