#!/usr/bin/env bash

# Fail on first error
set -e

# Fail on unset variable
set -u

fly deploy
