package processor

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"book-library/internal/apis"
	"book-library/internal/models"
	"book-library/internal/vision"
)

type VisionProcessor struct {
	ollamaClient   *vision.OllamaClient
	googleBooksAPI *apis.GoogleBooksAPI
	openLibraryAPI *apis.OpenLibraryAPI
}

func NewVisionProcessor(ollamaURL, ollamaModel string) *VisionProcessor {
	return &VisionProcessor{
		ollamaClient:   vision.NewOllamaClient(ollamaURL, ollamaModel),
		googleBooksAPI: apis.NewGoogleBooksAPI("", ""),
		openLibraryAPI: apis.NewOpenLibraryAPI(""),
	}
}

func (p *VisionProcessor) ProcessImage(imagePath string) (*models.Book, error) {
	log.Printf("Processing image with vision model: %s", imagePath)
	
	// Step 1: Analyze image with Ollama/Gemma
	bookInfo, err := p.ollamaClient.AnalyzeBookCover(imagePath)
	if err != nil {
		log.Printf("Warning: Vision analysis failed for %s: %v", imagePath, err)
		return nil, err
	}
	
	log.Printf("Vision extracted - Title: %s, Author: %s (confidence: %.2f)", 
		bookInfo.Title, bookInfo.Author, bookInfo.Confidence)
	
	// Create initial book record
	book := &models.Book{
		ID:          p.generateBookID(bookInfo.Title, bookInfo.Author),
		Title:       bookInfo.Title,
		Author:      bookInfo.Author,
		Description: bookInfo.Description,
		ImagePath:   imagePath,
		ProcessedAt: time.Now(),
		DataSource:  "Vision-Ollama",
	}
	
	// Add additional metadata from vision
	if bookInfo.PublicationYear != "" {
		if year, err := parseYear(bookInfo.PublicationYear); err == nil {
			book.ReleaseDate = &year
		}
	}
	if bookInfo.Language != "" {
		book.EditionLanguage = bookInfo.Language
	}
	if bookInfo.Publisher != "" {
		book.Publisher = bookInfo.Publisher
		if book.Description == "" {
			book.Description = fmt.Sprintf("Published by %s", bookInfo.Publisher)
		}
	}
	if bookInfo.ISBN != "" {
		book.ISBN = bookInfo.ISBN
	}
	
	// Step 2: Refine with API data if confidence is high enough
	if bookInfo.Confidence >= 0.7 && bookInfo.Title != "" {
		p.refineWithAPIs(book, bookInfo)
	}
	
	// Store vision analysis results
	book.ExtractedText = fmt.Sprintf("Vision Analysis:\nTitle: %s\nAuthor: %s\nISBN: %s\nPublisher: %s\nYear: %s\nConfidence: %.2f",
		bookInfo.Title, bookInfo.Author, bookInfo.ISBN, bookInfo.Publisher, bookInfo.PublicationYear, bookInfo.Confidence)
	
	return book, nil
}

func (p *VisionProcessor) refineWithAPIs(book *models.Book, visionInfo *vision.BookInfo) {
	// Try Google Books first if we have title and author
	if visionInfo.Title != "" {
		log.Printf("Searching Google Books for: %s by %s", visionInfo.Title, visionInfo.Author)
		googleBook, err := p.googleBooksAPI.SearchByTitleAndAuthor(visionInfo.Title, visionInfo.Author)
		if err == nil && googleBook != nil {
			p.mergeBookData(book, googleBook)
			book.DataSource = "Vision+GoogleBooks"
			return
		}
		
		// Try OpenLibrary as fallback
		log.Printf("Searching OpenLibrary for: %s by %s", visionInfo.Title, visionInfo.Author)
		openLibBook, err := p.openLibraryAPI.SearchByTitleAndAuthor(visionInfo.Title, visionInfo.Author)
		if err == nil && openLibBook != nil {
			p.mergeBookData(book, openLibBook)
			book.DataSource = "Vision+OpenLibrary"
			return
		}
	}
	
	// If we have ISBN, try direct lookup
	if visionInfo.ISBN != "" {
		log.Printf("Looking up by ISBN: %s", visionInfo.ISBN)
		
		// Try Google Books ISBN search
		googleBook, err := p.googleBooksAPI.SearchByISBN(visionInfo.ISBN)
		if err == nil && googleBook != nil {
			p.mergeBookData(book, googleBook)
			book.DataSource = "Vision+GoogleBooks-ISBN"
			return
		}
		
		// Try OpenLibrary ISBN search
		openLibBook, err := p.openLibraryAPI.SearchByISBN(visionInfo.ISBN)
		if err == nil && openLibBook != nil {
			p.mergeBookData(book, openLibBook)
			book.DataSource = "Vision+OpenLibrary-ISBN"
		}
	}
}

