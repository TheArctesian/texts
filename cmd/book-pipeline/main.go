package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"book-library/internal/models"
)

func main() {
	var (
		inputDir  = flag.String("dir", "", "Directory of images to process")
		inputList = flag.String("list", "", "File containing list of image paths")
		output    = flag.String("output", "", "Output JSON file (stdout if not specified)")
		quiet     = flag.Bool("q", false, "Quiet mode")
	)
	flag.Parse()

	var imagePaths []string

	if *inputDir != "" {
		entries, err := os.ReadDir(*inputDir)
		if err != nil {
			log.Fatalf("Failed to read directory: %v", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" {
					imagePaths = append(imagePaths, filepath.Join(*inputDir, entry.Name()))
				}
			}
		}
	} else if *inputList != "" {
		file, err := os.Open(*inputList)
		if err != nil {
			log.Fatalf("Failed to open list file: %v", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			imagePaths = append(imagePaths, scanner.Text())
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				imagePaths = append(imagePaths, line)
			}
		}
	}

	var books []models.Book

	for _, imagePath := range imagePaths {
		if !*quiet {
			fmt.Fprintf(os.Stderr, "Processing: %s\n", imagePath)
		}

		var book models.Book
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&book); err == io.EOF {
			break
		} else if err != nil {
			log.Printf("Failed to decode book for %s: %v", imagePath, err)
			continue
		}

		book.ImagePath = imagePath
		books = append(books, book)
	}

	var writer io.Writer = os.Stdout
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(books); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}

	if !*quiet && *output != "" {
		fmt.Fprintf(os.Stderr, "Processed %d books, output written to %s\n", len(books), *output)
	}
}