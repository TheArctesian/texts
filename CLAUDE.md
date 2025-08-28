# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a book library system built with Go that processes book cover images using OCR and LLM enhancement to extract book information, then generates static websites for browsing your library collection.

Key principles:
- **Simplicity First**: Multiple focused binaries, minimal dependencies
- **LLM Enhancement**: Ollama-based processing for improved book identification
- **Multiple Processing Modes**: OCR, LLM, Vision, and API-based approaches
- **Easy Deployment**: Docker-based deployment with host networking
- **Modular Architecture**: Separate tools for different processing stages

## Core Architecture

The system consists of multiple Go binaries for different processing approaches:

1. **OCR Processing**: Extract text from book cover images using Tesseract
2. **LLM Enhancement**: Use Ollama for improved book identification and metadata
3. **Vision Processing**: Direct image analysis using vision models
4. **API Integration**: Google Books and OpenLibrary API fallbacks
5. **Hugo Site Generation**: Create static websites to browse the library
6. **Pipeline Orchestration**: Coordinate multiple processing stages

### Key Components

- **`cmd/main.go`**: Main OCR-based processor (legacy)
- **`cmd/book-llm/main.go`**: LLM-enhanced book processor
- **`cmd/book-vision/main.go`**: Vision model processor
- **`cmd/book-api/main.go`**: API-based metadata enrichment
- **`cmd/book-hugo/main.go`**: Hugo static site generator
- **`cmd/book-pipeline/main.go`**: Processing pipeline orchestrator
- **`internal/ocr/simple_tesseract.go`**: Simple OCR wrapper
- **`internal/vision/ollama.go`**: Ollama vision integration
- **`internal/processor/vision_processor.go`**: Vision-based processing logic

## Development Commands

### Using Docker (Recommended)

```bash
# Build and run with Docker Compose
docker-compose up --build

# Access site at http://localhost:1313
```

### Direct Usage

```bash
# Build the main OCR processor
go build -o book-processor ./cmd/main.go

# Process images with OCR
./book-processor -process -generate -images ./Data/IMGS/iPhone/Recents

# Build and run LLM processor
go build -o book-llm ./cmd/book-llm/main.go
./book-llm -process -generate

# Build and run vision processor
go build -o book-vision ./cmd/book-vision/main.go
./book-vision -process -generate
```

### Multiple Processing Approaches

```bash
# OCR-only processing (fastest, least accurate)
go run ./cmd/main.go -process -generate

# LLM-enhanced processing (balanced speed/accuracy)
go run ./cmd/book-llm/main.go -process -generate

# Vision model processing (slowest, most accurate)
go run ./cmd/book-vision/main.go -process -generate
```

## How It Works

The system offers multiple processing approaches for different needs:

### 1. OCR Processing (cmd/main.go)

Traditional OCR-based approach:
1. **Image Discovery**: Scans directory for JPG/PNG files
2. **Text Extraction**: Uses Tesseract OCR to extract text from covers
3. **Simple Parsing**: Basic heuristics to identify title and author
4. **Fast Processing**: ~1-3 seconds per image

### 2. LLM Enhancement (cmd/book-llm/main.go)

Uses Ollama for improved accuracy:
1. **Image Analysis**: Send images to Ollama vision models
2. **Structured Output**: Extract title, author, genre, year, etc.
3. **High Accuracy**: Better identification than OCR alone
4. **Medium Speed**: ~5-10 seconds per image depending on model

### 3. Vision Processing (cmd/book-vision/main.go)

Direct vision model analysis:
1. **Image Understanding**: Advanced computer vision for book covers
2. **Context Awareness**: Understanding of cover design and typography
3. **Highest Accuracy**: Best results for complex covers
4. **Slower Processing**: ~10-30 seconds per image

### Output Format

All processors generate consistent JSON:
```json
{
  "id": "book-title-author",
  "title": "Book Title",
  "author": "Author Name",
  "description": "Book description from LLM/API",
  "original_date": 1984,
  "publisher": "Publisher Name",
  "genre": "Fiction",
  "original_language": "English",
  "image_path": "/path/to/image.jpg",
  "processed_at": "2025-08-27T18:42:08Z",
  "data_source": "LLM-Enhanced",
  "confidence_score": 0.95,
  "llm_analysis": {
    "corrections_made": ["Fixed title", "Added genre"],
    "confidence": 0.95
  }
}
```

## Docker Configuration

The system uses Docker with host networking for Ollama integration:

