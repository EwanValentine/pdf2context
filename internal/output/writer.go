package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChunkRecord is the JSONL record written for each text chunk.
type ChunkRecord struct {
	Source    string `json:"source"`
	Path      string `json:"path"`
	Chunk     int    `json:"chunk"`
	Text      string `json:"text"`
	WordCount int    `json:"word_count"`
}

// FailedFile records a PDF that could not be processed.
type FailedFile struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// OCRStats holds statistics about OCR processing.
type OCRStats struct {
	FilesOCRd int `json:"files_ocrd"`
}

// Manifest is the summary written after all PDFs are processed.
type Manifest struct {
	GeneratedAt  time.Time    `json:"generated_at"`
	InputDir     string       `json:"input_dir"`
	TotalPDFs    int          `json:"total_pdfs"`
	ProcessedOK  int          `json:"processed_ok"`
	TotalChunks  int          `json:"total_chunks"`
	FailedFiles  []FailedFile `json:"failed_files"`
	OCRStats     OCRStats     `json:"ocr_stats"`
	ProcessingMs int64        `json:"processing_ms"`
}

// WriteJSONL writes chunk records as JSONL to <pdfPath>.context.jsonl.
// It returns the output path.
func WriteJSONL(pdfPath string, records []ChunkRecord) (string, error) {
	outPath := pdfPath + ".context.jsonl"
	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("creating JSONL file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, r := range records {
		if err := enc.Encode(r); err != nil {
			return "", fmt.Errorf("encoding chunk record: %w", err)
		}
	}
	return outPath, nil
}

// CombinedWriter writes all chunks to a single combined.context.jsonl file.
// It is safe for concurrent use.
type CombinedWriter struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
	path string
}

// NewCombinedWriter creates a new CombinedWriter at dir/combined.context.jsonl.
func NewCombinedWriter(dir string) (*CombinedWriter, error) {
	path := filepath.Join(dir, "combined.context.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating combined JSONL: %w", err)
	}
	return &CombinedWriter{
		file: f,
		enc:  json.NewEncoder(f),
		path: path,
	}, nil
}

// Path returns the output file path.
func (cw *CombinedWriter) Path() string {
	if cw == nil {
		return ""
	}
	return cw.path
}

// Write appends the given records to the combined JSONL file.
func (cw *CombinedWriter) Write(records []ChunkRecord) error {
	if cw == nil {
		return nil
	}
	cw.mu.Lock()
	defer cw.mu.Unlock()
	for _, r := range records {
		if err := cw.enc.Encode(r); err != nil {
			return fmt.Errorf("encoding record to combined: %w", err)
		}
	}
	return nil
}

// Close closes the underlying file.
func (cw *CombinedWriter) Close() error {
	if cw == nil {
		return nil
	}
	return cw.file.Close()
}

// WriteManifest writes the manifest as indented JSON to dir/manifest.json.
// It returns the output path.
func WriteManifest(dir string, m Manifest) (string, error) {
	path := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("writing manifest: %w", err)
	}
	return path, nil
}
