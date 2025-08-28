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
	"strings"
	"time"

	"book-library/internal/models"
)

type SearchResult struct {
	Source      string                 `json:"source"`
	Title       string                 `json:"title"`
	Author      string                 `json:"author"`
	Year        int                    `json:"year"`
	Publisher   string                 `json:"publisher"`
	ISBN        string                 `json:"isbn"`
	Description string                 `json:"description"`
	URL         string                 `json:"url"`
	Confidence  float64                `json:"confidence"`
	RawData     map[string]interface{} `json:"raw_data,omitempty"`
}

type SearchEnhancedBook struct {
	models.Book
	SearchResults   []SearchResult `json:"search_results"`
	BestMatch       *SearchResult  `json:"best_match"`
	ConfidenceScore float64        `json:"confidence_score"`
}

func main() {
	var (
		inputFile    = flag.String("input", "", "JSON input file (reads from stdin if not provided)")
		outputFile   = flag.String("output", "", "Output file (stdout if not provided)")
		useDuckDuckGo = flag.Bool("ddg", true, "Use DuckDuckGo search")
		useBing      = flag.Bool("bing", false, "Use Bing search (requires API key)")
		useSerp      = flag.Bool("serp", false, "Use SerpAPI (requires API key)")
		bingKey      = flag.String("bing-key", os.Getenv("BING_API_KEY"), "Bing API key")
		serpKey      = flag.String("serp-key", os.Getenv("SERP_API_KEY"), "SerpAPI key")
		maxResults   = flag.Int("max", 5, "Maximum results per search engine")
		quiet        = flag.Bool("q", false, "Quiet mode")
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

	enhanced := searchAndEnhance(book, *useDuckDuckGo, *useBing, *useSerp, *bingKey, *serpKey, *maxResults)

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Web Search complete:\n")
		fmt.Fprintf(os.Stderr, "  Found %d results\n", len(enhanced.SearchResults))
		if enhanced.BestMatch != nil {
			fmt.Fprintf(os.Stderr, "  Best match: %s by %s (%d)\n", 
				enhanced.BestMatch.Title, enhanced.BestMatch.Author, enhanced.BestMatch.Year)
			fmt.Fprintf(os.Stderr, "  Confidence: %.2f%%\n", enhanced.ConfidenceScore*100)
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
	if err := encoder.Encode(enhanced); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}

func searchAndEnhance(book models.Book, useDDG, useBing, useSerp bool, bingKey, serpKey string, maxResults int) SearchEnhancedBook {
	enhanced := SearchEnhancedBook{
		Book:          book,
		SearchResults: []SearchResult{},
	}

	// Build search query
	query := buildSearchQuery(book)

	if useDDG {
		results := searchDuckDuckGo(query, maxResults)
		enhanced.SearchResults = append(enhanced.SearchResults, results...)
	}

	if useBing && bingKey != "" {
		results := searchBing(query, bingKey, maxResults)
		enhanced.SearchResults = append(enhanced.SearchResults, results...)
	}

	if useSerp && serpKey != "" {
		results := searchSerpAPI(query, serpKey, maxResults)
		enhanced.SearchResults = append(enhanced.SearchResults, results...)
	}

	// Find best match and apply corrections
	enhanced.BestMatch = findBestMatch(enhanced.SearchResults, book)
	if enhanced.BestMatch != nil {
		applySearchCorrections(&enhanced)
		enhanced.ConfidenceScore = enhanced.BestMatch.Confidence
	}

	return enhanced
}

func buildSearchQuery(book models.Book) string {
	parts := []string{}
	
	if book.Title != "" {
		parts = append(parts, fmt.Sprintf("\"%s\"", book.Title))
	}
	if book.Author != "" {
		parts = append(parts, book.Author)
	}
	if book.ISBN != "" {
		parts = append(parts, fmt.Sprintf("ISBN %s", book.ISBN))
	}
	
	// Add "book" to ensure we get book results
	parts = append(parts, "book")
	
	return strings.Join(parts, " ")
}

func searchDuckDuckGo(query string, maxResults int) []SearchResult {
	results := []SearchResult{}
	
	// DuckDuckGo Instant Answer API (limited but no key required)
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1",
		url.QueryEscape(query))

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("DuckDuckGo search failed: %v", err)
		return results
	}
	defer resp.Body.Close()

	var ddgResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		log.Printf("Failed to parse DuckDuckGo response: %v", err)
		return results
	}

	// Parse AbstractText for book information
	if abstract, ok := ddgResp["AbstractText"].(string); ok && abstract != "" {
		result := SearchResult{
			Source:      "DuckDuckGo",
			Description: abstract,
			URL:         ddgResp["AbstractURL"].(string),
			Confidence:  0.6,
			RawData:     ddgResp,
		}
		
		// Try to extract title and author from abstract
		result.Title = extractTitleFromText(abstract)
		result.Author = extractAuthorFromText(abstract)
		
		results = append(results, result)
	}

	// Parse RelatedTopics for additional results
	if topics, ok := ddgResp["RelatedTopics"].([]interface{}); ok {
		for i, topic := range topics {
			if i >= maxResults {
				break
			}
			
			if topicMap, ok := topic.(map[string]interface{}); ok {
				if text, ok := topicMap["Text"].(string); ok {
					result := SearchResult{
						Source:     "DuckDuckGo",
						Description: text,
						Confidence: 0.5,
					}
					
					if firstURL, ok := topicMap["FirstURL"].(string); ok {
						result.URL = firstURL
					}
					
					results = append(results, result)
				}
			}
		}
	}

	return results
}

