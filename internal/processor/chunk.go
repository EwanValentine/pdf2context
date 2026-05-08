package processor

import (
	"strings"
)

// Chunk represents a single text chunk from a document.
type Chunk struct {
	Index     int
	Text      string
	WordCount int
}

// ChunkText splits text into overlapping chunks of chunkSize words with the
// given overlap. Step = chunkSize - overlap.
func ChunkText(text string, chunkSize, overlap int) []Chunk {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	step := chunkSize - overlap
	if step <= 0 {
		step = 1
	}

	var chunks []Chunk
	index := 1

	for start := 0; start < len(words); start += step {
		end := start + chunkSize
		if end > len(words) {
			end = len(words)
		}

		chunkWords := words[start:end]
		chunks = append(chunks, Chunk{
			Index:     index,
			Text:      strings.Join(chunkWords, " "),
			WordCount: len(chunkWords),
		})
		index++

		if end == len(words) {
			break
		}
	}

	return chunks
}
