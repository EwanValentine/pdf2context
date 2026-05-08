# pdf2context

<a href="https://www.buymeacoffee.com/TWwJh0e"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" height="50" /></a>

A production-quality CLI tool that converts directories of PDF files into
chunked JSONL context files suitable for use with large language models and
retrieval-augmented generation (RAG) pipelines.

## Features

- Recursive PDF discovery
- Text extraction via `pdftotext` (preserves layout)
- Automatic OCR fallback via `ocrmypdf` + Tesseract when text yield is low
- Text cleaning: header/footer removal, Unicode normalisation, control-char stripping
- Configurable overlapping word chunks
- Per-file `.context.jsonl` output + optional combined file
- Processing manifest (`manifest.json`) with full statistics
- Live Bubble Tea TUI with worker status, progress bar, ETA, and recent log

---

## Requirements

### macOS

```bash
brew install poppler ocrmypdf tesseract
```

### Linux (Debian/Ubuntu)

```bash
apt-get install poppler-utils ocrmypdf tesseract-ocr
```

---

## Build

```bash
# Using make
make

# Or directly
go build -o pdf2context .
```

### Install to $GOPATH/bin

```bash
make install
# or
go install .
```

---

## Usage

```bash
# Process all PDFs in a directory with defaults
pdf2context /path/to/pdfs

# Use 8 workers, smaller chunks, no combined output
pdf2context /path/to/pdfs --workers 8 --chunk-size 800 --overlap 80 --no-combined

# Increase OCR timeout for large scanned documents
pdf2context /path/to/pdfs --ocr-timeout 20m

# Verbose mode
pdf2context /path/to/pdfs --verbose
```

---

## Flags

| Flag            | Default | Description                                          |
|-----------------|---------|------------------------------------------------------|
| `--workers`     | `4`     | Number of parallel worker goroutines                 |
| `--chunk-size`  | `1200`  | Words per chunk                                      |
| `--overlap`     | `100`   | Word overlap between consecutive chunks              |
| `--ocr-timeout` | `10m`   | Per-file OCR timeout                                 |
| `--min-words`   | `50`    | Trigger OCR when extracted word count is below this  |
| `--no-combined` | `false` | Skip writing `combined.context.jsonl`                |
| `--verbose`     | `false` | Verbose output                                       |

---

## Output Format

### Per-file: `<original>.pdf.context.jsonl`

Each line is a JSON object:

```json
{
  "source": "quarterly_report.pdf",
  "path": "/path/to/quarterly_report.pdf",
  "chunk": 0,
  "text": "This is the first chunk of extracted text ...",
  "word_count": 1200
}
```

### Combined: `combined.context.jsonl`

All records from all PDFs concatenated into a single JSONL file in the input
directory.

### Manifest: `manifest.json`

```json
{
  "generated_at": "2024-01-15T10:30:00Z",
  "input_dir": "/path/to/pdfs",
  "total_pdfs": 46,
  "processed_ok": 44,
  "total_chunks": 312,
  "failed_files": [
    {
      "path": "/path/to/corrupted.pdf",
      "error": "pdftotext failed: exit status 1"
    }
  ],
  "ocr_stats": {
    "files_ocrd": 8
  },
  "processing_ms": 187432
}
```
