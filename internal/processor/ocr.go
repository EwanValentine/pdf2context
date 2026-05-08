package processor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// OCRPDF runs ocrmypdf on the input PDF and writes the OCR'd result to a temp
// file. The caller is responsible for removing the temp file when done.
func OCRPDF(ctx context.Context, inputPath string) (outputPath string, err error) {
	tmpFile, err := os.CreateTemp("", "pdf2context-ocr-*.pdf")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	outputPath = tmpFile.Name()
	tmpFile.Close()

	cmd := exec.CommandContext(ctx, "ocrmypdf", "--skip-text", "--quiet", inputPath, outputPath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err = cmd.Run(); err != nil {
		os.Remove(outputPath)
		return "", fmt.Errorf("ocrmypdf failed: %w (stderr: %s)", err, stderr.String())
	}

	return outputPath, nil
}
