.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: unit-test
unit-test:
	go test -race -v ./pkg/...

.PHONY: integration-test
integration-test:
	go test -v ./test/integration/...

.PHONY: yukid
yukid:
	go build -trimpath ./cmd/yukid

.PHONY: yukictl
yukictl:
	go build -trimpath ./cmd/yukictl

.PHONY: yukid-linux
yukid-linux:
	@docker run \
		--rm \
		--mount source=go-cache,destination=/root/.cache/go-build \
		--mount source=go-mod,destination=/go/pkg/mod \
		-v $(PWD):/app \
		golang:1.21-bookworm \
		bash -c 'cd /app && make yukid'
