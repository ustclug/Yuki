#!/bin/bash

set -x
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

if [[ -z $GITHUB_TAG ]]; then
    GITHUB_TAG=$(git describe --tags || echo 'unknown')
fi

HERE=$(pwd)
NAME=${NAME:-yuki}

export GOOS=${GOOS:-$(go env GOOS)} GOARCH=${GOARCH:-$(go env GOARCH)} CGO_ENABLED=0

stage="/tmp/$NAME-$GITHUB_TAG-$GOOS-$GOARCH/"
rm -rf "$stage"

# static linking
go build -trimpath -ldflags '-w -s' -o "$stage/yukid" ./cmd/yukid
go build -trimpath -ldflags '-w -s' -o "$stage/yukictl" ./cmd/yukictl
cp LICENSE dist/{daemon.toml,yukid.service} "$stage"
tar -czf "$HERE/$NAME-$GITHUB_TAG-$GOOS-$GOARCH.tar.gz" -C /tmp -- "${stage#/tmp/}"
