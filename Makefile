.PHONY: all
all: build

.PHONY: build
build:
	goreleaser build --snapshot --clean

.PHONY: package
package:
	goreleaser release --snapshot --clean --skip=docker

.PHONY: release
release:
	goreleaser release --clean

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

.PHONY: yukid
yukid:
	GOAMD64=v1 goreleaser build --snapshot --clean --single-target --id yukid --output yukid --skip=before

.PHONY: yukictl
yukictl:
	GOAMD64=v1 goreleaser build --snapshot --clean --single-target --id yukictl --output yukictl --skip=before
