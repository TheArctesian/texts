#!/bin/bash

# Setup script for Ollama and vision models
set -e

echo "Book Vision Processor Setup"
echo "==========================="
echo ""

# Check if Ollama is installed
if ! command -v ollama &> /dev/null; then
    echo "Ollama not found. Installing..."
    curl -fsSL https://ollama.com/install.sh | sh
    echo "✓ Ollama installed successfully"
else
    echo "✓ Ollama is already installed"
fi

# Start Ollama service
echo "Starting Ollama service..."
if pgrep -x "ollama" > /dev/null; then
    echo "✓ Ollama is already running"
else
    ollama serve &
    sleep 5
    echo "✓ Ollama service started"
fi

# Check Ollama connectivity
echo "Checking Ollama API..."
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "✓ Ollama API is accessible"
else
    echo "✗ Cannot connect to Ollama API"
    echo "Please ensure Ollama is running: ollama serve"
    exit 1
fi

# List available models
echo ""
echo "Currently installed models:"
ollama list

# Ask which model to install
echo ""
echo "Which vision model would you like to use?"
echo "1) gemma2:27b (27B params, good balance of speed and accuracy)"
echo "2) llava:34b (34B params, excellent vision capabilities)"
echo "3) llava:13b (13B params, faster, good for testing)"
echo "4) bakllava:34b (34B params, alternative vision model)"
echo "5) Skip model installation"
echo ""
read -p "Enter choice (1-5): " choice

case $choice in
    1)
        echo "Pulling Gemma 2 27B model (this may take a while)..."
        ollama pull gemma2:27b
        MODEL="gemma2:27b"
        ;;
    2)
        echo "Pulling LLaVA 34B model (this may take a while)..."
        ollama pull llava:34b
        MODEL="llava:34b"
        ;;
    3)
        echo "Pulling LLaVA 13B model..."
        ollama pull llava:13b
        MODEL="llava:13b"
        ;;
    4)
        echo "Pulling BakLLaVA 34B model (this may take a while)..."
        ollama pull bakllava:34b
        MODEL="bakllava:34b"
        ;;
    5)
        echo "Skipping model installation"
        MODEL=""
        ;;
    *)
        echo "Invalid choice"
        exit 1
        ;;
esac

if [ -n "$MODEL" ]; then
    echo "✓ Model $MODEL is ready"
    
    # Test the model with a simple prompt
    echo ""
    echo "Testing model..."
    echo "Describe an image of a book cover" | ollama run $MODEL "You are a helpful assistant. Reply in one sentence." 2>/dev/null
    echo "✓ Model test successful"
fi

# Build the Go application
echo ""
echo "Building the vision processor..."
if command -v go &> /dev/null; then
    go build -o book-vision ./cmd/vision-main.go
    echo "✓ Vision processor built successfully"
else
    echo "Go is not installed. Please install Go 1.21+ to build the processor"
    echo "Visit: https://go.dev/dl/"
fi

echo ""
echo "Setup complete!"
echo ""
echo "To process your book images, run:"
if [ -n "$MODEL" ]; then
    echo "  ./book-vision -model $MODEL"
else
    echo "  ./book-vision -model <your-model>"
fi
echo ""
echo "Or use the Makefile:"
echo "  make process-all"
echo ""
echo "For help:"
echo "  ./book-vision -help"