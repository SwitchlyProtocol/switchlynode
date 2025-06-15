#!/usr/bin/env bash
set -euo pipefail

die() {
  echo "ERR: $*"
  exit 1
}

# check docs version
version=$(cat version)
if ! grep "^  version: ${version}" openapi/openapi.yaml; then
  die "docs version (openapi/openapi.yaml) does not match version file ${version}"
fi

# format golang
which gofumpt &>/dev/null || go install mvdan.cc/gofumpt@v0.5.0
FILTER=(-e '^docs/' -e '.pb.go$' -e '^openapi/gen' -e '_gen.go' -e '.pb.gw.go$' -e 'wire_gen.go$' -e '^api/')

if [ -n "$(git ls-files '*.go' | grep -v "${FILTER[@]}" | xargs gofumpt -l 2>/dev/null)" ]; then
  git ls-files '*.go' | grep -v "${FILTER[@]}" | xargs gofumpt -w 2>/dev/null
  die "Go formatting errors"
fi
go mod verify

./scripts/lint-handlers.bash

./scripts/lint-erc20s.bash

go run tools/analyze/main.go ./common/... ./constants/... ./x/...

go run tools/lint-whitelist-tokens/main.go
