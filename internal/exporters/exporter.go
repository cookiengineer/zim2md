// Package exporters writes Markdown and optional asset files to disk and
// records a run report.
package exporters

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ErrSkipExisting is returned when a file exists and --no-clobber is set.
var ErrSkipExisting = errors.New("file already exists")

// FileExporter writes files below a fixed output root using atomic renames.
type FileExporter struct {
	root      string
	noClobber bool

	mu      sync.Mutex
	written map[string]string
}

// NewFileExporter creates an exporter rooted at root.
func NewFileExporter(root string, noClobber bool) *FileExporter {
	return &FileExporter{
		root:      root,
		noClobber: noClobber,
		written:   make(map[string]string),
	}
}

// Root returns the output root directory.
func (e *FileExporter) Root() string {
	return e.root
}

// Write writes data to a slash separated path below the root.
func (e *FileExporter) Write(relativePath string, data []byte) error {
	cleaned, err := e.cleanRelativePath(relativePath)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(e.root, cleaned)
	directory := filepath.Dir(fullPath)

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("creating directory %q: %w", directory, err)
	}

	e.mu.Lock()
	_, alreadyWritten := e.written[cleaned]
	e.mu.Unlock()

	if e.noClobber {
		if alreadyWritten {
			return ErrSkipExisting
		}
		if _, err := os.Stat(fullPath); err == nil {
			return ErrSkipExisting
		}
	}

	tempFile, err := os.CreateTemp(directory, ".zim2md-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tempPath := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("writing %q: %w", cleaned, err)
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("closing %q: %w", cleaned, err)
	}
	if err := os.Chmod(tempPath, 0o644); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("setting permissions on %q: %w", cleaned, err)
	}
	if err := os.Rename(tempPath, fullPath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("renaming into place %q: %w", cleaned, err)
	}

	e.mu.Lock()
	e.written[cleaned] = relativePath
	e.mu.Unlock()

	return nil
}

func (e *FileExporter) cleanRelativePath(relativePath string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(relativePath))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("empty output path")
	}
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute output path %q is not allowed", relativePath)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("output path %q escapes the output root", relativePath)
	}
	return cleaned, nil
}
