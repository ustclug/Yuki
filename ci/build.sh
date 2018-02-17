#!/bin/bash

set -x
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

if [[ -z $TRAVIS_TAG ]]; then
    TRAVIS_TAG=$(git describe --abbrev=0 --tags)
fi

HERE=$(pwd)
NAME=${NAME:-yukid}

export GOOS=${GOOS:-$(go env GOOS)} GOARCH=${GOARCH:-$(go env GOARCH)}

stage="/tmp/$NAME-$TRAVIS_TAG-$GOOS-$GOARCH/"
rm -rf "$stage"

# static linking
CGO_ENABLED=0 go build -o "$stage/yukid" ./cmd/yukid
cp LICENSE dist/{daemon.toml,yukid.service} "$stage"
tar -czf "$HERE/$NAME-$TRAVIS_TAG-$GOOS-$GOARCH.tar.gz" -C /tmp -- "${stage#/tmp/}"
