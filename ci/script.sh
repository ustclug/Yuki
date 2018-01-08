#!/bin/bash
HERE=$(dirname "$0")
cd "$HERE/../" || exit 1

set -ex

pkgs=$(go list ./... | grep -v /vendor/)
go test $pkgs
go vet $pkgs
golint -set_exit_status core queue
