# Book Library System

A simple book library system built with Go that uses OCR to extract book information from cover images and generates static websites for browsing your collection.

## Features

- **OCR Processing**: Extract text from book cover images using Tesseract
- **LLM Enhancement**: Use Ollama for improved book identification and metadata
- **Hugo Site Generation**: Create static websites to browse your library
- **Multiple Deployment Options**: Docker and direct execution
- **Easy Setup**: Single binary, minimal dependencies

## Quick Start

### Using Docker (Recommended)

```bash
# Build and run with Docker Compose
docker-compose up --build

# Access the site at http://localhost:1313
```

### Direct Usage

```bash
# Build the application
go build -o book-processor ./cmd/main.go

# Process images and generate site
./book-processor -process -generate -images ./Data/IMGS/iPhone/Recents
```

## How It Works

1. **Image Discovery**: Scans directory for book cover images (JPG/PNG)
2. **OCR Processing**: Extracts text using Tesseract OCR
3. **LLM Enhancement**: Uses Ollama (if available) to improve book identification
4. **Site Generation**: Creates a Hugo static site for browsing your library

## Configuration

Edit `config.yaml`:

```yaml
image_dir: "./Data/IMGS/iPhone/Recents/"
output_file: "./Data/books.json"
hugo_dir: "./hugo-site/"

google_books_api:
  enabled: true
  api_key: ""  # Set via GOOGLE_BOOKS_API_KEY env var

openlibrary_api:
  enabled: true

tesseract:
  language: "eng"
  page_segmentation_mode: 6

ollama:
  url: "http://localhost:11434"
  model: "gemma3:27b"
```

## Project Structure

```
book-library/
├── cmd/main.go                 # Main application
├── internal/
│   ├── apis/                   # External API clients
│   ├── config/                 # Configuration
│   ├── models/                 # Data models
│   ├── ocr/                    # OCR processing
│   ├── processor/              # Main processing logic
│   └── storage/                # Data persistence
├── Data/                       # Data directory
│   ├── IMGS/                   # Book cover images
│   └── books.json             # Generated book data
├── hugo-site/                  # Generated Hugo site
├── config.yaml                 # Application configuration
├── docker-compose.yml          # Docker setup
└── README.md                   # This file
```

## API Integration

The system uses a fallback approach:

1. **OCR**: Extract title/author from covers
2. **Google Books API**: Search for detailed metadata (requires API key)
3. **OpenLibrary API**: Free fallback search
4. **LLM Processing**: Ollama-based enhancement (optional)

## Requirements

- Go 1.21+
- Tesseract OCR
- Docker (for containerized deployment)
- Ollama (optional, for LLM enhancement)

## Installation

### System Dependencies

```bash
# Ubuntu/Debian
sudo apt-get install tesseract-ocr tesseract-ocr-eng

# macOS
brew install tesseract

# Windows
# Download from https://github.com/UB-Mannheim/tesseract/wiki
```

### Optional: Ollama Setup

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull vision model
ollama pull gemma3:27b
```

## Docker Commands

```bash
# Start full stack
docker-compose up --build

# Process images only
docker-compose run --rm book-processor ./book-processor -process -generate

# View logs
docker-compose logs -f

# Access site at http://localhost:1313
```

## Environment Variables

- `GOOGLE_BOOKS_API_KEY`: Google Books API key (optional but recommended)

## Troubleshooting

1. **OCR not working**: Install tesseract-ocr and ensure it's in PATH
2. **Low OCR accuracy**: Use high-resolution images, check tesseract language data
3. **API rate limits**: Add delays between requests, use API keys
4. **Docker permissions**: Add user to docker group: `sudo usermod -aG docker $USER`

## License

MIT License