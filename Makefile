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
	mkdir -p $(addprefix $(deb_dir)/, DEBIAN etc/bash_completion.d etc/yuki lib/systemd/system usr/local/bin)
	cp etc/daemon.example.toml $(deb_dir)/etc/yuki
	cp etc/yukid.service $(deb_dir)/lib/systemd/system
	cp yukid yukictl $(deb_dir)/usr/local/bin
	ln -s yukictl $(deb_dir)/usr/local/bin/yuki
	$(deb_dir)/usr/local/bin/yukictl completion bash > $(deb_dir)/etc/bash_completion.d/yukictl
	sed "s/\$$VERSION\>/$(version)/g;s/^Version: v/Version: /g;s/\$$ARCH\>/$(shell go env GOARCH)/g" \
		etc/debian-control > $(deb_dir)/DEBIAN/control
	dpkg-deb --root-owner-group --build -Zxz $(deb_dir) .
