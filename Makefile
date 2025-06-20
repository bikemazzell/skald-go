.PHONY: debug release clean test install uninstall

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

# Uninstall system-wide installation
uninstall:
	@echo "🗑️  Uninstalling Skald-Go..."
	@# Stop and disable systemd service if it exists
	@if systemctl --user is-active skald.service >/dev/null 2>&1; then \
		echo "🛑 Stopping systemd service..."; \
		systemctl --user stop skald.service; \
	fi
	@if systemctl --user is-enabled skald.service >/dev/null 2>&1; then \
		echo "🚫 Disabling systemd service..."; \
		systemctl --user disable skald.service; \
	fi
	@# Stop any running skald processes
	@if pgrep -f "skald-server" >/dev/null 2>&1; then \
		echo "🛑 Stopping running skald-server processes..."; \
		pkill -TERM -f "skald-server" || true; \
		sleep 2; \
		if pgrep -f "skald-server" >/dev/null 2>&1; then \
			echo "🔨 Force killing remaining skald-server processes..."; \
			pkill -KILL -f "skald-server" || true; \
		fi; \
	fi
	@# Clean up socket and state files
	@echo "🧹 Cleaning up runtime files..."
	@rm -f /tmp/skald.sock
	@rm -f /tmp/skald-continuous-state
	@# Remove installation directory
	@if [ -d ~/.local/bin/skald-go ]; then \
		echo "📁 Removing installation directory..."; \
		rm -rf ~/.local/bin/skald-go; \
	fi
	@# Remove PATH entry from .bashrc
	@if [ -f ~/.bashrc ] && grep -q "\.local/bin/skald-go" ~/.bashrc; then \
		echo "🔧 Removing PATH entry from ~/.bashrc..."; \
		sed -i '/export PATH.*\.local\/bin\/skald-go/d' ~/.bashrc; \
	fi
	@# Reload systemd user daemon to remove service references
	@systemctl --user daemon-reload >/dev/null 2>&1 || true
	@echo "✅ Uninstall complete!"
	@echo "💡 Restart terminal or run: source ~/.bashrc to update PATH"
	@echo "💡 Any downloaded models in the old directory have been removed"

# Show help
help:
	@echo "Skald-Go Build System"
	@echo ""
	@echo "Commands:"
	@echo "  debug     - Fast build for development (requires dependencies)"
	@echo "  release   - Self-contained binary for distribution"
	@echo "  test      - Run test suite"
	@echo "  clean     - Remove all build artifacts"
	@echo "  install   - Install release binary system-wide"
	@echo "  uninstall - Remove system-wide installation completely"
	@echo "  deps      - Install/update dependencies (for debug)"
	@echo ""
	@echo "Examples:"
	@echo "  make debug     # Quick development build"
	@echo "  make release   # Production build"
	@echo "  make install   # Install for daily use"
	@echo "  make uninstall # Remove installation"