#!/bin/bash

set -o pipefail

export SIGNER_NAME="${SIGNER_NAME:=switchlynode}"
export SIGNER_PASSWD="${SIGNER_PASSWD:=password}"

"$(dirname "$0")/wait-for-thorchain-api.sh" http://switchlynode:1317

. "$(dirname "$0")/core.sh"

# create the user if it doesn't exist
create_thor_user

# create the bifrost config file
mkdir -p /var/data/bifrost
exec bifrost -p
