package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ImageDir         string `yaml:"image_dir"`
	OutputFile       string `yaml:"output_file"`
	HugoDir          string `yaml:"hugo_dir"`
	GoogleBooksAPI   APIConfig `yaml:"google_books_api"`
	OpenLibraryAPI   APIConfig `yaml:"openlibrary_api"`
	TesseractConfig  TesseractConfig `yaml:"tesseract"`
}

type APIConfig struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type TesseractConfig struct {
	Language string `yaml:"language"`
	PageSegmentationMode int `yaml:"page_segmentation_mode"`
}

func Load(configPath string) (*Config, error) {
	// Set defaults
	cfg := &Config{
		ImageDir:   "./Data/IMGS/iPhone/Recents/",
		OutputFile: "./Data/books.json",
		HugoDir:    "./hugo-site/",
		GoogleBooksAPI: APIConfig{
			Enabled: true,
			BaseURL: "https://www.googleapis.com/books/v1",
		},
		OpenLibraryAPI: APIConfig{
			Enabled: true,
			BaseURL: "https://openlibrary.org",
		},
		TesseractConfig: TesseractConfig{
			Language: "eng",
			PageSegmentationMode: 6, // Uniform block of text
		},
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		if err := cfg.Save(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Load API keys from environment if not set in config
	if cfg.GoogleBooksAPI.APIKey == "" {
		cfg.GoogleBooksAPI.APIKey = os.Getenv("GOOGLE_BOOKS_API_KEY")
	}

	return cfg, nil
}

func (c *Config) Save(configPath string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}