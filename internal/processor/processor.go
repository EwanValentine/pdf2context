package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/EwanValentine/pdf2context/internal/cleaner"
	"github.com/EwanValentine/pdf2context/internal/config"
	"github.com/EwanValentine/pdf2context/internal/output"
)

// --- Message types sent to the Bubble Tea program ---

// MsgWorkerStarted is sent when a worker starts processing a file.
type MsgWorkerStarted struct {
	WorkerID int
	Path     string
}

// MsgWorkerExtracting is sent when a worker begins text extraction.
type MsgWorkerExtracting struct{ WorkerID int }

// MsgWorkerOCR is sent when a worker is about to run OCR.
type MsgWorkerOCR struct{ WorkerID int }

// MsgWorkerChunking is sent when a worker is chunking extracted text.
type MsgWorkerChunking struct{ WorkerID int }

// MsgWorkerDone is sent when a worker finishes (success or failure).
type MsgWorkerDone struct {
	WorkerID int
	Result   Result
}

// MsgAllDone is sent after all PDFs have been processed.
type MsgAllDone struct {
	ManifestPath string
	CombinedPath string
	Duration     time.Duration
}

// Result holds the outcome of processing a single PDF.
type Result struct {
	Path     string
	Chunks   int
	Duration time.Duration
	OCRd     bool
	Err      error
}

// Run orchestrates concurrent PDF processing using a worker pool.
func Run(cfg config.Config, pdfs []string, program *tea.Program, combined *output.CombinedWriter) {
	startTime := time.Now()

	sem := make(chan int, cfg.Workers)
	for i := 0; i < cfg.Workers; i++ {
		sem <- i
	}

	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		failedFiles []output.FailedFile
		totalChunks int
		ocrdCount   int
		processedOK int
	)

	for _, pdfPath := range pdfs {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			workerID := <-sem
			defer func() { sem <- workerID }()

			program.Send(MsgWorkerStarted{WorkerID: workerID, Path: path})

			result, records := processPDF(cfg, path, workerID, program)

			if result.Err == nil && combined != nil {
				if err := combined.Write(records); err != nil {
					result.Err = fmt.Errorf("writing to combined: %w", err)
				}
			}

			mu.Lock()
			if result.Err != nil {
				failedFiles = append(failedFiles, output.FailedFile{
					Path:  path,
					Error: result.Err.Error(),
				})
			} else {
				processedOK++
				totalChunks += result.Chunks
				if result.OCRd {
					ocrdCount++
				}
			}
			mu.Unlock()

			program.Send(MsgWorkerDone{WorkerID: workerID, Result: result})
		}(pdfPath)
	}

	wg.Wait()

	duration := time.Since(startTime)

	manifest := output.Manifest{
		GeneratedAt: time.Now(),
		InputDir:    cfg.InputDir,
		TotalPDFs:   len(pdfs),
		ProcessedOK: processedOK,
		TotalChunks: totalChunks,
		FailedFiles: failedFiles,
		OCRStats:    output.OCRStats{FilesOCRd: ocrdCount},
		ProcessingMs: duration.Milliseconds(),
	}

	manifestPath, _ := output.WriteManifest(cfg.InputDir, manifest)
	combinedPath := ""
	if combined != nil {
		combinedPath = combined.Path()
	}

	program.Send(MsgAllDone{
		ManifestPath: manifestPath,
		CombinedPath: combinedPath,
		Duration:     duration,
	})
}

// processPDF processes a single PDF file, returning its Result and chunk records.
func processPDF(
	cfg config.Config,
	path string,
	workerID int,
	program *tea.Program,
) (result Result, records []output.ChunkRecord) {
	start := time.Now()
	result.Path = path

	defer func() {
		if r := recover(); r != nil {
			result.Err = fmt.Errorf("panic: %v", r)
			result.Duration = time.Since(start)
		}
	}()

	ctx := context.Background()

	// Step 1: Extract text
	program.Send(MsgWorkerExtracting{WorkerID: workerID})
	extracted, err := ExtractText(ctx, path)
	if err != nil {
		result.Err = err
		result.Duration = time.Since(start)
		return
	}

	// Step 2: OCR if needed
	if extracted.WordCount < cfg.MinWords {
		program.Send(MsgWorkerOCR{WorkerID: workerID})

		ocrCtx, cancel := context.WithTimeout(ctx, cfg.OCRTimeout)
		defer cancel()

		ocrPath, ocrErr := OCRPDF(ocrCtx, path)
		if ocrErr != nil {
			result.Err = ocrErr
			result.Duration = time.Since(start)
			return
		}
		defer os.Remove(ocrPath)

		// Re-extract from OCR'd PDF
		extracted, err = ExtractText(ctx, ocrPath)
		if err != nil {
			result.Err = err
			result.Duration = time.Since(start)
			return
		}
		result.OCRd = true
	}

	// Step 3: Clean text
	cleaned := cleaner.Clean(extracted.Text)

	// Step 4: Chunk
	program.Send(MsgWorkerChunking{WorkerID: workerID})
	chunks := ChunkText(cleaned, cfg.ChunkSize, cfg.Overlap)

	// Step 5: Build records
	source := filepath.Base(path)
	for _, c := range chunks {
		records = append(records, output.ChunkRecord{
			Source:    source,
			Path:      path,
			Chunk:     c.Index,
			Text:      c.Text,
			WordCount: c.WordCount,
		})
	}

	// Step 6: Write per-PDF JSONL
	if _, err := output.WriteJSONL(path, records); err != nil {
		result.Err = err
		result.Duration = time.Since(start)
		return
	}

	result.Chunks = len(chunks)
	result.Duration = time.Since(start)
	return
}
