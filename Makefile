.PHONY: build clean run test deps

# Binary names and build directory
BINARY_NAME=skald-server
CLIENT_NAME=skald-client
BUILD_DIR=bin

# Version from git tag (with fallback)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO_LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Whisper paths
WHISPER_PATH=$(shell pwd)/deps/whisper.cpp
INCLUDE_PATH=${WHISPER_PATH}/include:${WHISPER_PATH}/ggml/include
LIBRARY_PATH=${WHISPER_PATH}

# Library paths
LIB_PATH=$(shell pwd)/lib

# CGO flags for static compilation
CGO_ENV=C_INCLUDE_PATH=${INCLUDE_PATH} LIBRARY_PATH=${LIBRARY_PATH} LD_LIBRARY_PATH=${LIB_PATH}
CGO_FLAGS=CGO_ENABLED=1 ${CGO_ENV} CGO_LDFLAGS="-L${LIB_PATH} -lwhisper -lm -lstdc++ -fopenmp"

# Build both server and client
build:
	mkdir -p $(BUILD_DIR)
	# Build our application
	$(CGO_FLAGS) go build $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/service/main.go
	$(CGO_FLAGS) go build $(GO_LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME) cmd/client/main.go

# Build with static linking (standalone binary)
build-static:
	mkdir -p $(BUILD_DIR)
	# Build with static linking
	CGO_ENABLED=1 C_INCLUDE_PATH=${INCLUDE_PATH} go build \
		-tags netgo \
		$(GO_LDFLAGS) -ldflags '-s -w -extldflags "-static"' \
		-o $(BUILD_DIR)/$(BINARY_NAME) cmd/service/main.go
	CGO_ENABLED=1 C_INCLUDE_PATH=${INCLUDE_PATH} go build \
		-tags netgo \
		$(GO_LDFLAGS) -ldflags '-s -w -extldflags "-static"' \
		-o $(BUILD_DIR)/$(CLIENT_NAME) cmd/client/main.go

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run the service
run: build
	LD_LIBRARY_PATH=${LIB_PATH} ./$(BUILD_DIR)/$(BINARY_NAME)

# Run with static binary (no LD_LIBRARY_PATH needed)
run-static: build-static
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with specific config
run-config: build
	LD_LIBRARY_PATH=${LIB_PATH} ./$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG)

# Install dependencies
deps:
	./scripts/update_deps.sh

# Run tests
test:
	LD_LIBRARY_PATH=${LIB_PATH} go test -v ./...

# Package the application
package: build
	./scripts/package.sh