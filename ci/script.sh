#!/bin/bash
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

set -ex

go test -race ./...
go vet ./...
golint -set_exit_status $(go list ./...|grep -Pv 'auth|server')
