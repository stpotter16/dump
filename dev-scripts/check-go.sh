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

gofmt -s -w .

