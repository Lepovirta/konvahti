#!/bin/sh
#
# Run tests and collect a coverage report.
#
set -eu

TAGS="${1:-}"

go test \
    -v \
    -tags "$TAGS" \
    -race \
    -coverpkg ./internal/... \
    -coverprofile=coverage.txt \
    -covermode=atomic \
    ./...
