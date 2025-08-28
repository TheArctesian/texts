package ocr

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SimpleTesseractOCR struct {
	language string
}

func NewTesseractOCR() *SimpleTesseractOCR {
	return &SimpleTesseractOCR{
		language: "eng",
	}
}

func (t *SimpleTesseractOCR) Close() {
	// Nothing to close for this implementation
}

func (t *SimpleTesseractOCR) ExtractText(imagePath string) (string, error) {
	// Check if tesseract is available
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", fmt.Errorf("tesseract not found in PATH: %w", err)
	}

	// Create temporary output file
	tempDir := os.TempDir()
	outputBase := filepath.Join(tempDir, "tesseract_output")
	outputFile := outputBase + ".txt"
	
	// Clean up temporary files
	defer os.Remove(outputFile)

	// Run tesseract command
	cmd := exec.Command("tesseract", imagePath, outputBase, "-l", t.language)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract command failed: %w", err)
	}

	// Read output file
	content, err := os.ReadFile(outputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read tesseract output: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// ExtractBookInfo attempts to extract book title and author from OCR text
func (t *SimpleTesseractOCR) ExtractBookInfo(text string) (title, author string) {
	lines := strings.Split(text, "\n")
	var nonEmptyLines []string
	
	// Filter out empty lines and clean up
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 2 { // Ignore very short lines
			// Clean up common OCR artifacts more carefully
			line = cleanOCRArtifacts(line)
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}

	if len(nonEmptyLines) == 0 {
		return "", ""
	}

	// Heuristic: First substantial line is likely the title
	title = nonEmptyLines[0]

	// Look for author patterns in subsequent lines
	for i := 1; i < len(nonEmptyLines) && i < 5; i++ {
		line := strings.ToLower(nonEmptyLines[i])
		// Common author indicators
		if strings.Contains(line, "by ") || 
		   strings.Contains(line, "author") ||
		   strings.Contains(line, "written") {
			// Clean up the author line
			author = strings.ReplaceAll(nonEmptyLines[i], "by ", "")
			author = strings.ReplaceAll(author, "By ", "")
			author = strings.TrimSpace(author)
			break
		}
	}

	// If no explicit author found, second line might be author
	if author == "" && len(nonEmptyLines) > 1 {
		// Check if second line looks like an author name
		secondLine := strings.ToLower(nonEmptyLines[1])
		if !strings.Contains(secondLine, "edition") && 
		   !strings.Contains(secondLine, "volume") &&
		   !strings.Contains(secondLine, "part") &&
		   len(nonEmptyLines[1]) < 50 { // Authors names are usually shorter
			author = nonEmptyLines[1]
		}
	}

	return strings.TrimSpace(title), strings.TrimSpace(author)
}

// cleanOCRArtifacts carefully cleans common OCR errors without being too aggressive
func cleanOCRArtifacts(text string) string {
	// Only replace "|" with "I" if it's likely a misread letter (surrounded by letters)
	if strings.Contains(text, "|") {
		// Replace | with I only if it's between word characters
		text = strings.ReplaceAll(text, "l|", "lI")  // common pattern
		text = strings.ReplaceAll(text, "|l", "Il")  // common pattern
		text = strings.ReplaceAll(text, "I|", "II")  // double I
		text = strings.ReplaceAll(text, "|I", "II")  // double I
		// Replace isolated | with I
		words := strings.Fields(text)
		for i, word := range words {
			if word == "|" {
				words[i] = "I"
			}
		}
		text = strings.Join(words, " ")
	}
	
	// Only replace "0" with "O" in contexts where it's clearly a letter
	// Look for 0 surrounded by letters or at word boundaries
	words := strings.Fields(text)
	for i, word := range words {
		// Replace 0 with O if the word contains other letters and 0 is not at the end (likely not a number)
		if len(word) > 1 && strings.ContainsAny(word, "ABCDEFGHIJKLMNPQRSTUVWXYZabcdefghijklmnpqrstuvwxyz") {
			// Only replace 0 with O if it's clearly in a word context
			if strings.HasPrefix(word, "0") && len(word) > 1 {
				words[i] = "O" + word[1:]
			}
			// Middle positions - be very careful
			for j := 1; j < len(word)-1; j++ {
				if word[j] == '0' && 
					((word[j-1] >= 'A' && word[j-1] <= 'Z') || (word[j-1] >= 'a' && word[j-1] <= 'z')) &&
					((word[j+1] >= 'A' && word[j+1] <= 'Z') || (word[j+1] >= 'a' && word[j+1] <= 'z')) {
					runes := []rune(word)
					runes[j] = 'O'
					words[i] = string(runes)
					break
				}
			}
		}
	}
	
	return strings.Join(words, " ")
}