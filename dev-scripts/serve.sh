#!/usr/bin/env bash

# Fail on first error
set -e

# Fail on unset variable
set -u

# Echo commands
set -x

SCRIPT_DIR="$( cd "$(dirname ${BASH_SOURCE[0]})" &> /dev/null && pwd)"
readonly SCRIPT_DIR
cd "${SCRIPT_DIR}/.."

MODD_PATH="$(go env GOPATH)/bin/modd"
readonly MODD_PATH
readonly MODD_VERSION="v0.8"
if [[ ! -f "${MODD_PATH}" ]]; then
    go install \
        -ldflags=-linkmode=external \
        "github.com/cortesi/modd/cmd/modd@${MODD_VERSION}"
fi

"${MODD_PATH}"
