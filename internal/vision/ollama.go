package vision

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type OllamaClient struct {
	BaseURL string
	Model   string
	Timeout time.Duration
}

type OllamaRequest struct {
	Model  string   `json:"model"`
	Prompt string   `json:"prompt"`
	Images []string `json:"images,omitempty"`
	Stream bool     `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type BookInfo struct {
	Title               string  `json:"title"`
	Author             string  `json:"author"`
	ISBN               string  `json:"isbn,omitempty"`
	Publisher          string  `json:"publisher,omitempty"`
	PublicationYear    string  `json:"publication_year,omitempty"`
	Language           string  `json:"language,omitempty"`
	Genre              string  `json:"genre,omitempty"`
	Description        string  `json:"description,omitempty"`
	Series             string  `json:"series,omitempty"`
	Edition            string  `json:"edition,omitempty"`
	Confidence         float64 `json:"confidence"`
}

func NewOllamaClient(baseURL, model string) *OllamaClient {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "gemma2:27b"
	}
	return &OllamaClient{
		BaseURL: baseURL,
		Model:   model,
		Timeout: 60 * time.Second,
	}
}

func (c *OllamaClient) AnalyzeBookCover(imagePath string) (*BookInfo, error) {
	imageData, err := c.encodeImage(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	prompt := `Analyze this book cover image and extract the following information in JSON format:
{
  "title": "exact title as shown on cover",
  "author": "author name(s)",
  "isbn": "ISBN if visible",
  "publisher": "publisher name if visible",
  "publication_year": "year if visible",
  "language": "language of the text",
  "genre": "book genre/category if determinable",
  "description": "brief description based on cover",
  "series": "series name if part of a series",
  "edition": "edition information if visible",
  "confidence": 0.95
}

Be precise and only include information that is clearly visible on the cover. Set confidence between 0-1 based on clarity.`

	req := OllamaRequest{
		Model:  c.Model,
		Prompt: prompt,
		Images: []string{imageData},
		Stream: false,
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze image: %w", err)
	}

	bookInfo, err := c.parseBookInfo(resp.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse book info: %w", err)
	}

	return bookInfo, nil
}

func (c *OllamaClient) encodeImage(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (c *OllamaClient) sendRequest(req OllamaRequest) (*OllamaResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: c.Timeout}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, err
	}

	var resp OllamaResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (c *OllamaClient) parseBookInfo(response string) (*BookInfo, error) {
	// Extract JSON from response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	
	if start == -1 || end == -1 || start > end {
		// If no JSON found, try to parse the text
		return c.parseTextResponse(response), nil
	}

	jsonStr := response[start : end+1]
	
	var bookInfo BookInfo
	if err := json.Unmarshal([]byte(jsonStr), &bookInfo); err != nil {
		// Fallback to text parsing
		return c.parseTextResponse(response), nil
	}

	// Clean up extracted data
	bookInfo.Title = strings.TrimSpace(bookInfo.Title)
	bookInfo.Author = strings.TrimSpace(bookInfo.Author)
	
	return &bookInfo, nil
}

func (c *OllamaClient) parseTextResponse(text string) *BookInfo {
	info := &BookInfo{
		Confidence: 0.5,
	}

	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		
		if strings.Contains(lower, "title:") {
			info.Title = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(lower, "author:") {
			info.Author = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(lower, "isbn:") {
			info.ISBN = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(lower, "publisher:") {
			info.Publisher = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.Contains(lower, "year:") {
			info.PublicationYear = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

	return info
}