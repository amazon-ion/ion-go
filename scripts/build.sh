#!/usr/bin/env bash

set -eu

export COMMIT=$(git rev-parse --short HEAD 2> /dev/null)
export BUILDTIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

export LDFLAGS="\
    -X \"github.com/amzn/ion-go/internal.GitCommit=${COMMIT}\" \
    -X \"github.com/amzn/ion-go/internal.BuildTime=${BUILDTIME}\" \
    ${LDFLAGS:-}"

go build -o ion-go --ldflags "${LDFLAGS}" ./cmd/ion-go
