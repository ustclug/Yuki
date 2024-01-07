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

VERSION ?= $(shell git describe --tags)
OUT_DIR ?= $(PWD)

.PHONY: yukid
yukid:
	go build -ldflags "-X github.com/ustclug/Yuki/pkg/info.BuildDate=$(build_date) \
		-X github.com/ustclug/Yuki/pkg/info.GitCommit=$(git_commit) \
		-X github.com/ustclug/Yuki/pkg/info.Version=$(VERSION)" \
		-trimpath -o $(OUT_DIR)/yukid ./cmd/yukid

.PHONY: yukictl
yukictl:
	go build -ldflags "-X github.com/ustclug/Yuki/pkg/info.BuildDate=$(build_date) \
		-X github.com/ustclug/Yuki/pkg/info.GitCommit=$(git_commit) \
		-X github.com/ustclug/Yuki/pkg/info.Version=$(version)" \
		-trimpath ./cmd/yukictl

BUILD_IMAGE ?= golang:1.21-bookworm

.PHONY: yukid-linux
yukid-linux:
	@docker run \
		--rm \
		--mount source=go-cache,destination=/root/.cache/go-build \
		--mount source=go-mod,destination=/go/pkg/mod \
		-v $(PWD):/app \
		$(BUILD_IMAGE) \
		bash -c 'cd /app && make yukid'