func searchBing(query, apiKey string, maxResults int) []SearchResult {
	results := []SearchResult{}
	
	client := &http.Client{Timeout: 10 * time.Second}
	
	apiURL := fmt.Sprintf("https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d",
		url.QueryEscape(query), maxResults)
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Failed to create Bing request: %v", err)
		return results
	}
	
	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Bing search failed: %v", err)
		return results
	}
	defer resp.Body.Close()

	var bingResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&bingResp); err != nil {
		log.Printf("Failed to parse Bing response: %v", err)
		return results
	}

	// Parse web pages results
	if webPages, ok := bingResp["webPages"].(map[string]interface{}); ok {
		if values, ok := webPages["value"].([]interface{}); ok {
			for _, value := range values {
				if page, ok := value.(map[string]interface{}); ok {
					result := SearchResult{
						Source:     "Bing",
						Title:      getString(page, "name"),
						Description: getString(page, "snippet"),
						URL:        getString(page, "url"),
						Confidence: 0.7,
						RawData:    page,
					}
					
					// Extract book metadata from snippet
					result.Author = extractAuthorFromText(result.Description)
					result.Year = extractYearFromText(result.Description)
					
					results = append(results, result)
				}
			}
		}
	}

	return results
}

func searchSerpAPI(query, apiKey string, maxResults int) []SearchResult {
	results := []SearchResult{}
	
	apiURL := fmt.Sprintf("https://serpapi.com/search.json?q=%s&api_key=%s&num=%d",
		url.QueryEscape(query), apiKey, maxResults)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("SerpAPI search failed: %v", err)
		return results
	}
	defer resp.Body.Close()

	var serpResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&serpResp); err != nil {
		log.Printf("Failed to parse SerpAPI response: %v", err)
		return results
	}

	// Parse organic results
	if organicResults, ok := serpResp["organic_results"].([]interface{}); ok {
		for _, result := range organicResults {
			if res, ok := result.(map[string]interface{}); ok {
				searchResult := SearchResult{
					Source:     "Google (via SerpAPI)",
					Title:      getString(res, "title"),
					Description: getString(res, "snippet"),
					URL:        getString(res, "link"),
					Confidence: 0.8,
					RawData:    res,
				}
				
				// Extract metadata
				searchResult.Author = extractAuthorFromText(searchResult.Description)
				searchResult.Year = extractYearFromText(searchResult.Description)
				
				results = append(results, searchResult)
			}
		}
	}

	// Also check knowledge graph if available
	if knowledgeGraph, ok := serpResp["knowledge_graph"].(map[string]interface{}); ok {
		result := SearchResult{
			Source:     "Google Knowledge Graph",
			Title:      getString(knowledgeGraph, "title"),
			Description: getString(knowledgeGraph, "description"),
			Confidence: 0.9,
			RawData:    knowledgeGraph,
		}
		
		if author, ok := knowledgeGraph["author"].(string); ok {
			result.Author = author
		}
		
		results = append(results, result)
	}

	return results
}

