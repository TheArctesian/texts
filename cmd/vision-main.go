package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"book-library/internal/processor"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ImageDir    string `yaml:"image_dir"`
	OutputFile  string `yaml:"output_file"`
	HugoDir     string `yaml:"hugo_dir"`
	
	Ollama struct {
		URL   string `yaml:"url"`
		Model string `yaml:"model"`
	} `yaml:"ollama"`
	
	GoogleBooksAPI struct {
		Enabled bool   `yaml:"enabled"`
		APIKey  string `yaml:"api_key"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"google_books_api"`
	
	OpenLibraryAPI struct {
		Enabled bool   `yaml:"enabled"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"openlibrary_api"`
}

func loadConfig(configPath string) (*Config, error) {
	config := &Config{
		ImageDir:   "./Data/IMGS/iPhone/Recents/",
		OutputFile: "./Data/books.json",
		HugoDir:    "./hugo-site/",
	}
	
	// Set defaults
	config.Ollama.URL = "http://localhost:11434"
	config.Ollama.Model = "gemma2:27b"
	config.GoogleBooksAPI.Enabled = true
	config.GoogleBooksAPI.BaseURL = "https://www.googleapis.com/books/v1"
	config.OpenLibraryAPI.Enabled = true
	config.OpenLibraryAPI.BaseURL = "https://openlibrary.org"
	
	// Load from file if exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
			log.Printf("Config file not found, using defaults")
		} else {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config: %w", err)
			}
		}
	}
	
	// Override with environment variables
	if apiKey := os.Getenv("GOOGLE_BOOKS_API_KEY"); apiKey != "" {
		config.GoogleBooksAPI.APIKey = apiKey
	}
	
	if ollamaURL := os.Getenv("OLLAMA_URL"); ollamaURL != "" {
		config.Ollama.URL = ollamaURL
	}
	
	if ollamaModel := os.Getenv("OLLAMA_MODEL"); ollamaModel != "" {
		config.Ollama.Model = ollamaModel
	}
	
	return config, nil
}

func main() {
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		imageDir   = flag.String("images", "", "Directory containing book cover images")
		outputFile = flag.String("output", "", "Output JSON file path")
		singleFile = flag.String("file", "", "Process a single image file")
		ollamaURL  = flag.String("ollama-url", "", "Ollama server URL")
		ollamaModel = flag.String("model", "", "Ollama model to use (e.g., gemma2:27b, llava:34b)")
		help       = flag.Bool("help", false, "Show help message")
	)
	
	flag.Parse()
	
	if *help {
		fmt.Println("Book Vision Processor - Extract book information using AI vision models")
		fmt.Println("\nUsage:")
		fmt.Println("  book-vision [options]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  # Process all images in default directory")
		fmt.Println("  ./book-vision")
		fmt.Println("")
		fmt.Println("  # Process a single image")
		fmt.Println("  ./book-vision -file book_cover.jpg")
		fmt.Println("")
		fmt.Println("  # Use custom Ollama model")
		fmt.Println("  ./book-vision -model llava:34b")
		fmt.Println("")
		fmt.Println("  # Process custom directory")
		fmt.Println("  ./book-vision -images ./my-books/ -output ./my-books.json")
		fmt.Println("\nNote: Requires Ollama running with a vision-capable model (gemma2:27b, llava, etc.)")
		os.Exit(0)
	}
	
	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	
	// Override config with command-line flags
	if *imageDir != "" {
		config.ImageDir = *imageDir
	}
	if *outputFile != "" {
		config.OutputFile = *outputFile
	}
	if *ollamaURL != "" {
		config.Ollama.URL = *ollamaURL
	}
	if *ollamaModel != "" {
		config.Ollama.Model = *ollamaModel
	}
	
	// Validate Ollama is accessible
	log.Printf("Using Ollama at %s with model %s", config.Ollama.URL, config.Ollama.Model)
	
	// Create processor
	visionProcessor := processor.NewVisionProcessor(config.Ollama.URL, config.Ollama.Model)
	
	// Process single file or directory
	if *singleFile != "" {
		log.Printf("Processing single file: %s", *singleFile)
		book, err := visionProcessor.ProcessImage(*singleFile)
		if err != nil {
			log.Fatalf("Failed to process image: %v", err)
		}
		
		// Print results
		fmt.Printf("\nExtracted Book Information:\n")
		fmt.Printf("Title: %s\n", book.Title)
		fmt.Printf("Author: %s\n", book.Author)
		fmt.Printf("Description: %s\n", book.Description)
		fmt.Printf("Data Source: %s\n", book.DataSource)
		fmt.Printf("Release Date: %s\n", book.ReleaseDate)
		fmt.Printf("\nFull extraction:\n%s\n", book.ExtractedText)
	} else {
		log.Printf("Processing directory: %s", config.ImageDir)
		if err := visionProcessor.ProcessDirectory(config.ImageDir, config.OutputFile); err != nil {
			log.Fatalf("Failed to process directory: %v", err)
		}
		log.Printf("Processing complete! Results saved to %s", config.OutputFile)
	}
}