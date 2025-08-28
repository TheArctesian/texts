FROM ubuntu:22.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install all required dependencies in one layer
RUN apt-get update && apt-get install -y \
    wget \
    git \
    gcc \
    g++ \
    make \
    pkg-config \
    ca-certificates \
    curl \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.21
RUN wget -q https://go.dev/dl/go1.21.13.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.21.13.linux-amd64.tar.gz && \
    rm go1.21.13.linux-amd64.tar.gz

# Install Hugo
RUN wget -q https://github.com/gohugoio/hugo/releases/download/v0.134.3/hugo_0.134.3_linux-amd64.tar.gz && \
    tar -xzf hugo_0.134.3_linux-amd64.tar.gz && \
    mv hugo /usr/local/bin/ && \
    rm hugo_0.134.3_linux-amd64.tar.gz

# Set Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV CGO_ENABLED=1

# Set working directory
WORKDIR /app

# Copy Go module files first for better caching
COPY go.mod go.sum ./

# Download Go dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the LLM-first book processor with enhanced location/date extraction
RUN go build -o book-llm ./cmd/book-llm/main.go

# Create necessary directories
RUN mkdir -p /app/Data/IMGS /app/hugo-site

# Make executable
RUN chmod +x book-llm

# Set environment variables for LLM processing
ENV LLM_PROVIDER=ollama
ENV LLM_MODEL=gemma3:27b
ENV LLM_BASE_URL=http://localhost:11434

