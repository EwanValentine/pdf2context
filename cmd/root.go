package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/EwanValentine/pdf2context/internal/config"
	"github.com/EwanValentine/pdf2context/internal/deps"
	"github.com/EwanValentine/pdf2context/internal/output"
	"github.com/EwanValentine/pdf2context/internal/processor"
	"github.com/EwanValentine/pdf2context/internal/scanner"
	"github.com/EwanValentine/pdf2context/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "pdf2context [directory]",
	Short: "Convert PDFs to chunked JSONL context files",
	Long: `pdf2context recursively scans a directory for PDF files, extracts their
text (using OCR when necessary), cleans the output, splits it into overlapping
chunks, and writes per-file JSONL records plus an optional combined file and
a processing manifest.`,
	Args: cobra.ExactArgs(1),
	RunE: runRoot,
}

// flags
var (
	flagWorkers    int
	flagChunkSize  int
	flagOverlap    int
	flagOCRTimeout time.Duration
	flagVerbose    bool
	flagNoCombined bool
	flagMinWords   int
)

func init() {
	rootCmd.Flags().IntVarP(&flagWorkers, "workers", "w", 4, "number of parallel workers")
	rootCmd.Flags().IntVar(&flagChunkSize, "chunk-size", 1200, "words per chunk")
	rootCmd.Flags().IntVar(&flagOverlap, "overlap", 100, "word overlap between chunks")
	rootCmd.Flags().DurationVar(&flagOCRTimeout, "ocr-timeout", 10*time.Minute, "timeout per OCR operation")
	rootCmd.Flags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose output")
	rootCmd.Flags().BoolVar(&flagNoCombined, "no-combined", false, "skip writing combined.context.jsonl")
	rootCmd.Flags().IntVar(&flagMinWords, "min-words", 50, "trigger OCR when extracted word count is below this")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) error {
	// 1. Resolve absolute path.
	dir, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// 2. Check dependencies.
	if err := deps.Check(); err != nil {
		fmt.Fprintln(os.Stderr, "Error: "+err.Error())
		fmt.Fprintln(os.Stderr, "Please install the missing tools and try again.")
		os.Exit(1)
	}

	// 3. Scan for PDFs.
	pdfs, err := scanner.Scan(dir)
	if err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}
	if len(pdfs) == 0 {
		fmt.Fprintf(os.Stderr, "No PDF files found in %s\n", dir)
		os.Exit(0)
	}

	cfg := config.Config{
		InputDir:   dir,
		Workers:    flagWorkers,
		ChunkSize:  flagChunkSize,
		Overlap:    flagOverlap,
		OCRTimeout: flagOCRTimeout,
		Verbose:    flagVerbose,
		NoCombined: flagNoCombined,
		MinWords:   flagMinWords,
	}

	// 4. Create combined writer (unless disabled).
	var combined *output.CombinedWriter
	if !flagNoCombined {
		combined, err = output.NewCombinedWriter(dir)
		if err != nil {
			return fmt.Errorf("creating combined writer: %w", err)
		}
		defer combined.Close()
	}

	// 5. Build TUI model.
	model := ui.NewModel(dir, len(pdfs), flagWorkers)

	// 6. Create Bubble Tea program.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// 7. Launch processor in background goroutine.
	go processor.Run(cfg, pdfs, p, combined)

	// 8. Run TUI (blocks until done).
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