func findBestMatch(results []SearchResult, book models.Book) *SearchResult {
	if len(results) == 0 {
		return nil
	}

	var bestMatch *SearchResult
	bestScore := 0.0

	for i := range results {
		result := &results[i]
		score := calculateMatchScore(result, book)
		
		if score > bestScore {
			bestScore = score
			bestMatch = result
			bestMatch.Confidence = score
		}
	}

	return bestMatch
}

func calculateMatchScore(result *SearchResult, book models.Book) float64 {
	score := result.Confidence

	// Boost score for title match
	if strings.Contains(strings.ToLower(result.Title), strings.ToLower(book.Title)) ||
	   strings.Contains(strings.ToLower(result.Description), strings.ToLower(book.Title)) {
		score += 0.2
	}

	// Boost score for author match
	if book.Author != "" && (strings.Contains(strings.ToLower(result.Author), strings.ToLower(book.Author)) ||
	   strings.Contains(strings.ToLower(result.Description), strings.ToLower(book.Author))) {
		score += 0.2
	}

	// Boost score for ISBN match
	if book.ISBN != "" && strings.Contains(result.Description, book.ISBN) {
		score += 0.3
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func applySearchCorrections(enhanced *SearchEnhancedBook) {
	if enhanced.BestMatch == nil {
		return
	}

	// Apply corrections from best match
	if enhanced.BestMatch.Title != "" && enhanced.BestMatch.Title != enhanced.Title {
		enhanced.Title = enhanced.BestMatch.Title
	}
	
	if enhanced.BestMatch.Author != "" && enhanced.BestMatch.Author != enhanced.Author {
		enhanced.Author = enhanced.BestMatch.Author
	}
	
	if enhanced.BestMatch.Year > 0 && enhanced.OriginalDate == 0 {
		enhanced.OriginalDate = enhanced.BestMatch.Year
	}
	
	if enhanced.BestMatch.Publisher != "" && enhanced.Publisher == "" {
		enhanced.Publisher = enhanced.BestMatch.Publisher
	}
	
	if enhanced.BestMatch.ISBN != "" && enhanced.ISBN == "" {
		enhanced.ISBN = enhanced.BestMatch.ISBN
	}
	
	if enhanced.BestMatch.Description != "" && enhanced.Description == "" {
		enhanced.Description = enhanced.BestMatch.Description
	}

	enhanced.DataSource = "Web-Search-Enhanced"
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func extractTitleFromText(text string) string {
	// Simple extraction - could be improved with NLP
	lines := strings.Split(text, ".")
	if len(lines) > 0 {
		// Often the first sentence contains the title
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func extractAuthorFromText(text string) string {
	// Look for common patterns like "by Author Name"
	if idx := strings.Index(strings.ToLower(text), "by "); idx >= 0 {
		authorPart := text[idx+3:]
		// Take until next punctuation or end
		for i, ch := range authorPart {
			if ch == '.' || ch == ',' || ch == ';' || ch == '(' {
				return strings.TrimSpace(authorPart[:i])
			}
		}
	}
	return ""
}

func extractYearFromText(text string) int {
	// Look for 4-digit years
	for i := 0; i < len(text)-3; i++ {
		if text[i] >= '1' && text[i] <= '2' {
			yearStr := text[i:i+4]
			var year int
			if _, err := fmt.Sscanf(yearStr, "%d", &year); err == nil {
				if year >= 1000 && year <= 2100 {
					return year
				}
			}
		}
	}
	return 0
}