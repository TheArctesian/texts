package models

import (
	"fmt"
	"strings"
	"time"
)

type Book struct {
	ID                      string    `json:"id"`
	Title                   string    `json:"title"`
	Author                  string    `json:"author"`
	Authors                 []string  `json:"authors,omitempty"`
	Description             string    `json:"description"`
	OriginalDate            *int      `json:"original_date"` // Negative for BC, nil if unknown
	ReleaseDate             *int      `json:"release_date"`  // nil if unknown
	OriginalLocationName    string    `json:"original_location_name"`
	OriginalLocationLat     float64   `json:"original_location_latitude"`
	OriginalLocationLng     float64   `json:"original_location_longitude"`
	OriginalLanguage        string    `json:"original_language"`
	EditionLanguage         string    `json:"edition_language"`
	ImagePath               string    `json:"image_path"`
	ISBN                    string    `json:"isbn,omitempty"`
	Publisher               string    `json:"publisher,omitempty"`
	Categories              []string  `json:"categories,omitempty"`
	PageCount               int       `json:"page_count,omitempty"`
	AverageRating           float64   `json:"average_rating,omitempty"`
	RatingsCount            int       `json:"ratings_count,omitempty"`
	ThumbnailURL            string    `json:"thumbnail_url,omitempty"`
	PreviewLink             string    `json:"preview_link,omitempty"`
	ExtractedText           string    `json:"extracted_text,omitempty"`
	ProcessedAt             time.Time `json:"processed_at"`
	DataSource              string    `json:"data_source"` // "google_books", "openlibrary", "manual"
	
	// Legacy fields for compatibility
	Year     int    `json:"year,omitempty"`     // Alias for OriginalDate
	Source   string `json:"source,omitempty"`   // Alias for DataSource
	Location string `json:"location,omitempty"` // Alias for OriginalLocationName
}

type Location struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Country   string  `json:"country,omitempty"`
}

// GetDisplayDate returns a human-readable date string
func (b *Book) GetDisplayDate() string {
	if b.OriginalDate == nil || *b.OriginalDate == 0 {
		return "Unknown"
	}
	if *b.OriginalDate < 0 {
		return fmt.Sprintf("%d BC", -*b.OriginalDate)
	}
	return fmt.Sprintf("%d AD", *b.OriginalDate)
}

// GetAuthorsString returns authors as a comma-separated string
func (b *Book) GetAuthorsString() string {
	if len(b.Authors) > 0 {
		return strings.Join(b.Authors, ", ")
	}
	if b.Author != "" {
		return b.Author
	}
	return "Unknown Author"
}

// GetSortableDate returns a sortable date (negative BC dates come first)
func (b *Book) GetSortableDate() int {
	if b.OriginalDate == nil {
		return 0
	}
	return *b.OriginalDate
}