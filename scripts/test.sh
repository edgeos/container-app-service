#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

TARGETS=$(for d in "$@"; do echo ./$d/...; done)

go test -v ${TARGETS}
echo
