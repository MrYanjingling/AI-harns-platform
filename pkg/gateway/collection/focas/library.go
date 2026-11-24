package focas

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrLibraryMissing  = fmt.Errorf("FOCAS library is missing")
	ErrLibraryNotWired = fmt.Errorf("FOCAS library is present but the cgo API adapter is not wired")
)

var libraryCandidates = []string{
	"libfocas32.so.1",
	"libfocas32.so",
	"libfwlib32-linux-x64.so.1.0.5",
	"libfwlib32.so.1",
}

func findLibrary() (string, error) {
	if configured := os.Getenv("FOCAS_LIBRARY_PATH"); configured != "" {
		if info, err := os.Stat(configured); err == nil && !info.IsDir() {
			return configured, nil
		}
	}
	for _, name := range libraryCandidates {
		path := filepath.Join("plugins", "dependencies", "focas", name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path, nil
		}
	}
	return "", ErrLibraryMissing
}
