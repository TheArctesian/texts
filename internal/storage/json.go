package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"book-library/internal/models"
)

type JSONStorage struct {
	filePath string
}

type Storage interface {
	LoadBooks() ([]models.Book, error)
	SaveBooks(books []models.Book) error
}

func NewJSONStorage(filePath string) *JSONStorage {
	return &JSONStorage{
		filePath: filePath,
	}
}

func (j *JSONStorage) LoadBooks() ([]models.Book, error) {
	if _, err := os.Stat(j.filePath); os.IsNotExist(err) {
		return []models.Book{}, nil
	}

	data, err := os.ReadFile(j.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var books []models.Book
	if err := json.Unmarshal(data, &books); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return books, nil
}

func (j *JSONStorage) SaveBooks(books []models.Book) error {
	// Sort books by original date for consistent ordering
	sort.Slice(books, func(i, k int) bool {
		return books[i].GetSortableDate() < books[k].GetSortableDate()
	})

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(j.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(j.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}