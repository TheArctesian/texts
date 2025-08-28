package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"book-library/internal/models"
)

func main() {
	var (
		outputDir = flag.String("output", "./hugo-site", "Hugo site output directory")
		inputFile = flag.String("input", "", "JSON input file (reads from stdin if not provided)")
		quiet     = flag.Bool("q", false, "Quiet mode")
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

	var books []models.Book
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&books); err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}

	if err := generateHugoSite(*outputDir, books); err != nil {
		log.Fatalf("Failed to generate Hugo site: %v", err)
	}

	if !*quiet {
		fmt.Fprintf(os.Stderr, "Generated Hugo site in %s with %d books\n", *outputDir, len(books))
	}
}

func generateHugoSite(outputDir string, books []models.Book) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	dirs := []string{
		filepath.Join(outputDir, "content"),
		filepath.Join(outputDir, "layouts", "_default"),
		filepath.Join(outputDir, "static", "css"),
		filepath.Join(outputDir, "static", "js"),
		filepath.Join(outputDir, "static", "data"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := createHugoConfig(outputDir); err != nil {
		return err
	}

	if err := createContent(outputDir); err != nil {
		return err
	}

	if err := createLayouts(outputDir); err != nil {
		return err
	}

	if err := createStaticFiles(outputDir, books); err != nil {
		return err
	}

	return nil
}

func createHugoConfig(outputDir string) error {
	config := `baseURL = "/"
title = "Library"
languageCode = "en-us"

[params]
  description = "Digital book collection"

[markup.goldmark.renderer]
  unsafe = true
`
	return os.WriteFile(filepath.Join(outputDir, "hugo.toml"), []byte(config), 0644)
}

func createContent(outputDir string) error {
	pages := map[string]string{
		"_index.md": `---
title: "My Book Library"
---

# Library

Digital book collection with interactive visualizations.

- [Books](/books/) - Browse collection
- [Timeline](/timeline/) - Chronological view
- [Geography](/geography/) - Geographic origins
`,
		"timeline.md": `---
title: "Timeline"
layout: "timeline"
---
`,
		"geography.md": `---
title: "Geographic View"
layout: "geography"
---
`,
		"books.md": `---
title: "Books"
layout: "books"
---
`,
	}

	for filename, content := range pages {
		path := filepath.Join(outputDir, "content", filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

func createLayouts(outputDir string) error {
	layouts := map[string]string{
		"baseof.html": `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }} - {{ .Site.Title }}</title>
    <link rel="stylesheet" href="/css/style.css">
    {{ block "head" . }}{{ end }}
</head>
<body>
    <header>
        <nav>
            <a href="/">Home</a>
            <a href="/books/">Books</a>
            <a href="/timeline/">Timeline</a>
            <a href="/geography/">Geography</a>
        </nav>
    </header>
    <main>
        {{ block "main" . }}{{ end }}
    </main>
    <footer>
        <p>&copy; {{ now.Year }} Library Collection</p>
    </footer>
    {{ block "scripts" . }}{{ end }}
</body>
</html>
`,
		"single.html": `{{ define "main" }}
<article>
    <h1>{{ .Title }}</h1>
    {{ .Content }}
</article>
{{ end }}
`,
		"timeline.html": `{{ define "head" }}
<script src="https://d3js.org/d3.v7.min.js"></script>
{{ end }}

{{ define "main" }}
<h1>{{ .Title }}</h1>
<div id="timeline"></div>
{{ end }}

{{ define "scripts" }}
<script src="/js/timeline.js"></script>
{{ end }}
`,
		"geography.html": `{{ define "head" }}
<link rel="stylesheet" href="https://unpkg.com/leaflet/dist/leaflet.css" />
<script src="https://unpkg.com/leaflet/dist/leaflet.js"></script>
{{ end }}

{{ define "main" }}
<h1>{{ .Title }}</h1>
<div id="map" style="height: 600px;"></div>
{{ end }}

{{ define "scripts" }}
<script src="/js/geography.js"></script>
{{ end }}
`,
		"books.html": `{{ define "main" }}
<h1>{{ .Title }}</h1>
<div id="books-container"></div>
{{ end }}

{{ define "scripts" }}
<script>
fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        if (!books || books.length === 0) {
            document.getElementById('books-container').innerHTML = '<p>No books found</p>';
            return;
        }
        
        const container = document.getElementById('books-container');
        const booksGrid = document.createElement('div');
        booksGrid.style.cssText = 'display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1.5rem; margin-top: 2rem;';
        
        books.forEach(book => {
            const bookCard = document.createElement('div');
            bookCard.style.cssText = 'background: var(--nord1); border: 1px solid var(--nord3); border-radius: 8px; padding: 1.5rem; transition: transform 0.2s ease, border-color 0.2s ease;';
            
            bookCard.innerHTML = '<div style="font-weight: 600; color: var(--nord6); margin-bottom: 0.5rem; font-size: 1.1rem;">' + 
                book.title + 
                '</div><div style="color: var(--nord8); margin-bottom: 0.5rem;">by ' + 
                book.author + 
                '</div>' +
                (book.original_date ? '<div style="color: var(--nord9); font-size: 0.9rem; margin-bottom: 0.5rem;">' + book.original_date + '</div>' : '') +
                (book.publisher ? '<div style="color: var(--nord4); font-size: 0.8rem; margin-bottom: 0.5rem;">' + book.publisher + '</div>' : '') +
                (book.description ? '<div style="color: var(--nord4); font-size: 0.9rem; line-height: 1.4; margin-top: 1rem;">' + book.description.substring(0, 200) + (book.description.length > 200 ? '...' : '') + '</div>' : '') +
                (book.llm_analysis && book.llm_analysis.genre ? '<div style="color: var(--nord7); font-size: 0.8rem; margin-top: 1rem;">' + book.llm_analysis.genre + '</div>' : '');
            
            bookCard.addEventListener('mouseenter', () => {
                bookCard.style.transform = 'translateY(-2px)';
                bookCard.style.borderColor = 'var(--nord8)';
            });
            
            bookCard.addEventListener('mouseleave', () => {
                bookCard.style.transform = 'translateY(0)';
                bookCard.style.borderColor = 'var(--nord3)';
            });
            
            booksGrid.appendChild(bookCard);
        });
        
        const statsDiv = document.createElement('div');
        statsDiv.style.cssText = 'background: var(--nord2); border-radius: 6px; padding: 1rem; margin-bottom: 2rem; font-family: monospace;';
        statsDiv.innerHTML = '<span style="color: var(--nord8);">Total Books:</span> <span style="color: var(--nord6);">' + books.length + '</span>';
        
        container.appendChild(statsDiv);
        container.appendChild(booksGrid);
    });
</script>
{{ end }}
`,
	}

	layoutDir := filepath.Join(outputDir, "layouts", "_default")
	for filename, content := range layouts {
		path := filepath.Join(layoutDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

func createStaticFiles(outputDir string, books []models.Book) error {
	css := `/* Nord Color Scheme */
:root {
    --nord0: #2e3440;   /* Polar Night */
    --nord1: #3b4252;
    --nord2: #434c5e;
    --nord3: #4c566a;
    --nord4: #d8dee9;   /* Snow Storm */
    --nord5: #e5e9f0;
    --nord6: #eceff4;
    --nord7: #8fbcbb;   /* Frost */
    --nord8: #88c0d0;
    --nord9: #81a1c1;
    --nord10: #5e81ac;
    --nord11: #bf616a;  /* Aurora */
    --nord12: #d08770;
    --nord13: #ebcb8b;
    --nord14: #a3be8c;
    --nord15: #b48ead;
}

* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: 'Fira Code', 'JetBrains Mono', 'Cascadia Code', monospace;
    line-height: 1.4;
    background-color: var(--nord0);
    color: var(--nord4);
    min-height: 100vh;
}

header {
    background: var(--nord1);
    border-bottom: 2px solid var(--nord3);
    padding: 0.75rem 0;
    position: sticky;
    top: 0;
    z-index: 100;
}

nav {
    max-width: 1400px;
    margin: 0 auto;
    padding: 0 1rem;
    display: flex;
    gap: 1.5rem;
}

nav a {
    color: var(--nord4);
    text-decoration: none;
    font-weight: 500;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    transition: all 0.2s ease;
    border: 1px solid transparent;
}

nav a:hover {
    background: var(--nord2);
    color: var(--nord8);
    border-color: var(--nord3);
}

nav a:focus {
    outline: 2px solid var(--nord9);
    outline-offset: 2px;
}

main {
    max-width: 1400px;
    margin: 0 auto;
    padding: 2rem 1rem;
}

h1, h2, h3 {
    color: var(--nord6);
    font-weight: 600;
    margin-bottom: 1rem;
}

h1 {
    font-size: 2.5rem;
    border-left: 4px solid var(--nord8);
    padding-left: 1rem;
    margin-bottom: 2rem;
}

h2 {
    font-size: 1.8rem;
    color: var(--nord8);
}

h3 {
    font-size: 1.4rem;
    color: var(--nord9);
}

p {
    margin-bottom: 1rem;
    color: var(--nord4);
}

ul {
    list-style: none;
    margin-left: 1rem;
}

li {
    margin-bottom: 0.5rem;
    position: relative;
}

li::before {
    content: "▶";
    color: var(--nord8);
    position: absolute;
    left: -1rem;
}

a {
    color: var(--nord8);
    text-decoration: none;
    transition: color 0.2s ease;
}

a:hover {
    color: var(--nord7);
    text-decoration: underline;
}

#timeline, #map {
    background: var(--nord1);
    border: 1px solid var(--nord3);
    border-radius: 8px;
    margin: 2rem 0;
    padding: 1rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

/* Timeline specific styles */
.bar {
    transition: all 0.2s ease;
}

.bar:hover {
    fill: var(--nord7) !important;
    stroke: var(--nord8);
    stroke-width: 2;
}

/* Map styles */
.leaflet-container {
    background: var(--nord1) !important;
    border-radius: 6px;
}

.leaflet-popup-content-wrapper {
    background: var(--nord2);
    color: var(--nord4);
    border-radius: 6px;
}

.leaflet-popup-tip {
    background: var(--nord2);
}

/* Footer */
footer {
    margin-top: 4rem;
    padding: 2rem 0;
    border-top: 1px solid var(--nord3);
    text-align: center;
    color: var(--nord3);
}

/* Scrollbar */
::-webkit-scrollbar {
    width: 8px;
}

::-webkit-scrollbar-track {
    background: var(--nord1);
}

::-webkit-scrollbar-thumb {
    background: var(--nord3);
    border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
    background: var(--nord8);
}

/* Selection */
::selection {
    background: var(--nord9);
    color: var(--nord0);
}
`

	timelineJS := `fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        if (!books || books.length === 0) return;
        
        const margin = {top: 20, right: 30, bottom: 60, left: 200};
        const width = Math.max(1000, window.innerWidth - 100) - margin.left - margin.right;
        const height = Math.max(600, books.length * 25) - margin.top - margin.bottom;

        d3.select("#timeline").selectAll("*").remove();
        
        const svg = d3.select("#timeline")
            .append("svg")
            .attr("width", width + margin.left + margin.right)
            .attr("height", height + margin.top + margin.bottom)
            .append("g")
            .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

        const validBooks = books.filter(d => d.original_date && d.original_date !== 0);
        const years = validBooks.map(d => d.original_date);
        const [minYear, maxYear] = d3.extent(years);

        const x = d3.scaleLinear()
            .domain([minYear - 50, maxYear + 50])
            .range([0, width]);

        const y = d3.scaleBand()
            .domain(validBooks.map(d => d.title))
            .range([0, height])
            .padding(0.1);

        // Add grid lines
        svg.append("g")
            .attr("class", "grid")
            .selectAll(".grid-line")
            .data(x.ticks(10))
            .enter()
            .append("line")
            .attr("class", "grid-line")
            .attr("x1", d => x(d))
            .attr("x2", d => x(d))
            .attr("y1", 0)
            .attr("y2", height)
            .attr("stroke", "var(--nord3)")
            .attr("stroke-width", 0.5)
            .attr("opacity", 0.3);

        // X axis
        svg.append("g")
            .attr("transform", "translate(0," + height + ")")
            .call(d3.axisBottom(x)
                .tickFormat(d3.format("d"))
                .tickSize(-height))
            .selectAll("text")
            .style("fill", "var(--nord4)")
            .style("font-family", "monospace");

        // Y axis
        svg.append("g")
            .call(d3.axisLeft(y).tickSize(0))
            .selectAll("text")
            .style("fill", "var(--nord4)")
            .style("font-family", "monospace")
            .style("font-size", "11px");

        // Bars
        svg.selectAll(".bar")
            .data(validBooks)
            .enter().append("rect")
            .attr("class", "bar")
            .attr("x", d => x(Math.min(0, d.original_date)))
            .attr("y", d => y(d.title))
            .attr("width", d => Math.abs(x(d.original_date) - x(0)))
            .attr("height", y.bandwidth())
            .attr("fill", "var(--nord8)")
            .attr("rx", 3)
            .on("mouseover", function(event, d) {
                d3.select(this)
                    .attr("fill", "var(--nord7)")
                    .attr("stroke", "var(--nord8)")
                    .attr("stroke-width", 2);
                
                // Tooltip
                const tooltip = d3.select("body").append("div")
                    .attr("class", "tooltip")
                    .style("position", "absolute")
                    .style("background", "var(--nord2)")
                    .style("color", "var(--nord4)")
                    .style("padding", "8px")
                    .style("border-radius", "4px")
                    .style("border", "1px solid var(--nord3)")
                    .style("font-family", "monospace")
                    .style("font-size", "12px")
                    .style("pointer-events", "none")
                    .style("z-index", "1000")
                    .html(d.title + "<br>" + d.author + "<br>" + d.original_date);
                    
                tooltip.style("left", (event.pageX + 10) + "px")
                       .style("top", (event.pageY - 10) + "px");
            })
            .on("mouseout", function() {
                d3.select(this)
                    .attr("fill", "var(--nord8)")
                    .attr("stroke", "none");
                d3.selectAll(".tooltip").remove();
            });
    });
`

	geographyJS := `const map = L.map('map', {
    zoomControl: true,
    attributionControl: false
}).setView([20, 0], 2);

// Dark theme tiles
L.tileLayer('https://tiles.stadiamaps.com/tiles/alidade_smooth_dark/{z}/{x}/{y}{r}.png', {
    attribution: 'Map data &copy; OpenStreetMap contributors'
}).addTo(map);

// Custom marker icon with Nord colors
const customIcon = L.divIcon({
    html: '<div style="background: var(--nord8); width: 12px; height: 12px; border-radius: 50%; border: 2px solid var(--nord0); box-shadow: 0 2px 4px rgba(0,0,0,0.5);"></div>',
    className: 'custom-marker',
    iconSize: [12, 12],
    iconAnchor: [6, 6]
});

fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        // Extract unique locations from books
        const locationMap = {};
        books.forEach(book => {
            if (book.original_location_name && book.original_location_latitude && book.original_location_longitude) {
                const key = book.original_location_name;
                if (!locationMap[key]) {
                    locationMap[key] = {
                        name: book.original_location_name,
                        lat: book.original_location_latitude,
                        lon: book.original_location_longitude,
                        books: []
                    };
                }
                locationMap[key].books.push(book.title);
            }
        });
        
        const locations = Object.values(locationMap);
        if (locations.length === 0) {
            document.getElementById('map').innerHTML = '<div style="color: var(--nord4); text-align: center; padding: 2rem;">No geographic data available</div>';
            return;
        }
        
        locations.forEach(location => {
            const marker = L.marker([location.lat, location.lon], {icon: customIcon})
                .addTo(map);
            
            const popupContent = '<div style="color: var(--nord4); font-family: monospace;"><strong style="color: var(--nord8);">' + 
                location.name + '</strong><br>' + 
                location.books.map(book => '<span style="color: var(--nord4);">' + book + '</span>').join('<br>') + 
                '</div>';
            
            marker.bindPopup(popupContent, {
                className: 'nord-popup'
            });
        });
        
        // Add custom CSS for popups
        const style = document.createElement('style');
        style.textContent = '.nord-popup .leaflet-popup-content-wrapper { background: var(--nord2); color: var(--nord4); border-radius: 6px; } .nord-popup .leaflet-popup-tip { background: var(--nord2); }';
        document.head.appendChild(style);
    });
`

	if err := os.WriteFile(filepath.Join(outputDir, "static", "css", "style.css"), []byte(css), 0644); err != nil {
		return fmt.Errorf("failed to write CSS: %w", err)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "static", "js", "timeline.js"), []byte(timelineJS), 0644); err != nil {
		return fmt.Errorf("failed to write timeline.js: %w", err)
	}

	if err := os.WriteFile(filepath.Join(outputDir, "static", "js", "geography.js"), []byte(geographyJS), 0644); err != nil {
		return fmt.Errorf("failed to write geography.js: %w", err)
	}

	// Save all books to a single books.json file
	booksJSON, _ := json.Marshal(books)
	if err := os.WriteFile(filepath.Join(outputDir, "static", "data", "books.json"), booksJSON, 0644); err != nil {
		return fmt.Errorf("failed to write books data: %w", err)
	}

	return nil
}