func parseYear(yearStr string) (int, error) {
	// Try to extract year from various formats
	yearStr = strings.TrimSpace(yearStr)
	
	// Handle simple year format
	if year, err := strconv.Atoi(yearStr); err == nil {
		return year, nil
	}
	
	// Try to extract 4-digit year from string
	for _, part := range strings.Fields(yearStr) {
		if len(part) == 4 {
			if year, err := strconv.Atoi(part); err == nil && year > 1000 && year < 3000 {
				return year, nil
			}
		}
	}
	
	return 0, fmt.Errorf("could not parse year from: %s", yearStr)
}

func (p *VisionProcessor) mergeBookData(target, source *models.Book) {
	// Keep vision-extracted data but enrich with API data
	if target.Title == "" && source.Title != "" {
		target.Title = source.Title
	}
	if target.Author == "" && source.Author != "" {
		target.Author = source.Author
	}
	if target.Description == "" && source.Description != "" {
		target.Description = source.Description
	}
	if source.OriginalDate != nil {
		target.OriginalDate = source.OriginalDate
	}
	if target.ReleaseDate == nil && source.ReleaseDate != nil {
		target.ReleaseDate = source.ReleaseDate
	}
	if source.OriginalLocationName != "" {
		target.OriginalLocationName = source.OriginalLocationName
		target.OriginalLocationLat = source.OriginalLocationLat
		target.OriginalLocationLng = source.OriginalLocationLng
	}
	if target.OriginalLanguage == "" && source.OriginalLanguage != "" {
		target.OriginalLanguage = source.OriginalLanguage
	}
	if target.EditionLanguage == "" && source.EditionLanguage != "" {
		target.EditionLanguage = source.EditionLanguage
	}
}

func (p *VisionProcessor) generateBookID(title, author string) string {
	id := strings.ToLower(title + "-" + author)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, ".", "")
	id = strings.ReplaceAll(id, ",", "")
	id = strings.ReplaceAll(id, "'", "")
	id = strings.ReplaceAll(id, "\"", "")
	return id
}

func (p *VisionProcessor) ProcessDirectory(imageDir, outputFile string) error {
	// Find all image files
	imageFiles, err := filepath.Glob(filepath.Join(imageDir, "*.JPG"))
	if err != nil {
		return fmt.Errorf("failed to find images: %w", err)
	}
	
	pngFiles, _ := filepath.Glob(filepath.Join(imageDir, "*.PNG"))
	imageFiles = append(imageFiles, pngFiles...)
	
	jpgFiles, _ := filepath.Glob(filepath.Join(imageDir, "*.jpg"))
	imageFiles = append(imageFiles, jpgFiles...)
	
	pngLowerFiles, _ := filepath.Glob(filepath.Join(imageDir, "*.png"))
	imageFiles = append(imageFiles, pngLowerFiles...)
	
	log.Printf("Found %d image files to process", len(imageFiles))
	
	var books []*models.Book
	successCount := 0
	failCount := 0
	
	for i, imagePath := range imageFiles {
		log.Printf("Processing image %d/%d: %s", i+1, len(imageFiles), filepath.Base(imagePath))
		
		book, err := p.ProcessImage(imagePath)
		if err != nil {
			log.Printf("Failed to process %s: %v", imagePath, err)
			failCount++
			continue
		}
		
		if book != nil && book.Title != "" {
			books = append(books, book)
			successCount++
			log.Printf("Successfully processed: %s by %s", book.Title, book.Author)
		} else {
			failCount++
			log.Printf("No book data extracted from %s", imagePath)
		}
		
		// Add delay to avoid overwhelming Ollama
		time.Sleep(2 * time.Second)
	}
	
	log.Printf("Processing complete: %d successful, %d failed", successCount, failCount)
	
	// Save results
	if len(books) > 0 {
		return p.saveBooks(books, outputFile)
	}
	
	return nil
}

func (p *VisionProcessor) saveBooks(books []*models.Book, outputFile string) error {
	// Create output directory if needed
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Marshal books to JSON
	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal books: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	
	log.Printf("Saved %d books to %s", len(books), outputFile)
	return nil
}