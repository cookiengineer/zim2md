// Package cli contains argument parsing and the parallel extraction runner.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/cookiengineer/zim2md/internal/adapters"
	"github.com/cookiengineer/zim2md/internal/converters"
	"github.com/cookiengineer/zim2md/internal/exporters"
	"github.com/cookiengineer/zim2md/internal/mappers"
	"github.com/cookiengineer/zim2md/internal/parsers"
)

// Version is the zim2md release version.
const Version = "0.1.0"

const maxWorkerCount = 256

const defaultAssetSizeLimit = 50 * 1024 * 1024

// errNoArchives is returned when the command line contains no archive path.
var errNoArchives = errors.New("no archive given")

type configuration struct {
	outputDirectory     string
	exportAssets        bool
	assetSizeLimit      int64
	workerCount         int
	includeExpression   string
	excludeExpression   string
	includePattern      *regexp.Regexp
	excludePattern      *regexp.Regexp
	linkRewriteName     string
	linkRewrite         converters.LinkRewriteMode
	defaultCodeLanguage string
	titleHeading        string
	noClobber           bool
	reportPath          string
	dryRun              bool
	verbose             bool
	quiet               bool
	showVersion         bool
}

func (c *configuration) effectiveHasH1(documentHasH1 bool) bool {
	switch c.titleHeading {
	case "always":
		return false
	case "never":
		return true
	default:
		return documentHasH1
	}
}

type htmlJob struct {
	entry      adapters.Entry
	outputPath string
	indexKey   string
}

type assetJob struct {
	entry      adapters.Entry
	outputPath string
}

// Run parses arguments and converts every given archive. It returns a process
// exit code.
func Run(arguments []string) int {
	config, archives, err := parseArguments(arguments)
	if err != nil {
		switch {
		case errors.Is(err, flag.ErrHelp):
			return 0
		case errors.Is(err, errNoArchives):
			return 1
		default:
			fmt.Fprintln(os.Stderr, "zim2md:", err)
			return 1
		}
	}

	if config.showVersion {
		fmt.Println("zim2md " + Version)
		return 0
	}

	exitCode := 0
	for _, archivePath := range archives {
		switch code := processArchive(config, archivePath); code {
		case 1:
			return 1
		case 2:
			exitCode = 2
		}
	}
	return exitCode
}

func newFlagSet(config *configuration) *flag.FlagSet {
	flags := flag.NewFlagSet("zim2md", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() { printUsage(flags.Output(), flags) }

	flags.StringVar(&config.outputDirectory, "output", ".", "output root directory")
	flags.StringVar(&config.outputDirectory, "o", ".", "output root directory (shorthand)")
	flags.BoolVar(&config.exportAssets, "assets", false, "export referenced non-HTML assets as well")
	flags.Int64Var(&config.assetSizeLimit, "assets-max-size", defaultAssetSizeLimit, "maximum asset size in bytes")
	flags.IntVar(&config.workerCount, "workers", runtime.NumCPU(), "number of parallel conversion workers")
	flags.StringVar(&config.defaultCodeLanguage, "default-code-language", "", "fenced code language used when detection fails")
	flags.StringVar(&config.titleHeading, "title-heading", "auto", "title heading mode: auto, always, never")
	flags.BoolVar(&config.noClobber, "no-clobber", false, "do not overwrite existing files")
	flags.StringVar(&config.reportPath, "report", "", "report file path ('-' for stdout)")
	flags.BoolVar(&config.dryRun, "dry-run", false, "do not write any files")
	flags.BoolVar(&config.verbose, "verbose", false, "verbose progress output")
	flags.BoolVar(&config.quiet, "quiet", false, "suppress the summary")
	flags.BoolVar(&config.showVersion, "version", false, "print version and exit")
	flags.StringVar(&config.includeExpression, "include", "", "only convert entries matching this regular expression")
	flags.StringVar(&config.excludeExpression, "exclude", "", "skip entries matching this regular expression")
	flags.StringVar(&config.linkRewriteName, "link-rewrite", "index", "internal link mode: index, suffix, off")

	return flags
}

func parseArguments(arguments []string) (*configuration, []string, error) {
	config := &configuration{
		outputDirectory: ".",
		assetSizeLimit:  defaultAssetSizeLimit,
		workerCount:     runtime.NumCPU(),
		titleHeading:    "auto",
	}

	flags := newFlagSet(config)
	if err := flags.Parse(arguments); err != nil {
		return nil, nil, err
	}

	if config.includeExpression != "" {
		pattern, err := regexp.Compile(config.includeExpression)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid --include pattern: %w", err)
		}
		config.includePattern = pattern
	}
	if config.excludeExpression != "" {
		pattern, err := regexp.Compile(config.excludeExpression)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid --exclude pattern: %w", err)
		}
		config.excludePattern = pattern
	}

	switch strings.ToLower(config.linkRewriteName) {
	case "index":
		config.linkRewrite = converters.LinkRewriteIndex
	case "suffix":
		config.linkRewrite = converters.LinkRewriteSuffix
	case "off":
		config.linkRewrite = converters.LinkRewriteOff
	default:
		return nil, nil, fmt.Errorf("invalid --link-rewrite value %q", config.linkRewriteName)
	}

	switch strings.ToLower(config.titleHeading) {
	case "auto", "always", "never":
	default:
		return nil, nil, fmt.Errorf("invalid --title-heading value %q", config.titleHeading)
	}

	if config.workerCount <= 0 {
		config.workerCount = 1
	}
	if config.workerCount > maxWorkerCount {
		config.workerCount = maxWorkerCount
	}

	if len(flags.Args()) == 0 && !config.showVersion {
		flags.Usage()
		return nil, nil, errNoArchives
	}

	return config, flags.Args(), nil
}

