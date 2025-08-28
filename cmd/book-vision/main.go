package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"book-library/internal/models"
)

type VisionProvider interface {
	ExtractText(imagePath string) (*BookExtraction, error)
}

type BookExtraction struct {
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	ISBN        string   `json:"isbn"`
	Publisher   string   `json:"publisher"`
	RawText     string   `json:"raw_text"`
	Confidence  float64  `json:"confidence"`
	Provider    string   `json:"provider"`
	Annotations []string `json:"annotations,omitempty"`
}

type GoogleVisionProvider struct {
	APIKey string
}

type AzureVisionProvider struct {
	APIKey   string
	Endpoint string
}

type AWSRekognitionProvider struct {
	Region string
	// AWS credentials handled via environment/profile
}

func main() {
	var (
		imagePath  = flag.String("image", "", "Path to book cover image")
		provider   = flag.String("provider", "google", "Vision API provider (google, azure, aws)")
		apiKey     = flag.String("api-key", "", "API key for the provider")
		endpoint   = flag.String("endpoint", "", "API endpoint (for Azure)")
		outputFile = flag.String("output", "", "Output file (stdout if not provided)")
		fallback   = flag.Bool("fallback", true, "Use Tesseract as fallback")
		quiet      = flag.Bool("q", false, "Quiet mode")
	)
	flag.Parse()

	if *imagePath == "" {
		// Read image path from stdin if not provided
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			*imagePath = scanner.Text()
		} else {
			log.Fatal("No image path provided")
		}
	}

	// Set API key from environment if not provided
	if *apiKey == "" {
		switch *provider {
		case "google":
			*apiKey = os.Getenv("GOOGLE_VISION_API_KEY")
		case "azure":
			*apiKey = os.Getenv("AZURE_VISION_API_KEY")
			if *endpoint == "" {
				*endpoint = os.Getenv("AZURE_VISION_ENDPOINT")
			}
		}
	}

	var extraction *BookExtraction
	var err error

	switch *provider {
	case "google":
		p := &GoogleVisionProvider{APIKey: *apiKey}
		extraction, err = p.ExtractText(*imagePath)
	case "azure":
		p := &AzureVisionProvider{APIKey: *apiKey, Endpoint: *endpoint}
		extraction, err = p.ExtractText(*imagePath)
	case "aws":
		p := &AWSRekognitionProvider{Region: "us-east-1"}
		extraction, err = p.ExtractText(*imagePath)
	default:
		log.Fatalf("Unknown provider: %s", *provider)
	}

	if err != nil && *fallback {
		if !*quiet {
			fmt.Fprintf(os.Stderr, "Vision API failed, falling back to Tesseract: %v\n", err)
		}
		extraction = extractWithTesseract(*imagePath)
	} else if err != nil {
		log.Fatalf("Vision API extraction failed: %v", err)
	}

	// Convert to book model
	book := &models.Book{
		Title:     extraction.Title,
		Author:    extraction.Author,
		ISBN:      extraction.ISBN,
		Publisher: extraction.Publisher,
		DataSource:    fmt.Sprintf("Vision-%s", extraction.Provider),
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Extracted with %s (confidence: %.2f%%):\n", 
			extraction.Provider, extraction.Confidence*100)
		fmt.Fprintf(os.Stderr, "  Title: %s\n", book.Title)
		fmt.Fprintf(os.Stderr, "  Author: %s\n", book.Author)
		if book.ISBN != "" {
			fmt.Fprintf(os.Stderr, "  ISBN: %s\n", book.ISBN)
		}
		if book.Publisher != "" {
			fmt.Fprintf(os.Stderr, "  Publisher: %s\n", book.Publisher)
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
	if err := encoder.Encode(book); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}

func (p *GoogleVisionProvider) ExtractText(imagePath string) (*BookExtraction, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %v", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	request := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]string{
					"content": base64Image,
				},
				"features": []map[string]interface{}{
					{
						"type":       "TEXT_DETECTION",
						"maxResults": 1,
					},
					{
						"type":       "DOCUMENT_TEXT_DETECTION",
						"maxResults": 1,
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", p.APIKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	extraction := &BookExtraction{
		Provider:   "Google Vision",
		Confidence: 0.85,
	}

	// Extract text from response
	if responses, ok := result["responses"].([]interface{}); ok && len(responses) > 0 {
		if response, ok := responses[0].(map[string]interface{}); ok {
			// Try full text annotation first
			if fullText, ok := response["fullTextAnnotation"].(map[string]interface{}); ok {
				if text, ok := fullText["text"].(string); ok {
					extraction.RawText = text
				}
			} else if textAnnotations, ok := response["textAnnotations"].([]interface{}); ok && len(textAnnotations) > 0 {
				// Fallback to text annotations
				if firstAnnotation, ok := textAnnotations[0].(map[string]interface{}); ok {
					if description, ok := firstAnnotation["description"].(string); ok {
						extraction.RawText = description
					}
				}
			}
		}
	}

	// Parse book information from raw text
	parseBookInfo(extraction)
	return extraction, nil
}

