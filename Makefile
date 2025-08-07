# Simple Makefile for skald-go
# Following Unix philosophy: do one thing well

BINARY = skald
INSTALL_PREFIX ?= /usr/local

# Version from VERSION file
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")

# Go build parameters
GOFLAGS = -trimpath
LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: all build clean install uninstall run test test-coverage test-verbose version release tag deps

all: build

# Build whisper.cpp as static library
deps:
	@echo "Building whisper.cpp static libraries..."
	@cd deps/whisper.cpp && \
		cmake -B build -DBUILD_SHARED_LIBS=OFF -DWHISPER_BUILD_EXAMPLES=OFF -DWHISPER_BUILD_TESTS=OFF -DCMAKE_BUILD_TYPE=Release && \
		cmake --build build --config Release

# Build static binary with no external dependencies
build: deps
	@echo "Building $(BINARY)..."
	@CGO_ENABLED=1 \
		CGO_CFLAGS="-I$(PWD)/deps/whisper.cpp/include -I$(PWD)/deps/whisper.cpp/ggml/include" \
		CGO_LDFLAGS="-L$(PWD)/deps/whisper.cpp/build/src -L$(PWD)/deps/whisper.cpp/build/ggml/src -lwhisper -lggml -lggml-cpu -lggml-base -lm -lstdc++ -static-libgcc -static-libstdc++" \
		go build -a $(GOFLAGS) -ldflags="$(LDFLAGS) -linkmode=external -extldflags=-static" -o bin/$(BINARY) ./cmd/skald

clean:
	@echo "Cleaning..."
	@rm -f bin/$(BINARY)
	@rm -f *.out coverage* *coverage*.html
	@cd deps/whisper.cpp && rm -rf build

install: build
	@echo "Installing $(BINARY) to $(INSTALL_PREFIX)/bin..."
	@install -D bin/$(BINARY) $(INSTALL_PREFIX)/bin/$(BINARY)
	@echo "Installation complete. You can now run: $(BINARY)"

uninstall:
	@echo "Removing $(BINARY)..."
	@rm -f $(INSTALL_PREFIX)/bin/$(BINARY)

run: build
	@./bin/$(BINARY)

# Download models
download-model:
	@echo "Downloading large-v3-turbo model (default)..."
	@mkdir -p models
	@wget -nc -P models https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo.bin

download-tiny-model:
	@echo "Downloading tiny model (alternative, faster but less accurate)..."
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
release: clean deps
	@echo "Building release version $(VERSION)..."
	@CGO_ENABLED=1 \
		CGO_CFLAGS="-I$(PWD)/deps/whisper.cpp/include -I$(PWD)/deps/whisper.cpp/ggml/include" \
		CGO_LDFLAGS="-L$(PWD)/deps/whisper.cpp/build/src -L$(PWD)/deps/whisper.cpp/build/ggml/src -lwhisper -lggml -lggml-cpu -lggml-base -lm -lstdc++ -static-libgcc -static-libstdc++" \
		go build -a $(GOFLAGS) -ldflags="$(LDFLAGS) -linkmode=external -extldflags=-static" -o bin/$(BINARY) ./cmd/skald
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