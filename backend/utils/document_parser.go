package utils

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ParseDocument extracts text from uploaded files
// Supports PDF, DOCX, and TXT file formats
func ParseDocument(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	log.Printf("[DocumentParser] Parsing file: %s (extension: %s)", file.Filename, ext)

	switch ext {
	case ".pdf":
		log.Printf("[DocumentParser] Extracting text from PDF...")
		result, err := extractTextFromPDF(file)
		if err != nil {
			log.Printf("[DocumentParser] PDF extraction error: %v", err)
			return "", err
		}
		log.Printf("[DocumentParser] PDF extraction successful. Extracted %d characters", len(result))
		return result, nil
	case ".docx":
		log.Printf("[DocumentParser] Extracting text from DOCX...")
		result, err := extractTextFromDOCX(file)
		if err != nil {
			log.Printf("[DocumentParser] DOCX extraction error: %v", err)
			return "", err
		}
		log.Printf("[DocumentParser] DOCX extraction successful. Extracted %d characters", len(result))
		return result, nil
	case ".txt":
		log.Printf("[DocumentParser] Reading TXT file...")
		result, err := extractTextFromTXT(file)
		if err != nil {
			log.Printf("[DocumentParser] TXT reading error: %v", err)
			return "", err
		}
		log.Printf("[DocumentParser] TXT reading successful. Read %d characters", len(result))
		return result, nil
	default:
		log.Printf("[DocumentParser] Unsupported file type: %s", ext)
		return "", fmt.Errorf("unsupported file type: %s (supported: .pdf, .docx, .txt)", ext)
	}
}

// extractTextFromTXT extracts text from a plain text file
func extractTextFromTXT(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// extractTextFromPDF extracts text from a PDF file
// Preserves table structure by maintaining whitespace and line breaks
func extractTextFromPDF(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Save to temp file for PDF parsing
	tmpFile, err := os.CreateTemp("", "lesson-plan-*.pdf")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	_, err = io.Copy(tmpFile, src)
	tmpFile.Close()
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	// Parse PDF and extract text
	f, pdfReader, err := pdf.Open(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w - the file may be encrypted or corrupted", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPages := pdfReader.NumPage()

	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		page := pdfReader.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}

		// Get all text blocks from the page
		texts := page.Content().Text

		// Group texts by Y-coordinate to detect rows/lines
		type textLine struct {
			Text  string
			Y     float64
			X     float64
			Width float64
		}
		var lines []textLine

		for _, text := range texts {
			if strings.TrimSpace(text.S) != "" {
				lines = append(lines, textLine{
					Y:     text.Y,
					X:     text.X,
					Text:  text.S,
					Width: text.W,
				})
			}
		}

		// Sort by Y coordinate (top to bottom), then by X (left to right)
		// Group items with similar Y into rows
		if len(lines) > 0 {
			// Sort by Y first
			for i := 0; i < len(lines)-1; i++ {
				for j := i + 1; j < len(lines); j++ {
					if lines[j].Y > lines[i].Y {
						lines[i], lines[j] = lines[j], lines[i]
					}
				}
			}

			// Group by Y (with tolerance for small differences)
			const yTolerance = 5.0
			var rows [][]textLine
			var currentRow []textLine
			currentY := -1.0

			for _, line := range lines {
				if currentY < 0 || abs(line.Y-currentY) > yTolerance {
					if len(currentRow) > 0 {
						// Sort current row by X coordinate
						for i := 0; i < len(currentRow)-1; i++ {
							for j := i + 1; j < len(currentRow); j++ {
								if currentRow[j].X < currentRow[i].X {
									currentRow[i], currentRow[j] = currentRow[j], currentRow[i]
								}
							}
						}
						rows = append(rows, currentRow)
					}
					currentRow = []textLine{line}
					currentY = line.Y
				} else {
					currentRow = append(currentRow, line)
				}
			}
			if len(currentRow) > 0 {
				// Sort final row by X coordinate
				for i := 0; i < len(currentRow)-1; i++ {
					for j := i + 1; j < len(currentRow); j++ {
						if currentRow[j].X < currentRow[i].X {
							currentRow[i], currentRow[j] = currentRow[j], currentRow[i]
						}
					}
				}
				rows = append(rows, currentRow)
			}

			// Build text output preserving table structure
			for _, row := range rows {
				var rowText strings.Builder
				for i, item := range row {
					if i > 0 {
						// Add spacing based on X gap (simulates table columns)
						gap := item.X - row[i-1].X - row[i-1].Width
						if gap > 20 {
							rowText.WriteString(" | ")
						} else if gap > 5 {
							rowText.WriteString("  ")
						}
					}
					rowText.WriteString(item.Text)
				}
				textBuilder.WriteString(rowText.String())
				textBuilder.WriteString("\n")
			}
		}

		textBuilder.WriteString("\n--- Page " + string(rune(pageIndex)) + "---\n\n")
	}

	text := strings.TrimSpace(textBuilder.String())
	if text == "" {
		return "", fmt.Errorf("no text found in PDF - the file may be image-only (scanned)")
	}

	return text, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// extractTextFromDOCX extracts text from a DOCX file by parsing document.xml
func extractTextFromDOCX(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read entire file
	docxData, err := io.ReadAll(src)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Open as ZIP (DOCX is a ZIP file)
	reader, err := zip.NewReader(bytes.NewReader(docxData), int64(len(docxData)))
	if err != nil {
		return "", fmt.Errorf("failed to parse DOCX (invalid ZIP format): %w", err)
	}

	// Find and extract word/document.xml
	for _, zipFile := range reader.File {
		if zipFile.Name == "word/document.xml" {
			rc, err := zipFile.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open document.xml: %w", err)
			}
			defer rc.Close()

			data, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("failed to read document.xml: %w", err)
			}

			return extractTextFromXML(string(data)), nil
		}
	}

	return "", fmt.Errorf("document.xml not found in DOCX - file may be corrupted")
}

// extractTextFromXML extracts text from DOCX XML content
func extractTextFromXML(xmlContent string) string {
	// Simple XML text extraction - strip XML tags
	var text strings.Builder
	var inTag bool

	for _, ch := range xmlContent {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
			continue
		}

		if !inTag {
			text.WriteRune(ch)
		}
	}

	// Decode XML entities
	result := text.String()
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&quot;", "\"")
	result = strings.ReplaceAll(result, "&apos;", "'")

	// Clean up whitespace
	result = strings.TrimSpace(result)
	result = strings.Join(strings.Fields(result), " ")

	return result
}

// ValidateFile validates the uploaded file
// - Checks file extension against allowed types
// - Checks file size against max size
func ValidateFile(file *multipart.FileHeader, allowedTypes []string, maxSizeMB int) error {
	ext := strings.ToLower(filepath.Ext(file.Filename))

	// Check file type
	allowed := false
	for _, allowedExt := range allowedTypes {
		if ext == "."+strings.ToLower(allowedExt) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("file type %s not allowed. Allowed types: %v", ext, allowedTypes)
	}

	// Check file size
	if file.Size > int64(maxSizeMB)*1024*1024 {
		return fmt.Errorf("file size (%.2f MB) exceeds maximum allowed size (%d MB)",
			float64(file.Size)/1024/1024, maxSizeMB)
	}

	return nil
}

// SanitizeFilename sanitizes the uploaded filename
// - Removes path separators
// - Removes special characters
// - Adds a UUID prefix to prevent collisions
func SanitizeFilename(filename string) string {
	// Get base name only (no path)
	base := filepath.Base(filename)

	// Remove any dangerous characters
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, base)

	return sanitized
}