// printUsage writes the command overview, options and worked examples.
func printUsage(output io.Writer, flags *flag.FlagSet) {
	fmt.Fprintf(output, "zim2md %s - convert OpenZIM archives to Reader-Mode Markdown\n\n", Version)
	fmt.Fprintln(output, "Usage:")
	fmt.Fprintln(output, "  zim2md [options] <archive.zim> [more.zim ...]")
	fmt.Fprintln(output, "")
	fmt.Fprintln(output, "Options:")

	previousOutput := flags.Output()
	flags.SetOutput(output)
	flags.PrintDefaults()
	flags.SetOutput(previousOutput)

	fmt.Fprint(output, `
Output layout:
  <output>/<archive-basename>/<path>.md

  example-123.zim + "path/to/page.html"  ->  example-123/path/to/page.md
  example-123.zim + "path/to/index"      ->  example-123/path/to/index.md
  example-123.zim + "dir/"               ->  example-123/dir/index.md

  Only text/html entries become .md files. Non-HTML assets keep their
  original paths and are copied only with --assets.

Examples:
  # Convert one archive into ./example-123/
  zim2md example-123.zim

  # Convert several archives into ./markdown/ using 8 workers
  zim2md --output ./markdown --workers 8 enwiki.zim dewiki.zim

  # Also export referenced images and other non-HTML assets
  zim2md --assets --output ./markdown example-123.zim

  # Convert only a subset of paths
  zim2md --include '^Go_' --exclude '_talk$' enwiki.zim

  # Preview without writing files, print the report to stdout
  zim2md --dry-run --report - example-123.zim

  # Set a fallback code language and never overwrite existing files
  zim2md --default-code-language text --no-clobber example-123.zim
`)
}

func processArchive(config *configuration, archivePath string) int {
	source, err := adapters.OpenArchiveSource(archivePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "zim2md:", err)
		return 1
	}
	defer source.Close()

	outputRoot := filepath.Join(config.outputDirectory, source.OutputRootName())
	report := exporters.NewReport(archivePath, outputRoot)
	report.DryRun = config.dryRun

	if config.verbose && !config.quiet {
		fmt.Fprintf(os.Stderr, "%s: %d entries\n", archivePath, source.EntryCount())
	}

	runner := &archiveRunner{
		config:       config,
		source:       source,
		exporter:     exporters.NewFileExporter(config.outputDirectory, config.noClobber),
		report:       report,
		htmlIndex:    make(map[string]string),
		newScheme:    source.IsNewNamespaceScheme(),
		outputPrefix: source.OutputRootName(),
	}
	runner.run()

	reportPath := config.reportPath
	if reportPath == "" {
		reportPath = outputRoot + ".zim2md.report.txt"
	}
	if err := report.WriteTo(reportPath); err != nil {
		fmt.Fprintln(os.Stderr, "zim2md: writing report:", err)
	}

	if !config.quiet {
		if config.dryRun {
			fmt.Fprintf(os.Stderr, "%s: dry-run converted %d pages, %d redirects skipped, %d errors\n",
				source.OutputRootName(), report.MarkdownWritten, report.SkippedRedirects, report.ConversionErrors)
		} else {
			fmt.Fprintf(os.Stderr, "%s: wrote %d markdown files, %d redirects skipped, %d errors\n",
				source.OutputRootName(), report.MarkdownWritten, report.SkippedRedirects, report.ConversionErrors)
		}
	}

	if report.ConversionErrors > 0 {
		return 2
	}
	return 0
}

type archiveRunner struct {
	config       *configuration
	source       *adapters.ArchiveSource
	exporter     *exporters.FileExporter
	report       *exporters.Report
	htmlIndex    map[string]string
	newScheme    bool
	outputPrefix string
}

