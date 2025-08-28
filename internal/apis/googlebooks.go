package apis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"book-library/internal/models"
)

type GoogleBooksAPI struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

type GoogleBooksResponse struct {
	Items []GoogleBooksItem `json:"items"`
}

type GoogleBooksItem struct {
	ID         string                  `json:"id"`
	VolumeInfo GoogleBooksVolumeInfo   `json:"volumeInfo"`
}

type GoogleBooksVolumeInfo struct {
	Title               string                    `json:"title"`
	Authors             []string                  `json:"authors"`
	Description         string                    `json:"description"`
	PublishedDate       string                    `json:"publishedDate"`
	Publisher           string                    `json:"publisher"`
	PageCount           int                       `json:"pageCount"`
	Categories          []string                  `json:"categories"`
	AverageRating       float64                   `json:"averageRating"`
	RatingsCount        int                       `json:"ratingsCount"`
	ImageLinks          GoogleBooksImageLinks     `json:"imageLinks"`
	IndustryIdentifiers []GoogleBooksIdentifier   `json:"industryIdentifiers"`
	PreviewLink         string                    `json:"previewLink"`
	Language            string                    `json:"language"`
}

type GoogleBooksImageLinks struct {
	SmallThumbnail string `json:"smallThumbnail"`
	Thumbnail      string `json:"thumbnail"`
	Small          string `json:"small"`
	Medium         string `json:"medium"`
	Large          string `json:"large"`
	ExtraLarge     string `json:"extraLarge"`
}

type GoogleBooksIdentifier struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

func NewGoogleBooksAPI(apiKey, baseURL string) *GoogleBooksAPI {
	return &GoogleBooksAPI{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (g *GoogleBooksAPI) SearchByTitleAndAuthor(title, author string) (*models.Book, error) {
	query := buildSearchQuery(title, author)
	
	searchURL := fmt.Sprintf("%s/volumes?q=%s", g.baseURL, url.QueryEscape(query))
	if g.apiKey != "" {
		searchURL += "&key=" + g.apiKey
	}

	resp, err := g.client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var gbResponse GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&gbResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gbResponse.Items) == 0 {
		return nil, fmt.Errorf("no books found")
	}

	// Return the first result (most relevant)
	return g.convertToBook(gbResponse.Items[0]), nil
}

func (g *GoogleBooksAPI) SearchByISBN(isbn string) (*models.Book, error) {
	searchURL := fmt.Sprintf("%s/volumes?q=isbn:%s", g.baseURL, isbn)
	if g.apiKey != "" {
		searchURL += "&key=" + g.apiKey
	}

	resp, err := g.client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var gbResponse GoogleBooksResponse
	if err := json.NewDecoder(resp.Body).Decode(&gbResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gbResponse.Items) == 0 {
		return nil, fmt.Errorf("no books found for ISBN: %s", isbn)
	}

	return g.convertToBook(gbResponse.Items[0]), nil
}

func (g *GoogleBooksAPI) convertToBook(item GoogleBooksItem) *models.Book {
	book := &models.Book{
		ID:            item.ID,
		Title:         item.VolumeInfo.Title,
		Authors:       item.VolumeInfo.Authors,
		Description:   item.VolumeInfo.Description,
		Publisher:     item.VolumeInfo.Publisher,
		PageCount:     item.VolumeInfo.PageCount,
		Categories:    item.VolumeInfo.Categories,
		AverageRating: item.VolumeInfo.AverageRating,
		RatingsCount:  item.VolumeInfo.RatingsCount,
		PreviewLink:   item.VolumeInfo.PreviewLink,
		ProcessedAt:   time.Now(),
		DataSource:    "google_books",
	}

	// Set primary author
	if len(item.VolumeInfo.Authors) > 0 {
		book.Author = item.VolumeInfo.Authors[0]
	}

	// Extract ISBN
	for _, identifier := range item.VolumeInfo.IndustryIdentifiers {
		if identifier.Type == "ISBN_13" || identifier.Type == "ISBN_10" {
			book.ISBN = identifier.Identifier
			break
		}
	}

	// Set thumbnail URL
	if item.VolumeInfo.ImageLinks.Thumbnail != "" {
		book.ThumbnailURL = item.VolumeInfo.ImageLinks.Thumbnail
	} else if item.VolumeInfo.ImageLinks.SmallThumbnail != "" {
		book.ThumbnailURL = item.VolumeInfo.ImageLinks.SmallThumbnail
	}

	// Parse publication date
	if item.VolumeInfo.PublishedDate != "" {
		date := parsePublicationDate(item.VolumeInfo.PublishedDate)
		if date != 0 {
			book.ReleaseDate = &date
		}
	}

	// Set edition language
	if item.VolumeInfo.Language != "" {
		book.EditionLanguage = item.VolumeInfo.Language
	}

	return book
}

func buildSearchQuery(title, author string) string {
	var parts []string
	
	if title != "" {
		parts = append(parts, fmt.Sprintf("intitle:%s", title))
	}
	
	if author != "" {
		parts = append(parts, fmt.Sprintf("inauthor:%s", author))
	}
	
	return strings.Join(parts, " ")
}

func parsePublicationDate(dateStr string) int {
	// Try different date formats
	formats := []string{"2006", "2006-01", "2006-01-02"}
	
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Year()
		}
	}
	
	// Try to extract just the year if it's a longer string
	if len(dateStr) >= 4 {
		if year, err := strconv.Atoi(dateStr[:4]); err == nil && year > 0 && year < 3000 {
			return year
		}
	}
	
	return 0
}