# Create a startup script for LLM-first processing
RUN echo '#!/bin/bash' > /app/start-llm.sh && \
    echo 'echo "Starting Book Library with LLM-first approach..."' >> /app/start-llm.sh && \
    echo 'echo "Processing images with LLM analysis..."' >> /app/start-llm.sh && \
    echo '' >> /app/start-llm.sh && \
    echo '# Check if Ollama is available' >> /app/start-llm.sh && \
    echo 'if curl -f -s ${LLM_BASE_URL}/api/version > /dev/null 2>&1; then' >> /app/start-llm.sh && \
    echo '    echo "Ollama detected at ${LLM_BASE_URL}"' >> /app/start-llm.sh && \
    echo '    echo "Using LLM provider: ${LLM_PROVIDER} with model: ${LLM_MODEL}"' >> /app/start-llm.sh && \
    echo 'else' >> /app/start-llm.sh && \
    echo '    echo "Warning: Ollama not detected at ${LLM_BASE_URL}"' >> /app/start-llm.sh && \
    echo '    echo "Make sure to run Ollama separately or set LLM_BASE_URL to external service"' >> /app/start-llm.sh && \
    echo 'fi' >> /app/start-llm.sh && \
    echo '' >> /app/start-llm.sh && \
    echo '# Process all images with LLM enhancement' >> /app/start-llm.sh && \
    echo 'if curl -f -s ${LLM_BASE_URL}/api/version > /dev/null 2>&1; then' >> /app/start-llm.sh && \
    echo '    echo "Processing all images with LLM..."' >> /app/start-llm.sh && \
    echo '    echo "[]" > ./Data/books.go.json' >> /app/start-llm.sh && \
    echo '    # Find and process all JPG images' >> /app/start-llm.sh && \
    echo '    find ./Data/IMGS -name "*.JPG" -o -name "*.jpg" | while read img_path; do' >> /app/start-llm.sh && \
    echo '        echo "Processing image: $img_path"' >> /app/start-llm.sh && \
    echo '        # Create basic book entry' >> /app/start-llm.sh && \
    echo '        img_name=$(basename "$img_path" .JPG)' >> /app/start-llm.sh && \
    echo '        book_json=$(cat <<EOF' >> /app/start-llm.sh && \
    echo '{' >> /app/start-llm.sh && \
    echo '  "id": "book-$img_name",' >> /app/start-llm.sh && \
    echo '  "title": "Unknown Title",' >> /app/start-llm.sh && \
    echo '  "author": "Unknown Author",' >> /app/start-llm.sh && \
    echo '  "description": "Book processed from image",' >> /app/start-llm.sh && \
    echo '  "image_path": "$img_path",' >> /app/start-llm.sh && \
    echo '  "data_source": "manual",' >> /app/start-llm.sh && \
    echo '  "processed_at": "$(date -Iseconds)"' >> /app/start-llm.sh && \
    echo '}' >> /app/start-llm.sh && \
    echo 'EOF' >> /app/start-llm.sh && \
    echo ')' >> /app/start-llm.sh && \
    echo '        # Process with LLM' >> /app/start-llm.sh && \
    echo '        enhanced=$(echo "$book_json" | ./book-llm -provider ${LLM_PROVIDER} -model ${LLM_MODEL} -base-url ${LLM_BASE_URL} 2>/dev/null || echo "$book_json")' >> /app/start-llm.sh && \
    echo '        # Add to books array' >> /app/start-llm.sh && \
    echo '        jq --argjson book "$enhanced" ". += [\$book]" ./Data/books.go.json > ./Data/books-temp.json && mv ./Data/books-temp.json ./Data/books.go.json' >> /app/start-llm.sh && \
    echo '    done' >> /app/start-llm.sh && \
    echo '    echo "LLM processing complete. Processed $(jq length ./Data/books.go.json) books."' >> /app/start-llm.sh && \
    echo 'else' >> /app/start-llm.sh && \
    echo '    echo "Ollama not available. Creating basic books list from images."' >> /app/start-llm.sh && \
    echo '    echo "[]" > ./Data/books.go.json' >> /app/start-llm.sh && \
    echo '    find ./Data/IMGS -name "*.JPG" -o -name "*.jpg" | while read img_path; do' >> /app/start-llm.sh && \
    echo '        img_name=$(basename "$img_path" .JPG)' >> /app/start-llm.sh && \
    echo '        basic_book=$(cat <<EOF' >> /app/start-llm.sh && \
    echo '{' >> /app/start-llm.sh && \
    echo '  "id": "book-$img_name",' >> /app/start-llm.sh && \
    echo '  "title": "Unknown Title",' >> /app/start-llm.sh && \
    echo '  "author": "Unknown Author",' >> /app/start-llm.sh && \
    echo '  "image_path": "$img_path",' >> /app/start-llm.sh && \
    echo '  "data_source": "manual"' >> /app/start-llm.sh && \
    echo '}' >> /app/start-llm.sh && \
    echo 'EOF' >> /app/start-llm.sh && \
    echo ')' >> /app/start-llm.sh && \
    echo '        jq --argjson book "$basic_book" ". += [\$book]" ./Data/books.go.json > ./Data/books-temp.json && mv ./Data/books-temp.json ./Data/books.go.json' >> /app/start-llm.sh && \
    echo '    done' >> /app/start-llm.sh && \
    echo 'fi' >> /app/start-llm.sh && \
    echo '' >> /app/start-llm.sh && \
    echo '# Check if we have a Hugo generator binary' >> /app/start-llm.sh && \
    echo 'if [ -f "./book-processor" ]; then' >> /app/start-llm.sh && \
    echo '    echo "Regenerating Hugo site with updated data..."' >> /app/start-llm.sh && \
    echo '    ./book-processor -generate' >> /app/start-llm.sh && \
    echo 'elif [ -f "./cmd/book-hugo/main.go" ]; then' >> /app/start-llm.sh && \
    echo '    echo "Building and running Hugo generator..."' >> /app/start-llm.sh && \
    echo '    go run ./cmd/book-hugo/main.go -input ./Data/books.go.json' >> /app/start-llm.sh && \
    echo 'else' >> /app/start-llm.sh && \
    echo '    echo "No Hugo generator found, creating basic site structure..."' >> /app/start-llm.sh && \
    echo '    mkdir -p hugo-site/content hugo-site/static/data' >> /app/start-llm.sh && \
    echo '    cp ./Data/books.json hugo-site/static/data/ 2>/dev/null || true' >> /app/start-llm.sh && \
    echo 'fi' >> /app/start-llm.sh && \
    echo '' >> /app/start-llm.sh && \
    echo 'echo "Starting Hugo server on port 1313..."' >> /app/start-llm.sh && \
    echo 'cd hugo-site && hugo server --bind 0.0.0.0 --port 1313' >> /app/start-llm.sh && \
    chmod +x /app/start-llm.sh

# Expose port
EXPOSE 1313

# Run the LLM-first application
CMD ["/app/start-llm.sh"]