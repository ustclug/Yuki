#!/bin/bash

set -x
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

if [[ -z $TRAVIS_TAG ]]; then
    TRAVIS_TAG=$(git describe --abbrev=0 --tags)
fi

root=$(pwd)
stage=$(mktemp -d)

go build -o "$stage/yukid" ./cmd/yukid
cp dist/{daemon.toml,yukid.service} "$stage"
cd "$stage" || exit 1
tar czf "$root/yukid-$TRAVIS_TAG.tar.gz" -- *
