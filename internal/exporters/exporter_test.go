package exporters

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileExporterWritesAndOverwrites(t *testing.T) {
	root := t.TempDir()
	exporter := NewFileExporter(root, false)

	if err := exporter.Write("archive/dir/page.md", []byte("first")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := exporter.Write("archive/dir/page.md", []byte("second")); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "archive", "dir", "page.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("content = %q", data)
	}
}

func TestFileExporterNoClobber(t *testing.T) {
	root := t.TempDir()
	exporter := NewFileExporter(root, true)

	if err := exporter.Write("page.md", []byte("first")); err != nil {
		t.Fatalf("write: %v", err)
	}
	err := exporter.Write("page.md", []byte("second"))
	if !errors.Is(err, ErrSkipExisting) {
		t.Fatalf("expected ErrSkipExisting, got %v", err)
	}
}

func TestFileExporterRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	exporter := NewFileExporter(root, false)

	if err := exporter.Write("../escape.md", []byte("x")); err == nil {
		t.Fatalf("expected traversal to be rejected")
	}
}

func TestReportRender(t *testing.T) {
	report := NewReport("archive.zim", "out/archive")
	report.SetTotalEntries(10)
	report.AddHTMLPageFound()
	report.AddMarkdownWritten()
	report.AddRedirect("old", "new")
	report.AddError("broken", "boom")

	rendered := report.Render()
	for _, expected := range []string{"archive.zim", "html pages found:   1", "old -> new", "broken: boom"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("report missing %q:\n%s", expected, rendered)
		}
	}
}
