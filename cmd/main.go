package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"book-library/internal/models"
	"book-library/internal/ocr"
	"book-library/internal/processor"
)

func main() {
	var (
		imageDir   = flag.String("images", "./Data/IMGS/iPhone/Recents", "Directory containing book images")
		outputFile = flag.String("output", "./Data/books.go.json", "Output JSON file")
		hugoDir    = flag.String("hugo-dir", "./hugo-site", "Hugo site directory")
		process    = flag.Bool("process", false, "Process images with OCR")
		generate   = flag.Bool("generate", false, "Generate Hugo site")
		serve      = flag.Bool("serve", false, "Serve Hugo site after generation")
		port       = flag.String("port", "1313", "Port for Hugo server")
	)
	flag.Parse()

	fmt.Println("🚀 Book Library Processor - Nixpacks Edition")
	fmt.Printf("📚 Processing images from: %s\n", *imageDir)
	fmt.Printf("💾 Output file: %s\n", *outputFile)

	var books []models.Book

	if *process {
		fmt.Println("🔍 Processing images with OCR...")
		books = processImages(*imageDir)
		
		// Save to JSON file
		if err := saveBooks(books, *outputFile); err != nil {
			log.Fatalf("Failed to save books: %v", err)
		}
		fmt.Printf("✅ Processed %d books, saved to %s\n", len(books), *outputFile)
	} else {
		// Load existing books
		var err error
		books, err = loadBooks(*outputFile)
		if err != nil {
			log.Printf("Warning: Could not load existing books: %v", err)
			books = []models.Book{}
		}
	}

	if *generate {
		fmt.Println("🏗️  Generating Hugo site...")
		if err := generateHugoSite(books, *hugoDir); err != nil {
			log.Fatalf("Failed to generate Hugo site: %v", err)
		}
		fmt.Printf("✅ Generated Hugo site at %s\n", *hugoDir)
	}

	if *serve {
		fmt.Printf("🌐 Hugo site ready at %s\n", *hugoDir)
		fmt.Printf("📝 To serve: cd %s && hugo server --port %s\n", *hugoDir, *port)
	}
}

func processImages(imageDir string) []models.Book {
	var books []models.Book
	
	// Initialize OCR
	ocrProcessor := ocr.NewTesseractOCR()
	defer ocrProcessor.Close()

	// Find all image files with supported extensions
	var imageFiles []string
	extensions := []string{"*.jpg", "*.JPG", "*.jpeg", "*.JPEG", "*.png", "*.PNG"}
	
	for _, ext := range extensions {
		files, err := filepath.Glob(filepath.Join(imageDir, ext))
		if err != nil {
			log.Printf("Error finding %s files: %v", ext, err)
			continue
		}
		imageFiles = append(imageFiles, files...)
	}

	fmt.Printf("📖 Found %d images to process\n", len(imageFiles))

	for i, imagePath := range imageFiles {
		fmt.Printf("Processing %d/%d: %s\n", i+1, len(imageFiles), filepath.Base(imagePath))
		
		// Extract text using OCR
		extractedText, err := ocrProcessor.ExtractText(imagePath)
		if err != nil {
			log.Printf("OCR failed for %s: %v", imagePath, err)
			continue
		}

		// Parse title and author from extracted text using OCR's built-in parser
		title, author := ocrProcessor.ExtractBookInfo(extractedText)
		
		if title == "" {
			log.Printf("No title found for %s, skipping", filepath.Base(imagePath))
			continue
		}

		book := models.Book{
			ID:            generateBookID(title, author),
			Title:         title,
			Author:        author,
			Description:   fmt.Sprintf("Extracted via OCR from %s", filepath.Base(imagePath)),
			ImagePath:     imagePath,
			ExtractedText: extractedText,
			ProcessedAt:   time.Now(),
			DataSource:    "OCR-Only",
		}

		books = append(books, book)
	}

	return books
}


func generateBookID(title, author string) string {
	combined := strings.ToLower(title + "-" + author)
	combined = strings.ReplaceAll(combined, " ", "-")
	combined = strings.ReplaceAll(combined, "'", "")
	combined = strings.ReplaceAll(combined, "\"", "")
	return combined
}

func saveBooks(books []models.Book, filename string) error {
	// Ensure directory exists
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(books)
}

func loadBooks(filename string) ([]models.Book, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var books []models.Book
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&books)
	return books, err
}

func generateHugoSite(books []models.Book, hugoDir string) error {
	hugoGen := processor.NewHugoGenerator(hugoDir)
	return hugoGen.Generate(books)
}