```yaml
services:
  book-library:
    build: .
    container_name: book-library-llm
    ports:
      - "1313:1313"
    volumes:
      - ./Data:/app/Data
      - ./hugo-site:/app/hugo-site
    environment:
      - LLM_PROVIDER=ollama
      - LLM_MODEL=gemma3:27b
      - LLM_BASE_URL=http://localhost:11434
    network_mode: "host"
```

## Dependencies

### System Requirements
- Go 1.21+
- Docker & Docker Compose
- Ollama (for LLM processing)
- Hugo (installed automatically in Docker)

### Go Dependencies
The project uses minimal dependencies:
```bash
go mod tidy
# Main dependency: gopkg.in/yaml.v3 for configuration
```

### Ollama Setup
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull vision model
ollama pull gemma3:27b

# Start Ollama server
ollama serve
```

## Configuration

The system uses `config.yaml` for all processors:

```yaml
image_dir: "./Data/IMGS/iPhone/Recents/"
output_file: "./Data/books.json"
hugo_dir: "./hugo-site/"

google_books_api:
  enabled: true
  api_key: ""  # Set via GOOGLE_BOOKS_API_KEY env var
  base_url: "https://www.googleapis.com/books/v1"

openlibrary_api:
  enabled: true
  base_url: "https://openlibrary.org"

tesseract:
  language: "eng"
  page_segmentation_mode: 6

ollama:
  url: "http://localhost:11434"
  model: "gemma3:27b"
```

## Data Processing Flow

1. **Image Discovery**: Scan image directory for book covers
2. **Processing Selection**: Choose OCR, LLM, or Vision approach
3. **Book Identification**: Extract title, author, and metadata
4. **API Enrichment**: Search external APIs for additional data
5. **LLM Enhancement**: Improve and validate extracted information
6. **Storage**: Save enriched book data to JSON file
7. **Hugo Generation**: Create static site with visualizations

## Hugo Site Structure

The generated Hugo site includes:

```
hugo-site/
├── content/
│   ├── _index.md           # Home page
│   ├── books.md            # Books listing
│   ├── timeline.md         # Timeline visualization
│   └── geography.md        # Geographic visualization
├── layouts/
│   └── _default/
│       ├── baseof.html     # Base template
│       ├── books.html      # Books page layout
│       ├── timeline.html   # Timeline layout
│       └── geography.html  # Geography layout
├── static/
│   ├── css/style.css       # Styling
│   ├── js/                 # JavaScript for visualizations
│   └── data/               # Generated JSON data files
└── hugo.toml              # Hugo configuration
```

## Available Tools

### Processing Tools
- **book-processor**: Main OCR processor (legacy)
- **book-llm**: LLM-enhanced processor
- **book-vision**: Vision model processor
- **book-api**: API metadata enrichment
- **book-pipeline**: Processing orchestrator

### Utility Tools
- **book-hugo**: Hugo site generator
- **book-search**: Search existing books
- **book-verify**: Validate book data
- **book-ocr**: Standalone OCR tool

## Docker Commands

```bash
# Build and start the application
docker-compose up --build

# View logs
docker-compose logs -f book-library

# Run specific processors
docker-compose exec book-library ./book-llm -process -generate
docker-compose exec book-library ./book-vision -process -generate

# Access the generated site at http://localhost:1313
```

## Processing Your Full Dataset

To process all 320+ images in your collection:

```bash
# Using Docker (recommended for full dataset)
docker-compose up --build

# Using direct Go execution
go build -o book-processor ./cmd/main.go
./book-processor -process -generate -images ./Data/IMGS/iPhone/Recents

# Using LLM processor for better accuracy
go build -o book-llm ./cmd/book-llm/main.go
./book-llm -process -generate
```

## Environment Variables

- `LLM_PROVIDER`: LLM provider (default: ollama)
- `LLM_MODEL`: Model name (default: gemma3:27b)  
- `LLM_BASE_URL`: Ollama server URL (default: http://localhost:11434)
- `GOOGLE_BOOKS_API_KEY`: Google Books API key (optional)

## Important Notes

- **Main Processors**: Choose between OCR (`cmd/main.go`), LLM (`cmd/book-llm/main.go`), or Vision (`cmd/book-vision/main.go`)
- **Data Location**: Images in `./Data/IMGS/iPhone/Recents/` (320+ images)
- **Output Files**: Results in `./Data/books.json` and `./Data/books.go.json`
- **Hugo Site**: Generated in `./hugo-site/` directory
- **Docker Integration**: Uses host networking for Ollama access
- **Processing Time**: OCR (fast), LLM (medium), Vision (slow) but increasingly accurate