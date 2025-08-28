package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"book-library/internal/apis"
	"book-library/internal/models"
)

func main() {
	var (
		title       = flag.String("title", "", "Book title to search")
		author      = flag.String("author", "", "Book author to search")
		isbn        = flag.String("isbn", "", "ISBN to search")
		useGoogle   = flag.Bool("google", true, "Use Google Books API")
		useOpenLib  = flag.Bool("openlib", true, "Use OpenLibrary API")
		googleKey   = flag.String("api-key", os.Getenv("GOOGLE_BOOKS_API_KEY"), "Google Books API key")
		quiet       = flag.Bool("q", false, "Quiet mode (only output JSON)")
	)
	flag.Parse()

	if *title == "" && *author == "" && *isbn == "" {
		var book models.Book
		decoder := json.NewDecoder(os.Stdin)
		if err := decoder.Decode(&book); err != nil {
			log.Fatalf("Failed to read JSON from stdin: %v", err)
		}
		*title = book.Title
		*author = book.Author
		*isbn = book.ISBN
	}

	if *title == "" && *author == "" && *isbn == "" {
		log.Fatal("No search criteria provided. Use -title, -author, -isbn, or pipe JSON input")
	}

	var result *models.Book

	if *useGoogle && *googleKey != "" {
		client := apis.NewGoogleBooksAPI(*googleKey, "https://www.googleapis.com/books/v1")
		if *isbn != "" {
			result = client.SearchByISBN(*isbn)
		}
		if result == nil && (*title != "" || *author != "") {
			result = client.SearchByTitleAuthor(*title, *author)
		}
	}

	if result == nil && *useOpenLib {
		client := apis.NewOpenLibraryAPI("https://openlibrary.org")
		if *isbn != "" {
			result = client.SearchByISBN(*isbn)
		}
		if result == nil && (*title != "" || *author != "") {
			result = client.SearchByTitleAuthor(*title, *author)
		}
	}

	if result == nil {
		result = &models.Book{
			Title:  *title,
			Author: *author,
			ISBN:   *isbn,
			Source: "Manual",
		}
	}

	output, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Search results:\n")
		fmt.Fprintf(os.Stderr, "  Title: %s\n", result.Title)
		fmt.Fprintf(os.Stderr, "  Author: %s\n", result.Author)
		fmt.Fprintf(os.Stderr, "  Year: %d\n", result.Year)
		fmt.Fprintf(os.Stderr, "  Source: %s\n", result.Source)
		fmt.Fprintln(os.Stderr, "---")
	}

	fmt.Println(string(output))
}