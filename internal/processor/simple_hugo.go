package processor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"book-library/internal/models"
)

type SimpleHugoGenerator struct {
	outputDir string
}

func NewHugoGenerator(outputDir string) *SimpleHugoGenerator {
	return &SimpleHugoGenerator{
		outputDir: outputDir,
	}
}

func (h *SimpleHugoGenerator) Generate(books []models.Book) error {
	// Create directory structure
	if err := h.createDirectoryStructure(); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Generate Hugo config
	if err := h.generateHugoConfig(); err != nil {
		return fmt.Errorf("failed to generate Hugo config: %w", err)
	}

	// Generate content pages
	if err := h.generateContentPages(books); err != nil {
		return fmt.Errorf("failed to generate content pages: %w", err)
	}

	// Generate data files
	if err := h.generateDataFiles(books); err != nil {
		return fmt.Errorf("failed to generate data files: %w", err)
	}

	// Generate templates
	if err := h.generateTemplates(); err != nil {
		return fmt.Errorf("failed to generate templates: %w", err)
	}

	// Generate static assets
	if err := h.generateStaticAssets(); err != nil {
		return fmt.Errorf("failed to generate static assets: %w", err)
	}

	return nil
}

func (h *SimpleHugoGenerator) createDirectoryStructure() error {
	dirs := []string{
		"content",
		"layouts/_default",
		"static/css",
		"static/js", 
		"static/data",
		"static/images",
	}

	for _, dir := range dirs {
		fullPath := filepath.Join(h.outputDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (h *SimpleHugoGenerator) generateHugoConfig() error {
	config := `
baseURL = "http://localhost:1313"
languageCode = "en-us"
title = "Book Library"

[markup]
  [markup.goldmark]
    [markup.goldmark.renderer]
      unsafe = true

[params]
  description = "OCR-based Book Library System"
  author = "Book Library System"
`

	return os.WriteFile(filepath.Join(h.outputDir, "hugo.toml"), []byte(strings.TrimSpace(config)), 0644)
}

func (h *SimpleHugoGenerator) generateContentPages(books []models.Book) error {
	// Home page
	homeContent := `---
title: "Book Library"
description: "OCR-based Book Library System"
---

# Book Library

Welcome to your book library! This system uses OCR to extract book information from cover images.

## Features

- 📚 **Books Processed**: {{.Site.Data.books | len}}
- 🔍 **OCR-powered**: Automatic text extraction from book covers
- 📊 **Timeline View**: Visualize books chronologically
- 🗺️ **Geography View**: See book origins on a world map

[View All Books](/books/) | [Timeline](/timeline/) | [Geography](/geography/)
`

	if err := os.WriteFile(filepath.Join(h.outputDir, "content/_index.md"), []byte(strings.TrimSpace(homeContent)), 0644); err != nil {
		return err
	}

	// Books listing page
	booksContent := `---
title: "All Books"
layout: "books"
---

# All Books

Browse all books in the library:
`

	if err := os.WriteFile(filepath.Join(h.outputDir, "content/books.md"), []byte(strings.TrimSpace(booksContent)), 0644); err != nil {
		return err
	}

	// Timeline page
	timelineContent := `---
title: "Timeline"
layout: "timeline"
---

# Book Timeline

Explore books chronologically:
`

	if err := os.WriteFile(filepath.Join(h.outputDir, "content/timeline.md"), []byte(strings.TrimSpace(timelineContent)), 0644); err != nil {
		return err
	}

	// Geography page
	geographyContent := `---
title: "Geography"
layout: "geography"
---

# Book Geography

Explore book origins on the world map:
`

	return os.WriteFile(filepath.Join(h.outputDir, "content/geography.md"), []byte(strings.TrimSpace(geographyContent)), 0644)
}

func (h *SimpleHugoGenerator) generateDataFiles(books []models.Book) error {
	// Sort books by title
	sort.Slice(books, func(i, j int) bool {
		return books[i].Title < books[j].Title
	})

	// Generate books.json
	booksJSON, err := json.MarshalIndent(books, "", "  ")
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(filepath.Join(h.outputDir, "static/data/books.json"), booksJSON, 0644); err != nil {
		return err
	}

	// Generate timeline data (simplified - using ProcessedAt since no dates from OCR)
	timelineData := make(map[string]interface{})
	timelineData["books"] = books
	timelineData["title"] = "Book Processing Timeline"

	timelineJSON, err := json.MarshalIndent(timelineData, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(h.outputDir, "static/data/timeline.json"), timelineJSON, 0644); err != nil {
		return err
	}

	// Generate geography data (placeholder since OCR doesn't extract locations)
	geographyData := make(map[string]interface{})
	geographyData["books"] = books
	geographyData["locations"] = []interface{}{} // Empty for now

	geographyJSON, err := json.MarshalIndent(geographyData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(h.outputDir, "static/data/geography.json"), geographyJSON, 0644)
}

func (h *SimpleHugoGenerator) generateTemplates() error {
	// Base template
	baseTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ if .Title }}{{ .Title }} - {{ end }}{{ .Site.Title }}</title>
    <link rel="stylesheet" href="/css/style.css">
</head>
<body>
    <nav>
        <div class="nav-container">
            <a href="/" class="nav-brand">📚 Book Library</a>
            <div class="nav-links">
                <a href="/books/">Books</a>
                <a href="/timeline/">Timeline</a>
                <a href="/geography/">Geography</a>
            </div>
        </div>
    </nav>
    
    <main>
        {{ block "main" . }}{{ end }}
    </main>
    
    <footer>
        <p>&copy; 2025 Book Library - OCR System</p>
    </footer>
</body>
</html>`

	if err := os.WriteFile(filepath.Join(h.outputDir, "layouts/_default/baseof.html"), []byte(baseTemplate), 0644); err != nil {
		return err
	}

	// Single page template
	singleTemplate := `{{ define "main" }}
<div class="container">
    <h1>{{ .Title }}</h1>
    <div class="content">
        {{ .Content }}
    </div>
</div>
{{ end }}`

	if err := os.WriteFile(filepath.Join(h.outputDir, "layouts/_default/single.html"), []byte(singleTemplate), 0644); err != nil {
		return err
	}

	// Books template
	booksTemplate := `{{ define "main" }}
<div class="container">
    <h1>{{ .Title }}</h1>
    {{ .Content }}
    
    <div id="books-list">
        <!-- Books will be loaded via JavaScript -->
        <p>Loading books...</p>
    </div>
</div>

<script>
fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        const container = document.getElementById('books-list');
        if (books.length === 0) {
            container.innerHTML = '<p>No books found. Process some images first!</p>';
            return;
        }
        
        const bookCards = books.map(book => ` + "`" + `
            <div class="book-card">
                <h3>${book.title || 'Unknown Title'}</h3>
                <p><strong>Author:</strong> ${book.author || 'Unknown Author'}</p>
                <p><strong>Source:</strong> ${book.data_source || 'OCR'}</p>
                <p><strong>Processed:</strong> ${new Date(book.processed_at).toLocaleDateString()}</p>
                ${book.description ? ` + "`<p><strong>Description:</strong> ${book.description}</p>`" + ` : ''}
                <details>
                    <summary>OCR Text</summary>
                    <pre>${book.extracted_text || 'No extracted text'}</pre>
                </details>
            </div>
        ` + "`" + `).join('');
        
        container.innerHTML = ` + "`<div class=\"books-grid\">${bookCards}</div>`" + `;
    })
    .catch(error => {
        console.error('Error loading books:', error);
        document.getElementById('books-list').innerHTML = '<p>Error loading books.</p>';
    });
</script>
{{ end }}`

	if err := os.WriteFile(filepath.Join(h.outputDir, "layouts/_default/books.html"), []byte(booksTemplate), 0644); err != nil {
		return err
	}

	// Timeline template
	timelineTemplate := `{{ define "main" }}
<div class="container">
    <h1>{{ .Title }}</h1>
    {{ .Content }}
    
    <div id="timeline-viz">
        <p>Loading timeline...</p>
    </div>
</div>

<script>
fetch('/data/timeline.json')
    .then(response => response.json())
    .then(data => {
        const container = document.getElementById('timeline-viz');
        const books = data.books || [];
        
        if (books.length === 0) {
            container.innerHTML = '<p>No books to display in timeline.</p>';
            return;
        }
        
        // Simple timeline by processing date
        const sorted = books.sort((a, b) => new Date(a.processed_at) - new Date(b.processed_at));
        const timelineHTML = sorted.map((book, index) => ` + "`" + `
            <div class="timeline-item">
                <div class="timeline-date">${new Date(book.processed_at).toLocaleDateString()}</div>
                <div class="timeline-content">
                    <h3>${book.title}</h3>
                    <p>by ${book.author || 'Unknown Author'}</p>
                </div>
            </div>
        ` + "`" + `).join('');
        
        container.innerHTML = ` + "`<div class=\"timeline\">${timelineHTML}</div>`" + `;
    });
</script>
{{ end }}`

	if err := os.WriteFile(filepath.Join(h.outputDir, "layouts/_default/timeline.html"), []byte(timelineTemplate), 0644); err != nil {
		return err
	}

	// Geography template
	geographyTemplate := `{{ define "main" }}
<div class="container">
    <h1>{{ .Title }}</h1>
    {{ .Content }}
    
    <div id="geography-viz">
        <p>Geography visualization not implemented yet (requires book origin data).</p>
        <p>This would show a world map with book origins if location data was available from OCR or manual entry.</p>
    </div>
</div>
{{ end }}`

	if err := os.WriteFile(filepath.Join(h.outputDir, "layouts/_default/geography.html"), []byte(geographyTemplate), 0644); err != nil {
		return err
	}

	// Home template (index)
	homeTemplate := `{{ define "main" }}
<div class="container">
    <h1>{{ .Title }}</h1>
    {{ .Content }}
    
    <div class="stats">
        <div class="stat-card">
            <h3>📚 Books Processed</h3>
            <p id="book-count">Loading...</p>
        </div>
        <div class="stat-card">
            <h3>🔍 OCR System</h3>
            <p>Automatic text extraction</p>
        </div>
        <div class="stat-card">
            <h3>📊 Timeline View</h3>
            <p>Chronological browsing</p>
        </div>
    </div>
    
    <div class="quick-links">
        <a href="/books/" class="btn">View All Books</a>
        <a href="/timeline/" class="btn">Timeline</a>
        <a href="/geography/" class="btn">Geography</a>
    </div>
</div>

<script>
fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        document.getElementById('book-count').textContent = books.length;
    })
    .catch(error => {
        document.getElementById('book-count').textContent = '0';
    });
</script>
{{ end }}`

	return os.WriteFile(filepath.Join(h.outputDir, "layouts/index.html"), []byte(homeTemplate), 0644)
}

func (h *SimpleHugoGenerator) generateStaticAssets() error {
	// CSS
	css := `
:root {
    --primary-color: #2c5aa0;
    --secondary-color: #4a90a4;
    --background-color: #f8f9fa;
    --text-color: #333;
    --border-color: #dee2e6;
}

* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
    line-height: 1.6;
    color: var(--text-color);
    background-color: var(--background-color);
}

nav {
    background: var(--primary-color);
    padding: 1rem 0;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.nav-container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 0 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.nav-brand {
    color: white;
    text-decoration: none;
    font-size: 1.5rem;
    font-weight: bold;
}

.nav-links a {
    color: white;
    text-decoration: none;
    margin-left: 2rem;
    transition: color 0.3s;
}

.nav-links a:hover {
    color: #cce7ff;
}

.container {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem 1rem;
}

h1 {
    color: var(--primary-color);
    margin-bottom: 1rem;
}

h3 {
    color: var(--secondary-color);
    margin-bottom: 0.5rem;
}

.books-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1.5rem;
    margin-top: 2rem;
}

.book-card {
    background: white;
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 1.5rem;
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
    transition: transform 0.3s, box-shadow 0.3s;
}

.book-card:hover {
    transform: translateY(-2px);
    box-shadow: 0 4px 8px rgba(0,0,0,0.1);
}

.book-card p {
    margin-bottom: 0.5rem;
}

.book-card details {
    margin-top: 1rem;
}

.book-card pre {
    background: #f8f9fa;
    padding: 0.5rem;
    border-radius: 4px;
    font-size: 0.85rem;
    max-height: 200px;
    overflow-y: auto;
    white-space: pre-wrap;
    margin-top: 0.5rem;
}

.timeline {
    margin-top: 2rem;
}

.timeline-item {
    display: flex;
    margin-bottom: 2rem;
    padding-bottom: 2rem;
    border-bottom: 1px solid var(--border-color);
}

.timeline-date {
    flex: 0 0 150px;
    font-weight: bold;
    color: var(--secondary-color);
}

.timeline-content {
    flex: 1;
    padding-left: 2rem;
}

footer {
    background: var(--primary-color);
    color: white;
    text-align: center;
    padding: 2rem 0;
    margin-top: 4rem;
}

.stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 1.5rem;
    margin: 2rem 0;
}

.stat-card {
    background: white;
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 1.5rem;
    text-align: center;
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
}

.stat-card h3 {
    margin-bottom: 0.5rem;
    color: var(--primary-color);
}

.quick-links {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-top: 2rem;
    flex-wrap: wrap;
}

.btn {
    display: inline-block;
    background: var(--primary-color);
    color: white;
    text-decoration: none;
    padding: 0.75rem 1.5rem;
    border-radius: 8px;
    font-weight: bold;
    transition: background-color 0.3s;
}

.btn:hover {
    background: var(--secondary-color);
}

@media (max-width: 768px) {
    .nav-container {
        flex-direction: column;
        gap: 1rem;
    }
    
    .nav-links a {
        margin-left: 1rem;
    }
    
    .timeline-item {
        flex-direction: column;
    }
    
    .timeline-content {
        padding-left: 0;
        padding-top: 0.5rem;
    }
    
    .quick-links {
        flex-direction: column;
        align-items: center;
    }
}
`

	return os.WriteFile(filepath.Join(h.outputDir, "static/css/style.css"), []byte(strings.TrimSpace(css)), 0644)
}