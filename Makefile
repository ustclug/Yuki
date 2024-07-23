.PHONY: all
all: yukid yukictl

.PHONY: clean
clean:
	rm -f yukid yukictl *.deb

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

.PHONY: deb

deb_dir := $(shell mktemp -d)
deb: | yukid yukictl
	install -Dm644 etc/daemon.example.toml -t $(deb_dir)/etc/yuki
	install -Dm644 etc/yukid.service -t $(deb_dir)/lib/systemd/system
	install -Dm755 yukid yukictl -t $(deb_dir)/usr/local/bin
	ln -s yukictl $(deb_dir)/usr/local/bin/yuki
	install -d $(addprefix $(deb_dir), /etc/bash_completion.d /DEBIAN)
	$(deb_dir)/usr/local/bin/yukictl completion bash > $(deb_dir)/etc/bash_completion.d/yukictl
	sed "s/\$$VERSION\>/$(version:v%=%)/g;s/\$$ARCH\>/$(shell go env GOARCH)/g" \
		etc/debian-control > $(deb_dir)/DEBIAN/control
	dpkg-deb --root-owner-group --build -Zxz $(deb_dir) .
