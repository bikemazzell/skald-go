#!/bin/bash

# Script to update all external dependencies
# Usage: ./update_deps.sh [whisper_tag]
# If whisper_tag is not provided, it will update to the latest tag

set -e

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Create deps directory if it doesn't exist
mkdir -p deps

# Check if a specific whisper tag was provided
WHISPER_TAG=${1:-latest}

echo "Updating dependencies..."

# Step 1: Update whisper.cpp
echo "Updating whisper.cpp to $WHISPER_TAG..."

# Check if whisper.cpp directory exists
if [ ! -d "deps/whisper.cpp" ]; then
  echo "Cloning whisper.cpp repository..."
  git clone https://github.com/ggerganov/whisper.cpp.git deps/whisper.cpp
fi

# Update whisper.cpp
cd deps/whisper.cpp

# Make sure we're on the master branch
git checkout master

# Fetch the latest changes
git fetch origin

if [ "$WHISPER_TAG" = "latest" ]; then
  # Get the latest tag
  WHISPER_TAG=$(git tag -l | grep -v "pre" | sort -V | tail -n 1)
  echo "Latest whisper.cpp tag is $WHISPER_TAG"
fi

# Stash any local changes
git stash

# Checkout the specified tag
git checkout $WHISPER_TAG

# Check if cmake is installed
if command -v cmake &> /dev/null; then
  echo "Building whisper.cpp library..."
  # Try to build the library
  if ! cmake -B build -DBUILD_SHARED_LIBS=OFF && cmake --build build --config Release; then
    echo "Warning: Failed to build whisper.cpp library with CMake."
    echo "You may need to build it manually:"
    echo "  cd deps/whisper.cpp"
    echo "  cmake -B build -DBUILD_SHARED_LIBS=OFF"
    echo "  cmake --build build --config Release"
  fi
else
  echo "Warning: CMake not found. Cannot build whisper.cpp library automatically."
  echo "You need to install CMake and build the library manually:"
  echo "  cd deps/whisper.cpp"
  echo "  cmake -B build -DBUILD_SHARED_LIBS=OFF"
  echo "  cmake --build build --config Release"
fi

cd ../..

# Step 2: Update Go bindings
echo "Updating Go bindings..."

# Create whisper-go directory if it doesn't exist
mkdir -p deps/whisper-go

# Copy Go bindings from whisper.cpp
rsync -av --delete deps/whisper.cpp/bindings/go/ deps/whisper-go/

# Copy header files to the Go bindings directory
echo "Copying header files to the Go bindings directory..."

# Copy whisper.h
echo "Copying whisper.h to the Go bindings directory..."
cp deps/whisper.cpp/include/whisper.h deps/whisper-go/

# Copy ggml.h
echo "Copying ggml.h to the Go bindings directory..."
cp deps/whisper.cpp/ggml/include/ggml.h deps/whisper-go/

# Copy ggml-cpu.h
echo "Copying ggml-cpu.h to the Go bindings directory..."
cp deps/whisper.cpp/ggml/include/ggml-cpu.h deps/whisper-go/

# Copy ggml-backend.h
echo "Copying ggml-backend.h to the Go bindings directory..."
cp deps/whisper.cpp/ggml/include/ggml-backend.h deps/whisper-go/

# Copy ggml-alloc.h
echo "Copying ggml-alloc.h to the Go bindings directory..."
cp deps/whisper.cpp/ggml/include/ggml-alloc.h deps/whisper-go/

# Check if there are any other header files needed by the Go bindings
for header in $(grep -r "#include" deps/whisper-go --include="*.h" --include="*.go" | grep -o '"[^"]*\.h"' | tr -d '"' | sort | uniq); do
  if [ -f "deps/whisper.cpp/ggml/include/$header" ]; then
    echo "Copying $header to the Go bindings directory..."
    cp "deps/whisper.cpp/ggml/include/$header" deps/whisper-go/
  elif [ -f "deps/whisper.cpp/include/$header" ]; then
    echo "Copying $header to the Go bindings directory..."
    cp "deps/whisper.cpp/include/$header" deps/whisper-go/
  fi
done

# Step 3: Update Go module dependencies
echo "Updating Go module dependencies..."
cd deps/whisper-go
go mod tidy
go mod vendor
cd ../..

# Step 4: Update go.mod to use the local whisper-go
echo "Updating go.mod..."
# Check if the replace directive exists
if ! grep -q "github.com/ggerganov/whisper.cpp/bindings/go => ./deps/whisper-go" go.mod; then
  # Remove any existing replace directive for whisper.cpp bindings
  sed -i '/replace github.com\/ggerganov\/whisper.cpp\/bindings\/go/d' go.mod
  # Add the new replace directive
  echo "replace github.com/ggerganov/whisper.cpp/bindings/go => ./deps/whisper-go" >> go.mod
