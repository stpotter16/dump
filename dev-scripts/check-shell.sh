#!/usr/bin/env bash

# Fail on first error
set -e

# Fail on unset variable
set -u

BASH_SCRIPTS=()

while read -r filepath; do
    if head -n 1 "${filepath}" | grep --quiet --regexp 'bash'; then
        BASH_SCRIPTS+=("${filepath}")
    fi
done < <(git ls-files)

# Echo commands before running
set -x

shellcheck "${BASH_SCRIPTS[@]}"
