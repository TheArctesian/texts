package apis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"book-library/internal/models"
)

type OpenLibraryAPI struct {
	baseURL string
	client  *http.Client
}

type OpenLibrarySearchResponse struct {
	NumFound int                    `json:"numFound"`
	Docs     []OpenLibrarySearchDoc `json:"docs"`
}

type OpenLibrarySearchDoc struct {
	Key                    string   `json:"key"`
	Title                  string   `json:"title"`
	AuthorName             []string `json:"author_name"`
	FirstPublishYear       int      `json:"first_publish_year"`
	Publisher              []string `json:"publisher"`
	Language               []string `json:"language"`
	Subject                []string `json:"subject"`
	ISBN                   []string `json:"isbn"`
	HasFulltext            bool     `json:"has_fulltext"`
	CoverI                 int      `json:"cover_i"`
	EditionCount           int      `json:"edition_count"`
	FirstSentence          []string `json:"first_sentence"`
}

type OpenLibraryWorkResponse struct {
	Description interface{} `json:"description"`
	Title       string      `json:"title"`
	Authors     []struct {
		Author struct {
			Key string `json:"key"`
		} `json:"author"`
	} `json:"authors"`
	Subjects    []string `json:"subjects"`
	Created     struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"created"`
}

func NewOpenLibraryAPI(baseURL string) *OpenLibraryAPI {
	return &OpenLibraryAPI{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (o *OpenLibraryAPI) SearchByTitleAndAuthor(title, author string) (*models.Book, error) {
	params := url.Values{}
	
	if title != "" {
		params.Add("title", title)
	}
	if author != "" {
		params.Add("author", author)
	}
	params.Add("limit", "1")

	searchURL := fmt.Sprintf("%s/search.json?%s", o.baseURL, params.Encode())

	resp, err := o.client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var searchResponse OpenLibrarySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResponse.Docs) == 0 {
		return nil, fmt.Errorf("no books found")
	}

	book := o.convertSearchDocToBook(searchResponse.Docs[0])
	
	// Try to get additional details from work endpoint
	if workDetails, err := o.getWorkDetails(searchResponse.Docs[0].Key); err == nil {
		o.enrichBookWithWorkDetails(book, workDetails)
	}

	return book, nil
}

func (o *OpenLibraryAPI) SearchByISBN(isbn string) (*models.Book, error) {
	// First try the ISBN endpoint
	isbnURL := fmt.Sprintf("%s/isbn/%s.json", o.baseURL, isbn)
	
	resp, err := o.client.Get(isbnURL)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			// Process ISBN endpoint response - this would need additional implementation
			// For now, fall back to search
		}
	}

	// Fall back to search by ISBN
	params := url.Values{}
	params.Add("isbn", isbn)
	params.Add("limit", "1")

	searchURL := fmt.Sprintf("%s/search.json?%s", o.baseURL, params.Encode())

	resp, err = o.client.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var searchResponse OpenLibrarySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResponse.Docs) == 0 {
		return nil, fmt.Errorf("no books found for ISBN: %s", isbn)
	}

	return o.convertSearchDocToBook(searchResponse.Docs[0]), nil
}

func (o *OpenLibraryAPI) convertSearchDocToBook(doc OpenLibrarySearchDoc) *models.Book {
	book := &models.Book{
		ID:            doc.Key,
		Title:         doc.Title,
		Authors:       doc.AuthorName,
		Categories:    doc.Subject,
		ProcessedAt:   time.Now(),
		DataSource:    "openlibrary",
	}

	// Set dates if available
	if doc.FirstPublishYear != 0 {
		book.OriginalDate = &doc.FirstPublishYear
		book.ReleaseDate = &doc.FirstPublishYear
	}

	// Set primary author
	if len(doc.AuthorName) > 0 {
		book.Author = doc.AuthorName[0]
	}

	// Set ISBN
	if len(doc.ISBN) > 0 {
		book.ISBN = doc.ISBN[0]
	}

	// Set publisher
	if len(doc.Publisher) > 0 {
		book.Publisher = doc.Publisher[0]
	}

	// Set language
	if len(doc.Language) > 0 {
		book.EditionLanguage = doc.Language[0]
		book.OriginalLanguage = doc.Language[0]
	}

	// Set cover image URL if available
	if doc.CoverI > 0 {
		book.ThumbnailURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", doc.CoverI)
	}

	// Set description from first sentence if available
	if len(doc.FirstSentence) > 0 {
		book.Description = strings.Join(doc.FirstSentence, " ")
	}

	return book
}

func (o *OpenLibraryAPI) getWorkDetails(workKey string) (*OpenLibraryWorkResponse, error) {
	// Remove /works/ prefix if present
	workKey = strings.TrimPrefix(workKey, "/works/")
	
	workURL := fmt.Sprintf("%s/works/%s.json", o.baseURL, workKey)

	resp, err := o.client.Get(workURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("work details request failed with status: %d", resp.StatusCode)
	}

	var workResponse OpenLibraryWorkResponse
	if err := json.NewDecoder(resp.Body).Decode(&workResponse); err != nil {
		return nil, err
	}

	return &workResponse, nil
}

func (o *OpenLibraryAPI) enrichBookWithWorkDetails(book *models.Book, work *OpenLibraryWorkResponse) {
	// Add description if available and book doesn't have one
	if book.Description == "" && work.Description != nil {
		switch desc := work.Description.(type) {
		case string:
			book.Description = desc
		case map[string]interface{}:
			if value, ok := desc["value"].(string); ok {
				book.Description = value
			}
		}
	}

	// Add subjects/categories
	if len(work.Subjects) > 0 && len(book.Categories) == 0 {
		// Limit to first 10 subjects to avoid overwhelming data
		maxSubjects := 10
		if len(work.Subjects) < maxSubjects {
			maxSubjects = len(work.Subjects)
		}
		book.Categories = work.Subjects[:maxSubjects]
	}
}