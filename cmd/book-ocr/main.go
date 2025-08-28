package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"book-library/internal/models"
	"book-library/internal/ocr"
)

func main() {
	var (
		imagePath = flag.String("image", "", "Path to image file (reads from stdin if not provided)")
		lang      = flag.String("lang", "eng", "Tesseract language")
		psm       = flag.Int("psm", 6, "Page segmentation mode")
		quiet     = flag.Bool("q", false, "Quiet mode (only output JSON)")
	)
	flag.Parse()

	var reader io.Reader
	var filename string

	if *imagePath != "" {
		file, err := os.Open(*imagePath)
		if err != nil {
			log.Fatalf("Failed to open image: %v", err)
		}
		defer file.Close()
		reader = file
		filename = filepath.Base(*imagePath)
	} else {
		reader = os.Stdin
		filename = "stdin"
	}

	imageData, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Failed to read image: %v", err)
	}

	tempFile, err := os.CreateTemp("", "ocr-*.jpg")
	if err != nil {
		log.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(imageData); err != nil {
		log.Fatalf("Failed to write temp file: %v", err)
	}
	tempFile.Close()

	extractor := ocr.NewTesseractExtractor(*lang, *psm)
	bookInfo, err := extractor.ExtractBookInfo(tempFile.Name())
	if err != nil {
		log.Fatalf("OCR extraction failed: %v", err)
	}

	book := &models.Book{
		Title:     bookInfo.Title,
		Author:    bookInfo.Author,
		ISBN:      bookInfo.ISBN,
		ImagePath: filename,
		Source:    "OCR",
	}

	output, err := json.Marshal(book)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Extracted from %s:\n", filename)
		fmt.Fprintf(os.Stderr, "  Title: %s\n", book.Title)
		fmt.Fprintf(os.Stderr, "  Author: %s\n", book.Author)
		if book.ISBN != "" {
			fmt.Fprintf(os.Stderr, "  ISBN: %s\n", book.ISBN)
		}
		fmt.Fprintln(os.Stderr, "---")
	}

	fmt.Println(string(output))
}