.PHONY: debug release clean test install

# Version information
VERSION=$(shell cat VERSION 2>/dev/null || echo "0.0.0")
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')

# Determine if this is a dirty build
ifeq ($(shell git diff --quiet; echo $$?), 1)
    VERSION_STRING=${VERSION}-dev+${GIT_COMMIT}-dirty
else
    VERSION_STRING=${VERSION}
endif

# Build flags
GO_LDFLAGS=-ldflags "-X main.version=${VERSION_STRING} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

# Whisper paths for debug builds
WHISPER_PATH=$(shell pwd)/deps/whisper.cpp
INCLUDE_PATH=${WHISPER_PATH}/include:${WHISPER_PATH}/ggml/include
LIB_PATH=$(shell pwd)/lib

# CGO flags for debug builds
DEBUG_CGO_ENV=C_INCLUDE_PATH=${INCLUDE_PATH} LIBRARY_PATH=${LIB_PATH} LD_LIBRARY_PATH=${LIB_PATH}
DEBUG_CGO_FLAGS=CGO_ENABLED=1 ${DEBUG_CGO_ENV} CGO_LDFLAGS="-L${LIB_PATH} -lwhisper -lm -lstdc++ -fopenmp"

# Default target
all: debug

# DEBUG BUILD - Fast compilation, requires dependencies, verbose output
debug: deps
	@echo "🔧 Building DEBUG version..."
	@echo "   Version: $(VERSION_STRING)"
	@echo "   Dependencies: Dynamic linking"
	@echo "   Output: Verbose logging enabled"
	@mkdir -p bin
	$(DEBUG_CGO_FLAGS) go build $(GO_LDFLAGS) -tags debug -o bin/skald-server ./cmd/service
	$(DEBUG_CGO_FLAGS) go build $(GO_LDFLAGS) -tags debug -o bin/skald-client ./cmd/client
	@echo "✅ Debug build complete!"
	@echo "💡 Run with: ./scripts/run-server.sh (sets LD_LIBRARY_PATH)"

# RELEASE BUILD - Self-contained static binary, no dependencies
release: 
	@echo "🚀 Building RELEASE version..."
	@echo "   Version: $(VERSION_STRING)"
	@echo "   Dependencies: Static linking (self-contained)"
	@echo "   Output: Production optimized"
	@./scripts/build-release.sh
	@echo "✅ Release build complete!"
	@echo "📦 Find binaries in: ./release/"

# Install dependencies (only needed for debug builds)
deps:
	@echo "📦 Installing dependencies..."
	@if [ ! -f lib/libwhisper.so ]; then \
		echo "🔨 Building whisper.cpp libraries..."; \
		mkdir -p deps/whisper.cpp/build && \
		cd deps/whisper.cpp/build && \
		cmake -DBUILD_SHARED_LIBS=ON .. && \
		make -j$$(nproc) && \
		cd ../../.. && \
		mkdir -p lib && \
		cp deps/whisper.cpp/build/src/libwhisper.so* lib/ && \
		cp deps/whisper.cpp/build/ggml/src/libggml*.so lib/ && \
		echo "✅ Libraries built successfully"; \
	else \
		echo "✅ Libraries already exist"; \
	fi

# Run tests (debug mode)
test: debug
	@echo "🧪 Running tests..."
	@LD_LIBRARY_PATH=${LIB_PATH} go test -v ./...

# Clean all build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/ release/
	@echo "✅ Clean complete!"

# Install release binary system-wide
install: release
	@echo "📥 Installing Skald-Go system-wide..."
	@mkdir -p ~/.local/bin/skald-go
	@cp release/skald-go-*/skald ~/.local/bin/skald-go/
	@cp release/skald-go-*/config.json ~/.local/bin/skald-go/
	@cp release/skald-go-*/skald.service ~/.local/bin/skald-go/
	@echo "export PATH=\"\$$HOME/.local/bin/skald-go:\$$PATH\"" >> ~/.bashrc
	@echo "✅ Installed to ~/.local/bin/skald-go/"
	@echo "💡 Restart terminal or run: source ~/.bashrc"
	@echo "💡 Then use: skald server, skald start, etc."

# Show help
help:
	@echo "Skald-Go Build System"
	@echo ""
	@echo "Commands:"
	@echo "  debug    - Fast build for development (requires dependencies)"
	@echo "  release  - Self-contained binary for distribution"
	@echo "  test     - Run test suite"
	@echo "  clean    - Remove all build artifacts"
	@echo "  install  - Install release binary system-wide"
	@echo "  deps     - Install/update dependencies (for debug)"
	@echo ""
	@echo "Examples:"
	@echo "  make debug    # Quick development build"
	@echo "  make release  # Production build"
	@echo "  make install  # Install for daily use"