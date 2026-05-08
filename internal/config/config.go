package config

import "time"

// Config holds all runtime configuration for pdf2context.
type Config struct {
	InputDir   string
	Workers    int
	ChunkSize  int
	Overlap    int
	OCRTimeout time.Duration
	Verbose    bool
	NoCombined bool
	MinWords   int // below this → trigger OCR (default 50)
}
