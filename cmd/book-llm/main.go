package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"book-library/internal/models"
)

type LLMRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	MaxTokens int      `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type ImageContent struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *ImageURL   `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL string `json:"url"`
}

type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type EnhancedBook struct {
	models.Book
	LLMAnalysis     map[string]interface{} `json:"llm_analysis"`
	ImageAnalysis   string                 `json:"image_analysis"`
	ConfidenceScore float64                `json:"confidence_score"`
}

func main() {
	var (
		inputFile  = flag.String("input", "", "JSON input file (reads from stdin if not provided)")
		outputFile = flag.String("output", "", "Output file (stdout if not provided)")
		imagePath  = flag.String("image", "", "Path to book cover image")
		provider   = flag.String("provider", getEnvDefault("LLM_PROVIDER", "ollama"), "LLM provider (ollama, openai, anthropic)")
		model      = flag.String("model", getEnvDefault("LLM_MODEL", "gemma3:27b"), "Model to use (e.g., gemma3:27b for Ollama)")
		apiKey     = flag.String("api-key", os.Getenv("OPENAI_API_KEY"), "API key (not needed for Ollama)")
		baseURL    = flag.String("base-url", getEnvDefault("LLM_BASE_URL", "http://localhost:11434"), "API base URL")
		quiet      = flag.Bool("q", false, "Quiet mode")
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

	// If image path provided, use it; otherwise try to use book's image path
	if *imagePath == "" && book.ImagePath != "" {
		*imagePath = book.ImagePath
	}

	enhanced := enhanceWithLLM(book, *imagePath, *provider, *model, *apiKey, *baseURL)

	if !*quiet {
		fmt.Fprintf(os.Stderr, "LLM Enhancement complete:\n")
		fmt.Fprintf(os.Stderr, "  Title: %s\n", enhanced.Title)
		fmt.Fprintf(os.Stderr, "  Author: %s\n", enhanced.Author)
		fmt.Fprintf(os.Stderr, "  Year: %d\n", enhanced.OriginalDate)
		fmt.Fprintf(os.Stderr, "  Confidence: %.2f%%\n", enhanced.ConfidenceScore*100)
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

func enhanceWithLLM(book models.Book, imagePath, provider, model, apiKey, baseURL string) EnhancedBook {
	enhanced := EnhancedBook{
		Book:        book,
		LLMAnalysis: make(map[string]interface{}),
	}

	// Prepare the prompt
	prompt := fmt.Sprintf(`Analyze this book and provide accurate metadata including geographic information. 
Current OCR/API data:
- Title: %s
- Author: %s
- ISBN: %s
- Year: %d
- Publisher: %s

IMPORTANT: I need complete location and date information for timeline and map visualizations.

Please verify and correct this information. Return a JSON object with:
{
  "title": "correct full title",
  "author": "correct author name(s)", 
  "isbn": "ISBN if visible",
  "year": publication year as integer (REQUIRED - find original publication year),
  "publisher": "publisher name",
  "genre": "book genre/category",
  "original_language": "original language (e.g., English, French, German, etc.)",
  "edition": "edition information if visible",
  "description": "brief description of the book",
  "publication_location": "city and country where originally published (e.g., London, England)",
  "author_origin": "country or region where author is from (e.g., England, United States, Ancient Greece)",
  "setting_location": "if book has a specific geographic setting, the primary location",
  "corrections_made": ["list of any corrections"],
  "confidence": 0.0 to 1.0
}

CRITICAL REQUIREMENTS:
- ALWAYS provide a publication year (even if estimated based on author's lifetime)
- ALWAYS provide publication_location (city, country where first published)  
- ALWAYS provide author_origin (author's nationality/origin)
- If the book is about a specific place, include setting_location
- Research the book if you recognize it to provide accurate historical context
- For classics, provide original publication date, not modern edition dates

Focus on accuracy. If you can see the book cover in the image, use that as the primary source.`,
		book.Title, book.Author, book.ISBN, book.OriginalDate, book.Publisher)

	// Prepare the request based on provider
	var response string
	var err error

	switch provider {
	case "openai":
		response, err = callOpenAI(prompt, imagePath, model, apiKey, baseURL)
	case "anthropic":
		response, err = callAnthropic(prompt, imagePath, model, apiKey)
	case "ollama":
		response, err = callOllama(prompt, imagePath, model, baseURL)
	default:
		log.Fatalf("Unknown provider: %s", provider)
	}

	if err != nil {
		log.Printf("LLM call failed: %v", err)
		enhanced.ConfidenceScore = 0.0
		return enhanced
	}

	// Parse LLM response
	var llmData map[string]interface{}
	if err := json.Unmarshal([]byte(response), &llmData); err != nil {
		// Try to extract JSON from the response if it's wrapped in text
		start := strings.Index(response, "{")
		end := strings.LastIndex(response, "}")
		if start >= 0 && end > start {
			jsonStr := response[start : end+1]
			if err := json.Unmarshal([]byte(jsonStr), &llmData); err != nil {
				log.Printf("Failed to parse LLM response: %v", err)
				enhanced.ImageAnalysis = response
				enhanced.ConfidenceScore = 0.5
				return enhanced
			}
		}
	}

	// Apply corrections from LLM
	if title, ok := llmData["title"].(string); ok && title != "" {
		enhanced.Title = title
	}
	if author, ok := llmData["author"].(string); ok && author != "" {
		enhanced.Author = author
	}
	if isbn, ok := llmData["isbn"].(string); ok && isbn != "" {
		enhanced.ISBN = isbn
	}
	if yearFloat, ok := llmData["year"].(float64); ok {
		year := int(yearFloat)
		enhanced.OriginalDate = &year
	}
	if publisher, ok := llmData["publisher"].(string); ok && publisher != "" {
		enhanced.Publisher = publisher
	}
	if genre, ok := llmData["genre"].(string); ok {
		enhanced.LLMAnalysis["genre"] = genre
	}
	if description, ok := llmData["description"].(string); ok {
		enhanced.Description = description
	}
	
	// Extract location information
	if pubLocation, ok := llmData["publication_location"].(string); ok && pubLocation != "" {
		enhanced.OriginalLocationName = pubLocation
		// Try to geocode the location (simplified approach)
		lat, lng := geocodeLocation(pubLocation)
		enhanced.OriginalLocationLat = lat
		enhanced.OriginalLocationLng = lng
	}
	if authorOrigin, ok := llmData["author_origin"].(string); ok && authorOrigin != "" {
		// If no publication location, use author origin
		if enhanced.OriginalLocationName == "" {
			enhanced.OriginalLocationName = authorOrigin
			lat, lng := geocodeLocation(authorOrigin)
			enhanced.OriginalLocationLat = lat
			enhanced.OriginalLocationLng = lng
		}
	}
	if settingLocation, ok := llmData["setting_location"].(string); ok && settingLocation != "" {
		// Store setting location in LLM analysis for potential future use
		enhanced.LLMAnalysis["setting_location"] = settingLocation
	}
	
	if confidence, ok := llmData["confidence"].(float64); ok {
		enhanced.ConfidenceScore = confidence
	} else {
		enhanced.ConfidenceScore = 0.7
	}

	enhanced.LLMAnalysis = llmData
	enhanced.DataSource = "LLM-Enhanced"

	return enhanced
}

func callOpenAI(prompt, imagePath, model, apiKey, baseURL string) (string, error) {
	var messages []Message

	if imagePath != "" && fileExists(imagePath) {
		// Read and encode image
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			return "", fmt.Errorf("failed to read image: %v", err)
		}
		
		base64Image := base64.StdEncoding.EncodeToString(imageData)
		imageURL := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)

		messages = []Message{
			{
				Role: "user",
				Content: []ImageContent{
					{
						Type: "text",
						Text: prompt,
					},
					{
						Type: "image_url",
						ImageURL: &ImageURL{
							URL: imageURL,
						},
					},
				},
			},
		}
	} else {
		messages = []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		}
	}

	reqBody := LLMRequest{
		Model:    model,
		Messages: messages,
		MaxTokens: 1000,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return "", err
	}

	if len(llmResp.Choices) > 0 {
		return llmResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from LLM")
}

func callAnthropic(prompt, imagePath, model, apiKey string) (string, error) {
	// Anthropic Claude API implementation
	baseURL := "https://api.anthropic.com/v1/messages"
	
	var content []map[string]interface{}
	content = append(content, map[string]interface{}{
		"type": "text",
		"text": prompt,
	})

	if imagePath != "" && fileExists(imagePath) {
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			return "", fmt.Errorf("failed to read image: %v", err)
		}
		
		base64Image := base64.StdEncoding.EncodeToString(imageData)
		content = append(content, map[string]interface{}{
			"type": "image",
			"source": map[string]string{
				"type":         "base64",
				"media_type":   "image/jpeg",
				"data":         base64Image,
			},
		})
	}

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": 1000,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": content,
			},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", baseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
		if msg, ok := content[0].(map[string]interface{}); ok {
			if text, ok := msg["text"].(string); ok {
				return text, nil
			}
		}
	}

	return "", fmt.Errorf("no response from Anthropic")
}