func (runner *archiveRunner) run() {
	entries := runner.source.Entries()
	runner.report.SetTotalEntries(len(entries))

	allocator := mappers.NewOutputPathAllocator()
	htmlJobs := make([]htmlJob, 0, len(entries))
	assetJobs := make([]assetJob, 0, 16)

	for _, entry := range entries {
		if !runner.matchesFilter(entry.Path) {
			continue
		}

		if entry.IsRedirect {
			runner.report.AddRedirect(entry.Path, entry.RedirectTarget)
			continue
		}

		if entry.IsHTML() {
			runner.report.AddHTMLPageFound()
			mapped := mappers.MapHTMLPathToMarkdown(entry.Path, runner.newScheme)
			desired := path.Join(runner.outputPrefix, mapped)
			outputPath, collision := allocator.Allocate(entry.Path, desired)
			if collision {
				runner.report.AddCollision(entry.Path, outputPath)
			}
			indexKey := mappers.StripNamespacePrefix(entry.Path, runner.newScheme)
			runner.htmlIndex[indexKey] = outputPath
			htmlJobs = append(htmlJobs, htmlJob{entry: entry, outputPath: outputPath, indexKey: indexKey})
			continue
		}

		if !runner.config.exportAssets {
			runner.report.IncrementSkippedAssets()
			continue
		}

		desired := path.Join(runner.outputPrefix, mappers.SanitizeRelativePath(mappers.StripNamespacePrefix(entry.Path, runner.newScheme)))
		if desired == "" || desired == runner.outputPrefix {
			runner.report.AddSkippedAsset(entry.Path)
			continue
		}
		outputPath, collision := allocator.Allocate(entry.Path, desired)
		if collision {
			runner.report.AddCollision(entry.Path, outputPath)
		}
		assetJobs = append(assetJobs, assetJob{entry: entry, outputPath: outputPath})
	}

	runner.runHTMLJobs(htmlJobs)
	runner.runAssetJobs(assetJobs)
}

func (runner *archiveRunner) matchesFilter(path string) bool {
	if runner.config.includePattern != nil && !runner.config.includePattern.MatchString(path) {
		return false
	}
	if runner.config.excludePattern != nil && runner.config.excludePattern.MatchString(path) {
		return false
	}
	return true
}

func (runner *archiveRunner) runHTMLJobs(jobs []htmlJob) {
	if len(jobs) == 0 {
		return
	}
	queue := make(chan htmlJob)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < runner.config.workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for job := range queue {
				runner.convertHTMLJob(job)
			}
		}()
	}
	for _, job := range jobs {
		queue <- job
	}
	close(queue)
	waitGroup.Wait()
}

func (runner *archiveRunner) runAssetJobs(jobs []assetJob) {
	if len(jobs) == 0 {
		return
	}
	queue := make(chan assetJob)
	var waitGroup sync.WaitGroup
	for worker := 0; worker < runner.config.workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for job := range queue {
				runner.exportAssetJob(job)
			}
		}()
	}
	for _, job := range jobs {
		queue <- job
	}
	close(queue)
	waitGroup.Wait()
}

func (runner *archiveRunner) convertHTMLJob(job htmlJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			runner.report.AddError(job.entry.Path, fmt.Sprintf("panic: %v", recovered))
		}
	}()

	data, err := runner.source.ReadBytes(job.entry)
	if err != nil {
		runner.report.AddError(job.entry.Path, err.Error())
		return
	}

	document, err := parsers.ParseHTMLDocument(data, job.entry.MimeType)
	if err != nil {
		runner.report.AddError(job.entry.Path, err.Error())
		return
	}

	title := document.Title
	if title == "" {
		title = job.entry.Title
	}

	converter := converters.NewConverter(converters.Options{
		SourceEntryPath:     job.indexKey,
		OutputPath:          job.outputPath,
		HTMLIndex:           runner.htmlIndex,
		DefaultCodeLanguage: runner.config.defaultCodeLanguage,
		NewNamespaceScheme:  runner.newScheme,
		LinkRewrite:         runner.config.linkRewrite,
	})

	markdown := converter.ConvertDocument(document.Root, title, runner.config.effectiveHasH1(document.HasH1))
	if strings.TrimSpace(markdown) == "" {
		runner.report.AddError(job.entry.Path, "no readable content")
		return
	}

	if !runner.config.dryRun {
		if err := runner.exporter.Write(job.outputPath, []byte(markdown)); err != nil {
			if errors.Is(err, exporters.ErrSkipExisting) {
				return
			}
			runner.report.AddError(job.entry.Path, err.Error())
			return
		}
	}

	runner.report.AddMarkdownWritten()
}

func (runner *archiveRunner) exportAssetJob(job assetJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			runner.report.AddError(job.entry.Path, fmt.Sprintf("panic: %v", recovered))
		}
	}()

	data, err := runner.source.ReadBytes(job.entry)
	if err != nil {
		runner.report.AddError(job.entry.Path, err.Error())
		return
	}
	if runner.config.assetSizeLimit > 0 && int64(len(data)) > runner.config.assetSizeLimit {
		runner.report.AddSkippedAsset(job.entry.Path)
		return
	}
	if runner.config.dryRun {
		runner.report.AddAssetExported()
		return
	}
	if err := runner.exporter.Write(job.outputPath, data); err != nil {
		if errors.Is(err, exporters.ErrSkipExisting) {
			return
		}
		runner.report.AddError(job.entry.Path, err.Error())
		return
	}
	runner.report.AddAssetExported()
}
