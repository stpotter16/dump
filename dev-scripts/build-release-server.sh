#!/usr/bin/env bash

# Fail on first error
set -e

# Fail on unset variable
set -u

# Echo commands
set -x

go build \
    -ldflags "-s -w -linkmode external -extldflags '-static'" \
    -tags sqlite_omit_load_extension \
    -o ./release/go-template \
    cmd/server/main.go
