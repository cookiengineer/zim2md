package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	sampleGobyExample = "../../samples/gobyexample.com_en_all_2026-08.zim"
	sampleDevdocs     = "../../samples/devdocs_en_go_2026-07.zim"
	sampleArchlinux   = "../../samples/archlinux_en_all_maxi_2026-07.zim"
)

var rawHTMLPattern = regexp.MustCompile(`<[a-zA-Z][a-zA-Z0-9]*(\s|>|/)`)

func requireSample(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("sample %s not available: %v", path, err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

func assertNoRawHTML(t *testing.T, path string) {
	t.Helper()
	inFence := false
	for lineNumber, line := range strings.Split(readFile(t, path), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		unescaped := strings.ReplaceAll(line, `\<`, "")
		if rawHTMLPattern.MatchString(unescaped) {
			t.Errorf("%s:%d raw HTML outside a code fence: %s", path, lineNumber+1, line)
		}
	}
}

func TestEndToEndGobyExample(t *testing.T) {
	requireSample(t, sampleGobyExample)
	output := t.TempDir()

	if code := Run([]string{"--output", output, "--workers", "4", sampleGobyExample}); code != 0 {
		t.Fatalf("Run returned exit code %d", code)
	}

	base := filepath.Join(output, "gobyexample.com_en_all_2026-08")

	// Trailing slash source path becomes index.md.
	if _, err := os.Stat(filepath.Join(base, "gobyexample.com", "index.md")); err != nil {
		t.Fatalf("index.md not written: %v", err)
	}

	arraysPath := filepath.Join(base, "gobyexample.com", "arrays.md")
	arrays := readFile(t, arraysPath)
	if !strings.HasPrefix(arrays, "# Go by Example: Arrays") {
		t.Fatalf("missing prepended title:\n%s", arrays)
	}
	if !strings.Contains(arrays, "```go") {
		t.Fatalf("missing go code fence:\n%s", arrays)
	}
	if !strings.Contains(arrays, "slices.md") {
		t.Fatalf("internal link was not rewritten:\n%s", arrays)
	}
	if strings.Contains(arrays, "play.png") || strings.Contains(arrays, "clipboard.png") {
		t.Fatalf("UI icon images not skipped:\n%s", arrays)
	}
	assertNoRawHTML(t, arraysPath)

	report := readFile(t, filepath.Join(output, "gobyexample.com_en_all_2026-08.zim2md.report.txt"))
	if !strings.Contains(report, "mainPage -> gobyexample.com/") {
		t.Fatalf("redirect not reported:\n%s", report)
	}
	if !strings.Contains(report, "conversion errors:  0") {
		t.Fatalf("unexpected conversion errors:\n%s", report)
	}
}

func TestEndToEndDevdocs(t *testing.T) {
	requireSample(t, sampleDevdocs)
	output := t.TempDir()

	if code := Run([]string{"--output", output, "--workers", "4", sampleDevdocs}); code != 0 {
		t.Fatalf("Run returned exit code %d", code)
	}

	base := filepath.Join(output, "devdocs_en_go_2026-07")
	bufioPath := filepath.Join(base, "bufio", "index.md")
	bufio := readFile(t, bufioPath)
	if !strings.Contains(bufio, "# Package bufio") {
		t.Fatalf("missing heading:\n%s", bufio)
	}
	if !strings.Contains(bufio, "```go") {
		t.Fatalf("data-language code fence not detected:\n%s", bufio)
	}
	if strings.Contains(bufio, "devdocs-navbar") || strings.Contains(bufio, "<nav") {
		t.Fatalf("navigation leaked:\n%s", bufio)
	}
	assertNoRawHTML(t, bufioPath)
}

func TestEndToEndArchlinux(t *testing.T) {
	requireSample(t, sampleArchlinux)
	if testing.Short() {
		t.Skip("skipping large archive in short mode")
	}
	output := t.TempDir()

	if code := Run([]string{"--output", output, "--workers", "8", sampleArchlinux}); code != 0 {
		t.Fatalf("Run returned exit code %d", code)
	}

	base := filepath.Join(output, "archlinux_en_all_maxi_2026-07")

	dotnetPath := filepath.Join(base, ".NET.md")
	dotnet := readFile(t, dotnetPath)
	if !strings.Contains(dotnet, "FOSS software framework") {
		t.Fatalf("regression: .NET article body was not extracted:\n%s", dotnet)
	}
	if !strings.Contains(dotnet, "## Installation") {
		t.Fatalf("missing section heading:\n%s", dotnet)
	}
	assertNoRawHTML(t, dotnetPath)

	systemdPath := filepath.Join(base, "Systemd.md")
	if data, err := os.Stat(systemdPath); err == nil && data.Size() > 0 {
		if !strings.Contains(readFile(t, systemdPath), "| --- |") {
			t.Fatalf("data table was not converted:\n%s", readFile(t, systemdPath))
		}
		assertNoRawHTML(t, systemdPath)
	}

	report := readFile(t, filepath.Join(output, "archlinux_en_all_maxi_2026-07.zim2md.report.txt"))
	if !strings.Contains(report, "conversion errors:  0") {
		t.Fatalf("unexpected conversion errors:\n%s", report)
	}
}

func TestEndToEndAssets(t *testing.T) {
	requireSample(t, sampleGobyExample)
	output := t.TempDir()

	if code := Run([]string{"--output", output, "--workers", "4", "--assets", sampleGobyExample}); code != 0 {
		t.Fatalf("Run returned exit code %d", code)
	}

	found := false
	_ = filepath.Walk(filepath.Join(output, "gobyexample.com_en_all_2026-08"), func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(path, ".png") {
			found = true
		}
		return nil
	})
	if !found {
		t.Fatalf("--assets did not export any png")
	}
}

func TestEndToEndDryRun(t *testing.T) {
	requireSample(t, sampleGobyExample)
	output := t.TempDir()

	if code := Run([]string{"--output", output, "--dry-run", sampleGobyExample}); code != 0 {
		t.Fatalf("Run returned exit code %d", code)
	}

	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			t.Fatalf("dry run wrote %s", entry.Name())
		}
	}
}
