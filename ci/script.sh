#!/bin/bash
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

set -ex

go test ./...
go vet ./...
golint -set_exit_status core queue
