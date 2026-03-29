# Makefile for apitool
#
# Usage:
#   make build           Build for the current platform
#   make build-all       Cross-compile for all supported platforms
#   make test            Run all tests
#   make clean           Remove build output

# Binary name
BIN := apitool

# Version metadata — override at build time or detect from git.
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     ?= $(shell date -u +%Y-%m-%d)

# Build flags
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(DATE)
GOFLAGS := -trimpath

# Output directory for cross-compiled binaries
DIST := dist

.PHONY: build build-all test clean

## build: compile for the current platform
build:
	CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/apitool

## build-all: cross-compile for Linux, macOS, and Windows
build-all: clean
	mkdir -p $(DIST)
	GOOS=linux   GOARCH=amd64  CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-linux-amd64      ./cmd/apitool
	GOOS=linux   GOARCH=arm64  CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-linux-arm64      ./cmd/apitool
	GOOS=darwin  GOARCH=amd64  CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-darwin-amd64     ./cmd/apitool
	GOOS=darwin  GOARCH=arm64  CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-darwin-arm64     ./cmd/apitool
	GOOS=windows GOARCH=amd64  CGO_ENABLED=0 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(DIST)/$(BIN)-windows-amd64.exe ./cmd/apitool

## test: run all tests
test:
	go test ./...

## clean: remove compiled output
clean:
	rm -f $(BIN)
	rm -rf $(DIST)
