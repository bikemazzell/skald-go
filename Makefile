.PHONY: build clean run test deps

# Binary names and build directory
BINARY_NAME=skald-server
CLIENT_NAME=skald-client
BUILD_DIR=bin

# Version from git tag (with fallback)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO_LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Whisper paths
WHISPER_PATH=$(shell pwd)/external/whisper.cpp
INCLUDE_PATH=${WHISPER_PATH}/include:${WHISPER_PATH}/ggml/include
LIBRARY_PATH=${WHISPER_PATH}

# CGO flags
CGO_ENV=C_INCLUDE_PATH=${INCLUDE_PATH} LIBRARY_PATH=${LIBRARY_PATH}
CGO_FLAGS=CGO_ENABLED=1 ${CGO_ENV} CGO_LDFLAGS="-L${WHISPER_PATH} -lwhisper -lm -lstdc++ -fopenmp"

# Build both server and client
build:
	mkdir -p $(BUILD_DIR)
	# Build whisper.cpp
	cd $(WHISPER_PATH) && make libwhisper.a
	# Build Go bindings
	cd $(WHISPER_PATH)/bindings/go && make
	# Build our application
	$(CGO_FLAGS) go build $(GO_LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/service/main.go
	$(CGO_FLAGS) go build $(GO_LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME) cmd/client/main.go

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Run the service
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Run with specific config
run-config: build
	./$(BUILD_DIR)/$(BINARY_NAME) -config $(CONFIG)

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...