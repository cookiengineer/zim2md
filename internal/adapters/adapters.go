// Package adapters contains the boundary between zim2md and the OpenZIM
// archive library. It is the only package that imports
// github.com/cookiengineer/gozim, so the rest of the program deals with plain
// Go values and stays testable without a real archive.
package adapters

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cookiengineer/gozim/archive/zim"
)

const (
	mimeTypeHTML       = "text/html"
	mimeTypeXHTML      = "application/xhtml+xml"
	mimeTypeRedirect   = "redirect"
	mimeTypeLinkTarget = "linktarget"
	mimeTypeDeleted    = "deleted"
)

// Entry describes one usable entry of a ZIM archive.
type Entry struct {
	// Index is the archive-local entry index, stable for the archive lifetime.
	Index uint32
	// Path is the raw URL path as stored in the archive.
	Path string
	// Title is the human readable entry title.
	Title string
	// MimeType is the resolved MIME type string.
	MimeType string
	// IsRedirect is true when the entry forwards to RedirectTarget.
	IsRedirect bool
	// RedirectTarget is the raw path of the redirect target (if IsRedirect).
	RedirectTarget string
}

// IsHTML reports whether this entry should be converted into Markdown.
func (e Entry) IsHTML() bool {
	return !e.IsRedirect && (e.MimeType == mimeTypeHTML || e.MimeType == mimeTypeXHTML)
}

// ArchiveSource owns an opened OpenZIM archive.
type ArchiveSource struct {
	filePath string
	archive  *zim.Archive
	entries  []Entry
}

// OpenArchiveSource opens the archive at filePath and reads its index.
// The returned source must be closed by the caller.
func OpenArchiveSource(filePath string) (*ArchiveSource, error) {
	archive, err := zim.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening archive %q: %w", filePath, err)
	}

	source := &ArchiveSource{
		filePath: filePath,
		archive:  archive,
	}

	if err := source.readIndex(); err != nil {
		_ = archive.Close()
		return nil, err
	}

	return source, nil
}

// Close releases the archive resources.
func (s *ArchiveSource) Close() error {
	if s.archive == nil {
		return nil
	}
	return s.archive.Close()
}

// FilePath returns the path the archive was opened from.
func (s *ArchiveSource) FilePath() string {
	return s.filePath
}

// OutputRootName returns the directory name used below the output root.
// For "/data/example-123.zim" it returns "example-123".
func (s *ArchiveSource) OutputRootName() string {
	name := filepath.Base(s.filePath)
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".zim") {
		name = name[:len(name)-len(".zim")]
	}
	if name == "" {
		name = "archive"
	}
	return name
}

// EntryCount returns the number of indexed entries held for processing.
func (s *ArchiveSource) EntryCount() int {
	return len(s.entries)
}

// Entries returns every indexed entry (HTML pages, assets and redirects).
// Metadata and index namespaces are excluded.
func (s *ArchiveSource) Entries() []Entry {
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// IsNewNamespaceScheme reports which namespace scheme the archive uses.
func (s *ArchiveSource) IsNewNamespaceScheme() bool {
	return s.archive.HasNewNamespaceScheme()
}

// Metadata returns an archive metadata value by key.
func (s *ArchiveSource) Metadata(key string) (string, bool) {
	return s.archive.Metadata(key)
}

// ReadBytes returns the decompressed payload of an entry.
func (s *ArchiveSource) ReadBytes(entry Entry) ([]byte, error) {
	if entry.IsRedirect {
		return nil, fmt.Errorf("entry %q is a redirect", entry.Path)
	}

	zimEntry, err := s.archive.EntryByIndex(entry.Index)
	if err != nil {
		return nil, fmt.Errorf("looking up entry %q: %w", entry.Path, err)
	}

	item, err := zimEntry.Item(false)
	if err != nil {
		return nil, fmt.Errorf("reading item %q: %w", entry.Path, err)
	}

	data, err := item.DataAll()
	if err != nil {
		return nil, fmt.Errorf("reading data of %q: %w", entry.Path, err)
	}

	return data, nil
}

// readIndex walks the archive directory and classifies every entry once.
func (s *ArchiveSource) readIndex() error {
	mimeList := s.archive.MimeTypeList()
	if mimeList == nil {
		return fmt.Errorf("archive %q has no MIME type list", s.filePath)
	}

	entries := make([]Entry, 0, s.archive.EntryCount())

	for zimEntry := range s.archive.IterateByPath() {
		namespace := zimEntry.Namespace()
		if namespace == zim.NamespaceMetadata || namespace == zim.NamespaceIndex {
			continue
		}

		mimeType := mimeList.MimeType(zimEntry.MimeTypeIndex())
		switch mimeType {
		case mimeTypeDeleted, mimeTypeLinkTarget:
			continue
		case mimeTypeRedirect:
			target := ""
			if resolved, err := zimEntry.RedirectEntry(); err == nil {
				target = resolved.Path()
			}
			entries = append(entries, Entry{
				Index:          zimEntry.Index(),
				Path:           zimEntry.Path(),
				Title:          zimEntry.Title(),
				MimeType:       mimeType,
				IsRedirect:     true,
				RedirectTarget: target,
			})
			continue
		}

		entries = append(entries, Entry{
			Index:    zimEntry.Index(),
			Path:     zimEntry.Path(),
			Title:    zimEntry.Title(),
			MimeType: mimeType,
		})
	}

	s.entries = entries
	return nil
}
