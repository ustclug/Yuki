.PHONY: all
all: yukid yukictl

.PHONY: release
release:
	goreleaser release --snapshot --clean

.PHONY: clean
clean:
	rm -rf yukid yukictl dist/

.PHONY: lint
lint:
	golangci-lint run --fix ./...

.PHONY: unit-test
unit-test:
	go test -race -v ./pkg/...

.PHONY: integration-test
integration-test:
	go test -v ./test/integration/...

git_commit := $(shell git rev-parse HEAD)
build_date := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
version ?= $(shell git describe --tags)

go_ldflags := -s -w \
			  -X github.com/ustclug/Yuki/pkg/info.BuildDate=$(build_date) \
			  -X github.com/ustclug/Yuki/pkg/info.GitCommit=$(git_commit) \
			  -X github.com/ustclug/Yuki/pkg/info.Version=$(version)

.PHONY: yukid
yukid:
	go build -ldflags "$(go_ldflags)" -trimpath ./cmd/yukid

.PHONY: yukictl
yukictl:
	go build -ldflags "$(go_ldflags)" -trimpath ./cmd/yukictl
