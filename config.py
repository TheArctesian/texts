import os

# Editable variable for the directory containing book images
IMAGE_DIR = "./Data/IMGS/"  # Change this to your directory path (e.g., "./books" if running locally)

# Output file for extracted book data
OUTPUT_FILE = "./Data/books.json"

# Ollama setup (edit if needed; inside container, ollama is at localhost:11434)
OLLAMA_HOST = "http://localhost:11434"  # Refers to the ollama service in Docker Compose
MODEL_NAME = "gemma3:12b"  # Use a multimodal model like llava
