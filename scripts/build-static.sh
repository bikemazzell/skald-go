#!/bin/bash

# Script to build a fully static binary with whisper library compiled in
# This creates a standalone executable with no external dependencies

set -e

echo "Building static binary..."

# Set paths with absolute paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WHISPER_PATH="$SCRIPT_DIR/deps/whisper.cpp"
INCLUDE_PATH="$WHISPER_PATH/include:$WHISPER_PATH/ggml/include:$WHISPER_PATH/ggml/src"
BUILD_DIR="$WHISPER_PATH/build-static"
LIB_DIR="$SCRIPT_DIR/lib"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Create lib directory if it doesn't exist
mkdir -p "$LIB_DIR"

# Create build directory for whisper.cpp
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

# Build whisper.cpp with static libraries
echo "Building whisper.cpp static libraries..."
cmake -DCMAKE_POSITION_INDEPENDENT_CODE=ON -DBUILD_SHARED_LIBS=OFF -DGGML_STATIC=ON ..
make -j$(nproc)

# Check if the static libraries were built
if [ -f "src/libwhisper.a" ] && [ -f "ggml/src/libggml.a" ]; then
    echo "Static libraries built successfully"
    # Copy the static libraries to the lib directory
    cp src/libwhisper.a "$LIB_DIR/"
    cp ggml/src/libggml.a "$LIB_DIR/"
else
    echo "Failed to build static whisper libraries"
    exit 1
fi

cd "$SCRIPT_DIR"

# Create a combined static library
echo "Creating combined static library..."
mkdir -p "$TEMP_DIR/whisper_combined"
cd "$TEMP_DIR/whisper_combined"

# Extract object files from both libraries
ar x "$LIB_DIR/libwhisper.a"
ar x "$LIB_DIR/libggml.a"

# Create a new combined library
ar rcs "$LIB_DIR/libwhisper_combined.a" *.o

cd "$SCRIPT_DIR"

# Build Go application with static whisper library
echo "Building Go application with static whisper library..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the server
CGO_ENABLED=1 \
C_INCLUDE_PATH="$INCLUDE_PATH" \
CGO_CXXFLAGS="-std=c++11" \
CGO_LDFLAGS="-L$LIB_DIR -lwhisper_combined -lm -lstdc++ -fopenmp -static-libstdc++ -static-libgcc" \
go build -tags static -ldflags '-extldflags "-static"' -o bin/skald-server cmd/service/main.go

# Build the client
CGO_ENABLED=1 \
C_INCLUDE_PATH="$INCLUDE_PATH" \
CGO_CXXFLAGS="-std=c++11" \
CGO_LDFLAGS="-L$LIB_DIR -lwhisper_combined -lm -lstdc++ -fopenmp -static-libstdc++ -static-libgcc" \
go build -tags static -ldflags '-extldflags "-static"' -o bin/skald-client cmd/client/main.go

echo "Build completed. Binaries are in bin/"

# Check if the binaries are statically linked
echo "Checking if binaries are statically linked:"
ldd bin/skald-server || echo "Binary is statically linked!"
ldd bin/skald-client || echo "Binary is statically linked!" 