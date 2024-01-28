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

.PHONY: yukid
yukid:
	go build -ldflags "-X github.com/ustclug/Yuki/pkg/info.BuildDate=$(build_date) \
		-X github.com/ustclug/Yuki/pkg/info.GitCommit=$(git_commit) \
		-X github.com/ustclug/Yuki/pkg/info.Version=$(version)" \
		-trimpath ./cmd/yukid

.PHONY: yukictl
yukictl:
	go build -ldflags "-X github.com/ustclug/Yuki/pkg/info.BuildDate=$(build_date) \
		-X github.com/ustclug/Yuki/pkg/info.GitCommit=$(git_commit) \
		-X github.com/ustclug/Yuki/pkg/info.Version=$(version)" \
		-trimpath ./cmd/yukictl
