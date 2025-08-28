package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"book-library/internal/models"
)

type VerificationSource struct {
	Name       string
	Confidence float64
	Data       map[string]interface{}
}

type VerifiedBook struct {
	models.Book
	VerificationSources []VerificationSource `json:"verification_sources"`
	ConfidenceScore     float64              `json:"confidence_score"`
	Corrections         map[string]string    `json:"corrections,omitempty"`
}

func main() {
	var (
		inputFile   = flag.String("input", "", "JSON input file (reads from stdin if not provided)")
		outputFile  = flag.String("output", "", "Output file (stdout if not provided)")
		useWikidata = flag.Bool("wikidata", true, "Use Wikidata for verification")
		useWorldCat = flag.Bool("worldcat", true, "Use WorldCat for verification")
		useLOC      = flag.Bool("loc", true, "Use Library of Congress for verification")
		useISBNDB   = flag.Bool("isbndb", false, "Use ISBN DB (requires API key)")
		isbndbKey   = flag.String("isbndb-key", os.Getenv("ISBNDB_API_KEY"), "ISBN DB API key")
		quiet       = flag.Bool("q", false, "Quiet mode")
	)
	flag.Parse()

	var reader io.Reader
	if *inputFile != "" {
		file, err := os.Open(*inputFile)
		if err != nil {
			log.Fatalf("Failed to open input file: %v", err)
		}
		defer file.Close()
		reader = file
	} else {
		reader = os.Stdin
	}

	var book models.Book
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&book); err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}

	verified := verifyBook(book, *useWikidata, *useWorldCat, *useLOC, *useISBNDB, *isbndbKey)

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Verification complete:\n")
		fmt.Fprintf(os.Stderr, "  Title: %s\n", verified.Title)
		fmt.Fprintf(os.Stderr, "  Confidence: %.2f%%\n", verified.ConfidenceScore*100)
		fmt.Fprintf(os.Stderr, "  Sources: %d\n", len(verified.VerificationSources))
		if len(verified.Corrections) > 0 {
			fmt.Fprintf(os.Stderr, "  Corrections made:\n")
			for field, correction := range verified.Corrections {
				fmt.Fprintf(os.Stderr, "    %s: %s\n", field, correction)
			}
		}
	}

	var writer io.Writer = os.Stdout
	if *outputFile != "" {
		file, err := os.Create(*outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		writer = file
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(verified); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}

func verifyBook(book models.Book, useWikidata, useWorldCat, useLOC, useISBNDB bool, isbndbKey string) VerifiedBook {
	verified := VerifiedBook{
		Book:                book,
		VerificationSources: []VerificationSource{},
		Corrections:         make(map[string]string),
	}

	if useWikidata && (book.Title != "" || book.ISBN != "") {
		if source := verifyWithWikidata(book); source != nil {
			verified.VerificationSources = append(verified.VerificationSources, *source)
			applyCorrections(&verified, source)
		}
	}

	if useWorldCat && book.ISBN != "" {
		if source := verifyWithWorldCat(book); source != nil {
			verified.VerificationSources = append(verified.VerificationSources, *source)
			applyCorrections(&verified, source)
		}
	}

	if useLOC && (book.Title != "" || book.ISBN != "") {
		if source := verifyWithLOC(book); source != nil {
			verified.VerificationSources = append(verified.VerificationSources, *source)
			applyCorrections(&verified, source)
		}
	}

	if useISBNDB && isbndbKey != "" && book.ISBN != "" {
		if source := verifyWithISBNDB(book, isbndbKey); source != nil {
			verified.VerificationSources = append(verified.VerificationSources, *source)
			applyCorrections(&verified, source)
		}
	}

	verified.ConfidenceScore = calculateConfidence(verified)
	return verified
}

func verifyWithWikidata(book models.Book) *VerificationSource {
	query := fmt.Sprintf(`
		SELECT ?item ?itemLabel ?authorLabel ?publicationDate WHERE {
			?item wdt:P31 wd:Q7725634.
			?item rdfs:label "%s"@en.
			OPTIONAL { ?item wdt:P50 ?author. }
			OPTIONAL { ?item wdt:P577 ?publicationDate. }
			SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
		} LIMIT 1
	`, book.Title)

	endpoint := "https://query.wikidata.org/sparql"
	resp, err := http.Get(fmt.Sprintf("%s?query=%s&format=json", endpoint, url.QueryEscape(query)))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	return &VerificationSource{
		Name:       "Wikidata",
		Confidence: 0.9,
		Data:       result,
	}
}

func verifyWithWorldCat(book models.Book) *VerificationSource {
	// WorldCat Search API (public, no key required for basic search)
	searchURL := fmt.Sprintf("http://www.worldcat.org/webservices/catalog/search/opensearch?q=%s&format=json",
		url.QueryEscape(book.ISBN))

	resp, err := http.Get(searchURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	if len(result) > 1 {
		return &VerificationSource{
			Name:       "WorldCat",
			Confidence: 0.85,
			Data: map[string]interface{}{
				"results": result[1],
			},
		}
	}
	return nil
}

func verifyWithLOC(book models.Book) *VerificationSource {
	// Library of Congress SRU API
	var query string
	if book.ISBN != "" {
		query = fmt.Sprintf("bath.isbn=%s", book.ISBN)
	} else {
		query = fmt.Sprintf("bath.title=\"%s\" AND bath.author=\"%s\"", book.Title, book.Author)
	}

	searchURL := fmt.Sprintf("https://lccn.loc.gov/sru?query=%s&operation=searchRetrieve&version=1.1&maximumRecords=1&recordSchema=mods",
		url.QueryEscape(query))

	resp, err := http.Get(searchURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	return &VerificationSource{
		Name:       "Library of Congress",
		Confidence: 0.95,
		Data: map[string]interface{}{
			"response": string(body),
		},
	}
}

func verifyWithISBNDB(book models.Book, apiKey string) *VerificationSource {
	client := &http.Client{Timeout: 10 * time.Second}
	
	req, err := http.NewRequest("GET", 
		fmt.Sprintf("https://api.isbndb.com/book/%s", book.ISBN), nil)
	if err != nil {
		return nil
	}
	
	req.Header.Set("Authorization", apiKey)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	return &VerificationSource{
		Name:       "ISBN DB",
		Confidence: 0.9,
		Data:       result,
	}
}

func applyCorrections(verified *VerifiedBook, source *VerificationSource) {
	// Extract and apply corrections from verification source
	// This is a simplified version - you'd want more sophisticated parsing
	
	if data, ok := source.Data["book"].(map[string]interface{}); ok {
		if title, ok := data["title"].(string); ok && title != verified.Title {
			verified.Corrections["title"] = fmt.Sprintf("%s -> %s", verified.Title, title)
			verified.Title = title
		}
		
		if authors, ok := data["authors"].([]interface{}); ok && len(authors) > 0 {
			if author, ok := authors[0].(string); ok && author != verified.Author {
				verified.Corrections["author"] = fmt.Sprintf("%s -> %s", verified.Author, author)
				verified.Author = author
			}
		}
		
		if yearStr, ok := data["date_published"].(string); ok {
			// Parse year from date string
			if len(yearStr) >= 4 {
				yearStr = yearStr[:4]
				var year int
				fmt.Sscanf(yearStr, "%d", &year)
				if year > 0 && year != verified.OriginalDate {
					verified.Corrections["year"] = fmt.Sprintf("%d -> %d", verified.OriginalDate, year)
					verified.OriginalDate = year
				}
			}
		}
	}
}

func calculateConfidence(verified VerifiedBook) float64 {
	if len(verified.VerificationSources) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	for _, source := range verified.VerificationSources {
		totalConfidence += source.Confidence
	}
	
	// Average confidence weighted by number of sources
	baseConfidence := totalConfidence / float64(len(verified.VerificationSources))
	
	// Bonus for multiple agreeing sources
	sourceBonus := float64(len(verified.VerificationSources)) * 0.05
	if sourceBonus > 0.2 {
		sourceBonus = 0.2
	}
	
	confidence := baseConfidence + sourceBonus
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}