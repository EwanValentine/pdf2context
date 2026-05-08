package cleaner

import (
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"
)

var (
	reControl    = regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	reSpaces     = regexp.MustCompile(`[ \t]+`)
	reMultiNewln = regexp.MustCompile(`\n{3,}`)
)

// Clean applies a normalisation pipeline to the raw text extracted from a PDF.
func Clean(text string) string {
	text = removeHeadersFooters(text)
	text = norm.NFC.String(text)
	text = reControl.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "\f", "\n\n")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = reSpaces.ReplaceAllString(text, " ")
	text = reMultiNewln.ReplaceAllString(text, "\n\n")
	text = trimTrailingSpacePerLine(text)
	return strings.TrimSpace(text)
}

// removeHeadersFooters splits by form-feed (page break), identifies lines that
// appear in more than 50% of pages (length 3-100 chars), and removes them.
func removeHeadersFooters(text string) string {
	pages := strings.Split(text, "\f")
	if len(pages) < 2 {
		return text
	}

	// Count occurrences of each candidate line.
	lineCount := make(map[string]int)
	for _, page := range pages {
		seen := make(map[string]bool)
		for _, line := range strings.Split(page, "\n") {
			trimmed := strings.TrimSpace(line)
			l := len(trimmed)
			if l < 3 || l > 100 {
				continue
			}
			if !seen[trimmed] {
				lineCount[trimmed]++
				seen[trimmed] = true
			}
		}
	}

	threshold := len(pages) / 2
	if threshold < 1 {
		threshold = 1
	}

	// Build set of lines to remove.
	remove := make(map[string]bool)
	for line, count := range lineCount {
		if count > threshold {
			remove[line] = true
		}
	}

	if len(remove) == 0 {
		return text
	}

	// Remove those lines from each page.
	var cleanedPages []string
	for _, page := range pages {
		var kept []string
		for _, line := range strings.Split(page, "\n") {
			trimmed := strings.TrimSpace(line)
			if !remove[trimmed] {
				kept = append(kept, line)
			}
		}
		cleanedPages = append(cleanedPages, strings.Join(kept, "\n"))
	}

	return strings.Join(cleanedPages, "\f")
}

// trimTrailingSpacePerLine removes trailing whitespace from each line.
func trimTrailingSpacePerLine(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}