func callOllama(prompt, imagePath, model, baseURL string) (string, error) {
	// Ollama local API implementation
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	
	// First check if Ollama is running
	resp, err := http.Get(baseURL + "api/version")
	if err != nil {
		return "", fmt.Errorf("Ollama not running. Please start it with: ollama serve\nOriginal error: %v", err)
	}
	resp.Body.Close()
	
	// Check if the model is available
	modelsResp, err := http.Get(baseURL + "api/tags")
	if err != nil {
		log.Printf("Warning: Could not check available models: %v", err)
	} else {
		var models map[string]interface{}
		json.NewDecoder(modelsResp.Body).Decode(&models)
		modelsResp.Body.Close()
		
		// Simple check if model exists
		if modelsData, ok := models["models"].([]interface{}); ok {
			found := false
			for _, m := range modelsData {
				if modelData, ok := m.(map[string]interface{}); ok {
					if name, ok := modelData["name"].(string); ok && strings.HasPrefix(name, model) {
						found = true
						break
					}
				}
			}
			if !found {
				log.Printf("Warning: Model '%s' not found. Consider pulling it with: ollama pull %s", model, model)
			}
		}
	}
	
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1,  // Low temperature for more consistent book metadata
			"top_p": 0.9,
		},
	}

	if imagePath != "" && fileExists(imagePath) {
		imageData, err := os.ReadFile(imagePath)
		if err != nil {
			return "", fmt.Errorf("failed to read image: %v", err)
		}
		base64Image := base64.StdEncoding.EncodeToString(imageData)
		reqBody["images"] = []string{base64Image}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err = http.Post(baseURL+"api/generate", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if response, ok := result["response"].(string); ok {
		return response, nil
	}

	return "", fmt.Errorf("no response from Ollama")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getEnvDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Simple geocoding function with common locations
func geocodeLocation(location string) (float64, float64) {
	// Convert to lowercase for matching
	loc := strings.ToLower(strings.TrimSpace(location))
	
	// Common locations and their coordinates
	locations := map[string][2]float64{
		// Countries (capital cities as default)
		"england":         {51.5074, -0.1278},  // London
		"united kingdom":  {51.5074, -0.1278},  // London
		"uk":              {51.5074, -0.1278},  // London
		"united states":   {40.7128, -74.0060}, // New York
		"usa":             {40.7128, -74.0060}, // New York
		"france":          {48.8566, 2.3522},   // Paris
		"germany":         {52.5200, 13.4050},  // Berlin
		"italy":           {41.9028, 12.4964},  // Rome
		"spain":           {40.4168, -3.7038},  // Madrid
		"russia":          {55.7558, 37.6176},  // Moscow
		"china":           {39.9042, 116.4074}, // Beijing
		"japan":           {35.6762, 139.6503}, // Tokyo
		"india":           {28.6139, 77.2090},  // Delhi
		"greece":          {37.9755, 23.7348},  // Athens
		"ancient greece":  {37.9755, 23.7348},  // Athens
		
		// Major cities
		"london":          {51.5074, -0.1278},
		"london, england": {51.5074, -0.1278},
		"new york":        {40.7128, -74.0060},
		"paris":           {48.8566, 2.3522},
		"paris, france":   {48.8566, 2.3522},
		"berlin":          {52.5200, 13.4050},
		"berlin, germany": {52.5200, 13.4050},
		"rome":            {41.9028, 12.4964},
		"rome, italy":     {41.9028, 12.4964},
		"madrid":          {40.4168, -3.7038},
		"madrid, spain":   {40.4168, -3.7038},
		"moscow":          {55.7558, 37.6176},
		"moscow, russia":  {55.7558, 37.6176},
		"beijing":         {39.9042, 116.4074},
		"beijing, china":  {39.9042, 116.4074},
		"tokyo":           {35.6762, 139.6503},
		"tokyo, japan":    {35.6762, 139.6503},
		"athens":          {37.9755, 23.7348},
		"athens, greece":  {37.9755, 23.7348},
		"vienna":          {48.2082, 16.3738},
		"vienna, austria": {48.2082, 16.3738},
		"amsterdam":       {52.3676, 4.9041},
		"dublin":          {53.3498, -6.2603},
		"edinburgh":       {55.9533, -3.1883},
		
		// Publishers' locations
		"penguin classics": {51.5074, -0.1278}, // London
		"oxford":          {51.7520, -1.2577},  // Oxford
		"cambridge":       {52.2053, 0.1218},   // Cambridge
		"harvard":         {42.3736, -71.1097}, // Cambridge, MA
		"mit":             {42.3601, -71.0942}, // Cambridge, MA
		"stanford":        {37.4275, -122.1697}, // Stanford, CA
	}
	
	// Try exact match first
	if coords, exists := locations[loc]; exists {
		return coords[0], coords[1]
	}
	
	// Try partial matches
	for key, coords := range locations {
		if strings.Contains(loc, key) || strings.Contains(key, loc) {
			return coords[0], coords[1]
		}
	}
	
	// Default to London if no match (many English-language books)
	return 51.5074, -0.1278
}