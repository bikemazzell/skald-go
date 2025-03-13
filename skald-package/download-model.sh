#!/bin/bash

# Script to download whisper models for Skald-Go
# This script reads the available models from config.json and allows the user to download them

set -e

# Get the directory of the script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required for this script to work."
    echo "Please install jq using your package manager:"
    echo "  - Ubuntu/Debian: sudo apt install jq"
    echo "  - Fedora: sudo dnf install jq"
    echo "  - Arch Linux: sudo pacman -S jq"
    echo "  - macOS: brew install jq"
    exit 1
fi

# Check if config.json exists
if [ ! -f "config.json" ]; then
    echo "Error: config.json not found in the current directory."
    echo "Please run this script from the Skald-Go root directory."
    exit 1
fi

# Create models directory if it doesn't exist
mkdir -p models

# Function to display download progress
display_progress() {
    local current=$1
    local total=$2
    local percent=$((current * 100 / total))
    local progress=$((percent / 2))
    local remaining=$((50 - progress))
    
    printf "\r[%-${progress}s%-${remaining}s] %d%%" "$(printf "%0.s=" $(seq 1 $progress))" "$(printf "%0.s " $(seq 1 $remaining))" $percent
}

# Function to download a model
download_model() {
    local model_name=$1
    local model_url=$2
    local model_size=$3
    
    echo "Downloading model: $model_name ($model_size)"
    echo "URL: $model_url"
    
    local output_file="models/ggml-$model_name.bin"
    
    # Check if the model already exists
    if [ -f "$output_file" ]; then
        echo "Model already exists at $output_file"
        read -p "Do you want to redownload it? (y/n): " choice
        if [[ ! $choice =~ ^[Yy]$ ]]; then
            echo "Download skipped."
            return
        fi
    fi
    
    echo "Downloading to $output_file..."
    
    # Download with curl if available (with progress bar)
    if command -v curl &> /dev/null; then
        curl -L --progress-bar "$model_url" -o "$output_file"
    # Otherwise use wget if available
    elif command -v wget &> /dev/null; then
        wget -q --show-progress "$model_url" -O "$output_file"
    # Fall back to a basic download with progress using dd and pv if available
    else
        echo "Neither curl nor wget found. Using basic HTTP client."
        
        # Create a temporary file for the download
        local temp_file=$(mktemp)
        
        # Start the download
        local response=$(wget --server-response --spider "$model_url" 2>&1)
        local size=$(echo "$response" | grep -i "Content-Length" | awk '{print $2}' | tail -1)
        
        if [ -z "$size" ]; then
            size=0
        fi
        
        # Download with progress
        {
            downloaded=0
            exec 3>&1
            exec 4>&2
            exec 1>/dev/null
            exec 2>/dev/null
            
            # Use Python if available for the download
            if command -v python3 &> /dev/null; then
                python3 -c "
import urllib.request
import sys

class ProgressReporter:
    def __init__(self, total):
        self.total = total
        self.downloaded = 0
    
    def report(self, block_count, block_size, total_size):
        self.downloaded += block_size
        if self.total > 0:
            percent = min(int(self.downloaded * 100 / self.total), 100)
            sys.stderr.write(f'\r{percent}%')
            sys.stderr.flush()

reporter = ProgressReporter($size)
urllib.request.urlretrieve('$model_url', '$output_file', reporter.report)
sys.stderr.write('\n')
"
            else
                # Fallback to a very basic download without progress
                wget -q "$model_url" -O "$output_file"
            fi
            
            exec 1>&3
            exec 2>&4
        }
    fi
    
    # Verify the download
    if [ -f "$output_file" ]; then
        echo "Download completed successfully."
    else
        echo "Error: Download failed."
        exit 1
    fi
}

# Extract model information from config.json
echo "Reading available models from config.json..."
models=$(jq -r '.whisper.models | to_entries | .[] | "\(.key)|\(.value.url)|\(.value.size)"' config.json)

# Display available models
echo "Available models:"
echo "----------------"
i=1
declare -A model_map
while IFS='|' read -r name url size; do
    echo "$i) $name ($size)"
    model_map[$i]="$name|$url|$size"
    i=$((i+1))
done <<< "$models"
echo "----------------"

# Ask user which model to download
read -p "Enter the number of the model to download (or 'a' for all): " choice

# Download selected model(s)
if [[ $choice == "a" || $choice == "A" ]]; then
    echo "Downloading all models..."
    while IFS='|' read -r name url size; do
        download_model "$name" "$url" "$size"
    done <<< "$models"
else
    if [[ ! $choice =~ ^[0-9]+$ ]] || [ $choice -lt 1 ] || [ $choice -ge $i ]; then
        echo "Invalid choice. Exiting."
        exit 1
    fi
    
    IFS='|' read -r name url size <<< "${model_map[$choice]}"
    download_model "$name" "$url" "$size"
fi

echo "Model download(s) completed."
echo "You can now use the model with Skald-Go."
echo "Run ./run-server.sh to start the server." 