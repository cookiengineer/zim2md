package exporters

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// RedirectRecord documents a skipped redirect entry.
type RedirectRecord struct {
	Source string
	Target string
}

// ErrorRecord documents a page that failed to convert.
type ErrorRecord struct {
	Path   string
	Reason string
}

// CollisionRecord documents an output path collision.
type CollisionRecord struct {
	SourcePath string
	OutputPath string
}

// Report accumulates run statistics and anomalies. It is safe for concurrent
// use.
type Report struct {
	mu sync.Mutex

	ArchivePath     string
	OutputDirectory string
	DryRun          bool

	TotalEntries      int
	HTMLPagesFound    int
	MarkdownWritten   int
	AssetsExported    int
	SkippedAssetCount int
	SkippedRedirects  int
	ConversionErrors  int

	Redirects  []RedirectRecord
	Skipped    []string
	Errors     []ErrorRecord
	Collisions []CollisionRecord
}

// NewReport creates an empty report for one archive.
func NewReport(archivePath, outputDirectory string) *Report {
	return &Report{
		ArchivePath:     archivePath,
		OutputDirectory: outputDirectory,
	}
}

// SetTotalEntries records how many entries the archive index contains.
func (r *Report) SetTotalEntries(count int) {
	r.mu.Lock()
	r.TotalEntries = count
	r.mu.Unlock()
}

// AddHTMLPageFound increments the HTML page counter.
func (r *Report) AddHTMLPageFound() {
	r.mu.Lock()
	r.HTMLPagesFound++
	r.mu.Unlock()
}

// AddMarkdownWritten increments the written-file counter.
func (r *Report) AddMarkdownWritten() {
	r.mu.Lock()
	r.MarkdownWritten++
	r.mu.Unlock()
}

// AddAssetExported increments the exported-asset counter.
func (r *Report) AddAssetExported() {
	r.mu.Lock()
	r.AssetsExported++
	r.mu.Unlock()
}

// AddRedirect records a skipped redirect.
func (r *Report) AddRedirect(source, target string) {
	r.mu.Lock()
	r.Redirects = append(r.Redirects, RedirectRecord{Source: source, Target: target})
	r.SkippedRedirects++
	r.mu.Unlock()
}

// AddSkippedAsset records an asset that was not exported.
func (r *Report) AddSkippedAsset(path string) {
	r.mu.Lock()
	r.Skipped = append(r.Skipped, path)
	r.SkippedAssetCount++
	r.mu.Unlock()
}

// IncrementSkippedAssets counts an asset that was intentionally not exported
// without listing its path (used when --assets is disabled).
func (r *Report) IncrementSkippedAssets() {
	r.mu.Lock()
	r.SkippedAssetCount++
	r.mu.Unlock()
}

// AddError records a conversion error.
func (r *Report) AddError(path, reason string) {
	r.mu.Lock()
	r.Errors = append(r.Errors, ErrorRecord{Path: path, Reason: reason})
	r.ConversionErrors++
	r.mu.Unlock()
}

// AddCollision records an output path collision.
func (r *Report) AddCollision(sourcePath, outputPath string) {
	r.mu.Lock()
	r.Collisions = append(r.Collisions, CollisionRecord{SourcePath: sourcePath, OutputPath: outputPath})
	r.mu.Unlock()
}

// Render returns a human readable report.
func (r *Report) Render() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var builder strings.Builder
	builder.WriteString("zim2md report\n")
	builder.WriteString("=============\n")
	fmt.Fprintf(&builder, "archive:            %s\n", r.ArchivePath)
	fmt.Fprintf(&builder, "output:             %s\n", r.OutputDirectory)
	if r.DryRun {
		builder.WriteString("mode:               dry-run (no files written)\n")
	}
	fmt.Fprintf(&builder, "total entries:      %d\n", r.TotalEntries)
	fmt.Fprintf(&builder, "html pages found:   %d\n", r.HTMLPagesFound)
	fmt.Fprintf(&builder, "markdown written:   %d\n", r.MarkdownWritten)
	fmt.Fprintf(&builder, "assets exported:    %d\n", r.AssetsExported)
	fmt.Fprintf(&builder, "assets skipped:     %d\n", r.SkippedAssetCount)
	fmt.Fprintf(&builder, "redirects skipped:  %d\n", r.SkippedRedirects)
	fmt.Fprintf(&builder, "conversion errors:  %d\n", r.ConversionErrors)
	fmt.Fprintf(&builder, "path collisions:    %d\n", len(r.Collisions))

	writeRedirectSection(&builder, r.Redirects)
	writeCollisionSection(&builder, r.Collisions)
	writeErrorSection(&builder, r.Errors)
	writeSkippedSection(&builder, r.Skipped)

	return builder.String()
}

// WriteTo writes the rendered report to path, creating parent directories.
func (r *Report) WriteTo(path string) error {
	if path == "" || path == "-" {
		fmt.Print(r.Render())
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating report directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(r.Render()), 0o644); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}
	return nil
}

func writeRedirectSection(builder *strings.Builder, redirects []RedirectRecord) {
	if len(redirects) == 0 {
		return
	}
	sort.Slice(redirects, func(i, j int) bool { return redirects[i].Source < redirects[j].Source })
	builder.WriteString("\nredirects:\n")
	for _, redirect := range redirects {
		fmt.Fprintf(builder, "  %s -> %s\n", redirect.Source, redirect.Target)
	}
}

func writeCollisionSection(builder *strings.Builder, collisions []CollisionRecord) {
	if len(collisions) == 0 {
		return
	}
	sort.Slice(collisions, func(i, j int) bool { return collisions[i].SourcePath < collisions[j].SourcePath })
	builder.WriteString("\ncollisions:\n")
	for _, collision := range collisions {
		fmt.Fprintf(builder, "  %s -> %s\n", collision.SourcePath, collision.OutputPath)
	}
}

func writeErrorSection(builder *strings.Builder, errors []ErrorRecord) {
	if len(errors) == 0 {
		return
	}
	sort.Slice(errors, func(i, j int) bool { return errors[i].Path < errors[j].Path })
	builder.WriteString("\nerrors:\n")
	for _, item := range errors {
		fmt.Fprintf(builder, "  %s: %s\n", item.Path, item.Reason)
	}
}

func writeSkippedSection(builder *strings.Builder, skipped []string) {
	if len(skipped) == 0 {
		return
	}
	sort.Strings(skipped)
	builder.WriteString("\nskipped assets:\n")
	for _, path := range skipped {
		fmt.Fprintf(builder, "  %s\n", path)
	}
}
