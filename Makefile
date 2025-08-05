# Simple Makefile for skald-go
# Following Unix philosophy: do one thing well

BINARY = skald
INSTALL_PREFIX ?= /usr/local

# Version from VERSION file
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")

# Go build parameters
GOFLAGS = -trimpath
LDFLAGS = -s -w -X main.version=$(VERSION)

# Library path for embedded RPATH (relative to binary location)
LIB_PATH = \$$ORIGIN/../lib

.PHONY: all build clean install uninstall run test test-coverage test-verbose version release tag

all: build

build:
	@echo "Building $(BINARY)..."
	@CGO_ENABLED=1 CGO_LDFLAGS="-Wl,-rpath,$(LIB_PATH)" go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/skald

clean:
	@echo "Cleaning..."
	@rm -f bin/$(BINARY)

install: build
	@echo "Installing $(BINARY) to $(INSTALL_PREFIX)/bin..."
	@install -D bin/$(BINARY) $(INSTALL_PREFIX)/bin/$(BINARY)
	@echo "Installing libraries to $(INSTALL_PREFIX)/lib..."
	@install -D lib/*.so* $(INSTALL_PREFIX)/lib/
	@echo "Updating library cache..."
	@ldconfig
	@echo "Installation complete. You can now run: $(BINARY)"

uninstall:
	@echo "Removing $(BINARY)..."
	@rm -f $(INSTALL_PREFIX)/bin/$(BINARY)

run: build
	@./bin/$(BINARY) -model models/ggml-large-v3-turbo-q8_0.bin

# Download base model (optional - large model is default)
download-base-model:
	@echo "Downloading base model..."
	@mkdir -p models
	@wget -nc -P models https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin

# Download small model (faster than base)
download-tiny-model:
	@echo "Downloading tiny model..."
	@mkdir -p models
	@wget -nc -P models https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin

# Run all tests
test:
	@echo "Running tests..."
	@go test ./pkg/skald/...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./pkg/skald/...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | grep total
	@echo "Coverage report generated: coverage.html"

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	@go test -v ./pkg/skald/...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./pkg/skald/...

# Show current version
version:
	@echo "skald version $(VERSION)"

# Create a release build with version info
release: clean
	@echo "Building release version $(VERSION)..."
	@CGO_ENABLED=1 CGO_LDFLAGS="-Wl,-rpath,$(LIB_PATH)" go build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/skald
	@echo "Release $(VERSION) built successfully"

# Tag and create a git release
tag:
	@echo "Creating git tag v$(VERSION)..."
	@git tag -a v$(VERSION) -m "Release v$(VERSION)"
	@echo "Tag v$(VERSION) created. Push with: git push origin v$(VERSION)"

help:
	@echo "Usage:"
	@echo "  make build              - Build the binary"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make install            - Install binary and libraries"
	@echo "  make uninstall          - Remove installed files"
	@echo "  make run                - Build and run with default model"
	@echo "  make download-base-model - Download base whisper model"
	@echo "  make download-tiny-model - Download tiny whisper model"
	@echo "  make test               - Run all tests"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo "  make test-verbose       - Run tests with verbose output"
	@echo "  make bench              - Run benchmarks"
	@echo "  make version            - Show current version"
	@echo "  make release            - Build release with version info"
	@echo "  make tag                - Create git tag for current version"
	@echo "  make help               - Show this help"