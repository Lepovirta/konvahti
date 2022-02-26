#!/bin/sh
#
# Run tests and collect a coverage report.
#
set -eu

TAGS="${1:-}"

cleanup() {
    if [ -f report.txt ]; then
        rm -rf report.txt
    fi
}
trap cleanup EXIT

go test \
    -v \
    -tags "$TAGS" \
    -race \
    -coverpkg ./internal/... \
    -coverprofile=coverage.txt \
    -covermode=atomic \
    ./... \
| tee report.txt

go tool test2json < report.txt > report.json
