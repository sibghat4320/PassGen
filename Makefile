# Makefile for passgen. All targets rely only on the Go toolchain, so they work
# the same on Windows, Linux and macOS.

BINARY := passgen
PKG := ./cmd/passgen
BIN_DIR := bin
COVERAGE := coverage.out

.PHONY: all build test vet fmt fmt-check race cover check run tui clean

all: check build

build:
	go build -o $(BIN_DIR)/$(BINARY) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

# Fails when any file is not gofmt formatted.
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

race:
	go test -race ./...

cover:
	go test -coverprofile=$(COVERAGE) ./...
	go tool cover -func=$(COVERAGE)

check: fmt-check vet test

run:
	go run $(PKG)

tui:
	go run $(PKG) --interactive

clean:
	go clean
	rm -rf $(BIN_DIR) $(COVERAGE) $(BINARY) $(BINARY).exe
