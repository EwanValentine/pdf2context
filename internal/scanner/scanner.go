package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// Scan walks dir recursively and returns all PDF file paths (case-insensitive).
func Scan(dir string) ([]string, error) {
	var pdfs []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".pdf" {
			pdfs = append(pdfs, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pdfs, nil
}
