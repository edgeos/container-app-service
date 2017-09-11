#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

export CGO_ENABLED=0
export GOARCH="${ARCH}"

TARGETS=$(for d in "$@"; do echo ./$d/...; done)

go test -v ${TARGETS}
echo