fi

# Check if the replace directive for whisper.cpp exists
if ! grep -q "github.com/ggerganov/whisper.cpp => ./deps/whisper.cpp" go.mod; then
  # Remove any existing replace directive for whisper.cpp
  sed -i '/replace github.com\/ggerganov\/whisper.cpp =>/d' go.mod
  # Add the new replace directive
  echo "replace github.com/ggerganov/whisper.cpp => ./deps/whisper.cpp" >> go.mod
fi

# Step 5: Update vendor directory
echo "Updating vendor directory..."
go mod tidy
go mod vendor

# Copy header files to the vendor directory
echo "Copying header files to the vendor directory..."

# Copy whisper.h
echo "Copying whisper.h to the vendor directory..."
cp deps/whisper.cpp/include/whisper.h vendor/github.com/ggerganov/whisper.cpp/bindings/go/

# Copy ggml.h
echo "Copying ggml.h to the vendor directory..."
cp deps/whisper.cpp/ggml/include/ggml.h vendor/github.com/ggerganov/whisper.cpp/bindings/go/

# Copy ggml-cpu.h
echo "Copying ggml-cpu.h to the vendor directory..."
cp deps/whisper.cpp/ggml/include/ggml-cpu.h vendor/github.com/ggerganov/whisper.cpp/bindings/go/

# Copy ggml-backend.h
echo "Copying ggml-backend.h to the vendor directory..."
cp deps/whisper.cpp/ggml/include/ggml-backend.h vendor/github.com/ggerganov/whisper.cpp/bindings/go/

# Copy ggml-alloc.h
echo "Copying ggml-alloc.h to the vendor directory..."
cp deps/whisper.cpp/ggml/include/ggml-alloc.h vendor/github.com/ggerganov/whisper.cpp/bindings/go/

# Copy any other header files needed by the Go bindings
for header in $(grep -r "#include" vendor/github.com/ggerganov/whisper.cpp/bindings/go --include="*.h" --include="*.go" | grep -o '"[^"]*\.h"' | tr -d '"' | sort | uniq); do
  if [ -f "deps/whisper.cpp/ggml/include/$header" ]; then
    echo "Copying $header to the vendor directory..."
    cp "deps/whisper.cpp/ggml/include/$header" vendor/github.com/ggerganov/whisper.cpp/bindings/go/
  elif [ -f "deps/whisper.cpp/include/$header" ]; then
    echo "Copying $header to the vendor directory..."
    cp "deps/whisper.cpp/include/$header" vendor/github.com/ggerganov/whisper.cpp/bindings/go/
  fi
done

# Check if whisper library exists
if [ -f "deps/whisper.cpp/build/src/libwhisper.so" ]; then
  echo "Found shared library: deps/whisper.cpp/build/src/libwhisper.so"
  
  # Create lib directory if it doesn't exist
  mkdir -p lib
  
  # Copy the shared library to the lib directory
  echo "Copying libwhisper.so to lib directory..."
  cp deps/whisper.cpp/build/src/libwhisper.so lib/
  cp deps/whisper.cpp/build/src/libwhisper.so.1 lib/
  cp deps/whisper.cpp/build/src/libwhisper.so.1.7.4 lib/
  
  echo "To use the shared library, you may need to set LD_LIBRARY_PATH:"
  echo "  export LD_LIBRARY_PATH=\$LD_LIBRARY_PATH:$(pwd)/lib"
elif [ -f "deps/whisper.cpp/build/libwhisper.a" ]; then
  echo "Found static library: deps/whisper.cpp/build/libwhisper.a"
  
  # Create lib directory if it doesn't exist
  mkdir -p lib
  
  # Copy the static library to the lib directory
  echo "Copying libwhisper.a to lib directory..."
  cp deps/whisper.cpp/build/libwhisper.a lib/
else
  echo "Warning: whisper library not found. You need to build it manually:"
  echo "  cd deps/whisper.cpp"
  echo "  cmake -B build -DBUILD_SHARED_LIBS=OFF"
  echo "  cmake --build build --config Release"
  echo "This is required for the Go bindings to work properly."
fi

echo "Update completed successfully!"
echo "whisper.cpp updated to version $WHISPER_TAG"

# Cleanup old directories if they exist and are not needed
if [ -d "external/whisper.cpp" ]; then
  echo "Note: You can remove the old external/whisper.cpp directory if it's no longer needed."
fi

if [ -d "bindings/go" ]; then
  echo "Note: You can remove the old bindings/go directory if it's no longer needed."
fi 