VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo dev)
GO ?= $(shell command -v go 2>/dev/null || echo /opt/data/.local/bin/go)
BINARY := bin/git-stack
LDFLAGS := -ldflags "-X github.com/hermes-do-bruno/GitStack/internal/gitstack.Version=$(VERSION)"

.PHONY: build test clean release-build

build:
	mkdir -p bin
	$(GO) build $(LDFLAGS) -o $(BINARY) ./cmd/git-stack

test:
	$(GO) test ./...

release-build:
	mkdir -p dist
	GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build $(LDFLAGS) -o dist/git-stack ./cmd/git-stack

clean:
	rm -rf bin dist