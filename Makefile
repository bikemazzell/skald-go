# Simple Makefile for skald-go
# Following Unix philosophy: do one thing well

BINARY = skald
INSTALL_PREFIX ?= /usr/local

# Version from VERSION file
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")

# Go build parameters
GOFLAGS = -trimpath
LDFLAGS = -s -w -X main.version=$(VERSION)

# Upstream for whisper.cpp (used only if repo exists as git)
WHISPER_CPP_DIR := deps/whisper.cpp
WHISPER_CPP_REMOTE ?= https://github.com/ggerganov/whisper.cpp.git

# Control how aggressively to update Go deps: -u (minor+patch) or -u=patch
GOGETFLAGS ?= -u

.PHONY: all build clean install uninstall run test test-coverage test-verbose version release tag deps \
	update-deps update-go-deps update-whisper-cpp

all: build

# Build whisper.cpp as static library
deps: update-go-deps
	@echo "Building whisper.cpp static libraries..."
	@cd deps/whisper.cpp && \
		cmake -B build -DBUILD_SHARED_LIBS=OFF -DWHISPER_BUILD_EXAMPLES=OFF -DWHISPER_BUILD_TESTS=OFF -DCMAKE_BUILD_TYPE=Release -DCMAKE_WARN_DEPRECATED=OFF && \
		cmake --build build --config Release

# Build static binary with no external dependencies
build: deps
	@echo "Building $(BINARY)..."
	@CGO_ENABLED=1 \
		CGO_CFLAGS="-I$(PWD)/deps/whisper.cpp/include -I$(PWD)/deps/whisper.cpp/ggml/include" \
		CGO_LDFLAGS="-L$(PWD)/deps/whisper.cpp/build/src -L$(PWD)/deps/whisper.cpp/build/ggml/src -lwhisper -lggml -lggml-cpu -lggml-base -lm -lstdc++" \
		go build -a $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/skald

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
		CGO_LDFLAGS="-L$(PWD)/deps/whisper.cpp/build/src -L$(PWD)/deps/whisper.cpp/build/ggml/src -lwhisper -lggml -lggml-cpu -lggml-base -lm -lstdc++" \
		go build -a $(GOFLAGS) -ldflags="$(LDFLAGS)" -o bin/$(BINARY) ./cmd/skald
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
	@echo "  make deps               - Update Go modules and build whisper.cpp libraries"
	@echo "  make update-deps        - Update Go modules, vendor, and whisper.cpp (if git)"

# Update Go module dependencies and vendor folder
update-go-deps:
	@echo "Updating Go module dependencies (flags: $(GOGETFLAGS))..."
	@go get $(GOGETFLAGS) ./...
	@echo "Tidying go.mod/go.sum..."
	@go mod tidy
	@echo "Refreshing vendor directory..."
	@go mod vendor
	@echo "Go dependencies updated."

# Update whisper.cpp if it is a git checkout
update-whisper-cpp:
	@if [ -d "$(WHISPER_CPP_DIR)/.git" ]; then \
		echo "Updating whisper.cpp from git remote..."; \
		git -C "$(WHISPER_CPP_DIR)" fetch --tags --prune; \
		# Determine default remote branch (origin/HEAD), fallback to origin/master; strip 'origin/' prefix to get local branch name \
		DEF_REMOTE=$$(git -C "$(WHISPER_CPP_DIR)" symbolic-ref -q --short refs/remotes/origin/HEAD || echo origin/master); \
		DEF_BRANCH=$${DEF_REMOTE#origin/}; \
		echo "Switching to branch '$${DEF_BRANCH}'..."; \
		git -C "$(WHISPER_CPP_DIR)" checkout "$${DEF_BRANCH}" 2>/dev/null || git -C "$(WHISPER_CPP_DIR)" checkout master; \
		echo "Pulling latest commits..."; \
		git -C "$(WHISPER_CPP_DIR)" pull --ff-only origin "$${DEF_BRANCH}" 2>/dev/null || git -C "$(WHISPER_CPP_DIR)" pull --ff-only origin master; \
		echo "Rebuilding whisper.cpp..."; \
		$(MAKE) deps; \
	else \
		echo "Skipping whisper.cpp update: $(WHISPER_CPP_DIR) is not a git checkout."; \
		echo "Hint: clone it with git to enable updates (remote: $(WHISPER_CPP_REMOTE))."; \
	fi

# Convenience target: update everything
update-deps: update-go-deps update-whisper-cpp
	@echo "All dependencies updated."