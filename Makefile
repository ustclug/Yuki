.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: unit-test
unit-test:
	go test -race -v ./pkg/...
