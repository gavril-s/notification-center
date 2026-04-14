#!/bin/sh

set -eu

for file in contracts/openapi/*.yaml contracts/events/*.json; do
  if [ ! -f "$file" ]; then
    printf 'contract file not found: %s\n' "$file"
    exit 1
  fi
done

printf 'contract validation placeholder passed\n'
