#!/bin/bash
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

set -ex

go test -race ./...
go vet ./...