func (p *AzureVisionProvider) ExtractText(imagePath string) (*BookExtraction, error) {
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read image: %v", err)
	}

	url := fmt.Sprintf("%s/vision/v3.2/ocr", p.Endpoint)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(imageData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", p.APIKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	extraction := &BookExtraction{
		Provider:   "Azure Vision",
		Confidence: 0.83,
	}

	// Extract text from Azure response
	var textParts []string
	if regions, ok := result["regions"].([]interface{}); ok {
		for _, region := range regions {
			if regionMap, ok := region.(map[string]interface{}); ok {
				if lines, ok := regionMap["lines"].([]interface{}); ok {
					for _, line := range lines {
						if lineMap, ok := line.(map[string]interface{}); ok {
							if words, ok := lineMap["words"].([]interface{}); ok {
								var lineText []string
								for _, word := range words {
									if wordMap, ok := word.(map[string]interface{}); ok {
										if text, ok := wordMap["text"].(string); ok {
											lineText = append(lineText, text)
										}
									}
								}
								textParts = append(textParts, strings.Join(lineText, " "))
							}
						}
					}
				}
			}
		}
	}

	extraction.RawText = strings.Join(textParts, "\n")
	parseBookInfo(extraction)
	return extraction, nil
}

func (p *AWSRekognitionProvider) ExtractText(imagePath string) (*BookExtraction, error) {
	// AWS Rekognition implementation would go here
	// For now, returning a placeholder
	return nil, fmt.Errorf("AWS Rekognition not yet implemented")
}

func parseBookInfo(extraction *BookExtraction) {
	lines := strings.Split(extraction.RawText, "\n")
	
	// Common patterns for book covers
	isbnPattern := regexp.MustCompile(`(?i)ISBN[:\s-]*([0-9X-]+)`)
	authorPatterns := []string{
		`(?i)^by\s+(.+)$`,
		`(?i)author[:\s]+(.+)$`,
		`(?i)written\s+by\s+(.+)$`,
	}
	publisherPatterns := []string{
		`(?i)published\s+by\s+(.+)$`,
		`(?i)publisher[:\s]+(.+)$`,
		`(?i)([A-Z][a-z]+\s+(?:Books|Press|Publishers|Publishing|House))`,
	}

	// Extract ISBN
	if match := isbnPattern.FindStringSubmatch(extraction.RawText); len(match) > 1 {
		extraction.ISBN = cleanISBN(match[1])
	}

	// Extract title (usually the largest/first text block)
	if len(lines) > 0 {
		extraction.Title = cleanTitle(lines[0])
		
		// Sometimes title spans multiple lines
		if len(lines) > 1 && !containsAuthorKeywords(lines[1]) {
			extraction.Title += " " + cleanTitle(lines[1])
		}
	}

	// Extract author
	for _, pattern := range authorPatterns {
		re := regexp.MustCompile(pattern)
		for _, line := range lines {
			if match := re.FindStringSubmatch(line); len(match) > 1 {
				extraction.Author = cleanAuthor(match[1])
				break
			}
		}
		if extraction.Author != "" {
			break
		}
	}

	// If no author pattern matched, look for name-like text
	if extraction.Author == "" {
		extraction.Author = findAuthorName(lines)
	}

	// Extract publisher
	for _, pattern := range publisherPatterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(extraction.RawText); len(match) > 1 {
			extraction.Publisher = cleanPublisher(match[1])
			break
		}
	}
}

func extractWithTesseract(imagePath string) *BookExtraction {
	// Fallback to Tesseract OCR
	// This would call the existing book-ocr tool or use gosseract directly
	
	cmd := exec.Command("tesseract", imagePath, "-", "--psm", "6")
	output, err := cmd.Output()
	if err != nil {
		return &BookExtraction{
			Provider:   "Tesseract-Fallback",
			Confidence: 0.5,
		}
	}

	extraction := &BookExtraction{
		RawText:    string(output),
		Provider:   "Tesseract",
		Confidence: 0.6,
	}

	parseBookInfo(extraction)
	return extraction
}

// Helper functions
func cleanISBN(isbn string) string {
	// Remove all non-alphanumeric characters
	re := regexp.MustCompile(`[^0-9X]`)
	cleaned := re.ReplaceAllString(strings.ToUpper(isbn), "")
	
	// Validate length (ISBN-10 or ISBN-13)
	if len(cleaned) == 10 || len(cleaned) == 13 {
		return cleaned
	}
	return isbn
}

func cleanTitle(title string) string {
	// Remove common noise
	title = strings.TrimSpace(title)
	title = strings.Trim(title, "\"'")
	
	// Remove edition markers at the end
	re := regexp.MustCompile(`(?i)\s*\(?([0-9]+(?:st|nd|rd|th)?\s+edition|edition\s+[0-9]+)\)?$`)
	title = re.ReplaceAllString(title, "")
	
	return title
}

func cleanAuthor(author string) string {
	author = strings.TrimSpace(author)
	
	// Remove common prefixes
	prefixes := []string{"By ", "by ", "Author: ", "Written by "}
	for _, prefix := range prefixes {
		author = strings.TrimPrefix(author, prefix)
	}
	
	return author
}

func cleanPublisher(publisher string) string {
	return strings.TrimSpace(publisher)
}

func containsAuthorKeywords(text string) bool {
	keywords := []string{"by ", "By ", "author", "Author", "written"}
	textLower := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func findAuthorName(lines []string) string {
	// Look for lines that look like author names (capitalized words, not too long)
	namePattern := regexp.MustCompile(`^([A-Z][a-z]+(?:\s+[A-Z][a-z]+){1,3})$`)
	
	for i, line := range lines {
		// Skip first line (likely title)
		if i == 0 {
			continue
		}
		
		line = strings.TrimSpace(line)
		if match := namePattern.FindStringSubmatch(line); len(match) > 0 {
			// Check it's not a publisher name
			if !strings.Contains(strings.ToLower(line), "press") && 
			   !strings.Contains(strings.ToLower(line), "publishing") &&
			   !strings.Contains(strings.ToLower(line), "books") {
				return match[1]
			}
		}
	}
	
	return ""
}