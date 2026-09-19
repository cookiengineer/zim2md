package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestPrintUsageContainsOverviewAndExamples(t *testing.T) {
	flags := newFlagSet(&configuration{})

	var output bytes.Buffer
	printUsage(&output, flags)
	text := output.String()

	for _, expected := range []string{
		"zim2md " + Version,
		"Usage:",
		"Options:",
		"Output layout:",
		"Examples:",
		"example-123/path/to/page.md",
		"zim2md example-123.zim",
		"--dry-run --report -",
		"--assets",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("usage output missing %q", expected)
		}
	}
}

func TestParseArgumentsWithoutArchives(t *testing.T) {
	originalStderr := os.Stderr
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("opening %s: %v", os.DevNull, err)
	}
	os.Stderr = devNull
	defer func() {
		os.Stderr = originalStderr
		devNull.Close()
	}()

	_, _, err = parseArguments(nil)
	if !errors.Is(err, errNoArchives) {
		t.Fatalf("expected errNoArchives, got %v", err)
	}
}

func TestParseArgumentsWithVersionDoesNotRequireArchive(t *testing.T) {
	config, archives, err := parseArguments([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !config.showVersion {
		t.Fatalf("showVersion not set")
	}
	if len(archives) != 0 {
		t.Fatalf("expected no archives, got %v", archives)
	}
}
