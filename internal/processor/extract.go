package processor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ExtractResult holds the extracted text and word count from a PDF.
type ExtractResult struct {
	Text      string
	WordCount int
}

// ExtractText runs pdftotext -layout on the given PDF and returns the text.
func ExtractText(ctx context.Context, pdfPath string) (ExtractResult, error) {
	cmd := exec.CommandContext(ctx, "pdftotext", "-layout", pdfPath, "-")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return ExtractResult{}, fmt.Errorf("pdftotext failed: %w (stderr: %s)", err, stderr.String())
	}

	text := stdout.String()
	words := strings.Fields(text)

	return ExtractResult{
		Text:      text,
		WordCount: len(words),
	}, nil
}
