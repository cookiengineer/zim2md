package mappers

import (
	"strings"
	"testing"
)

func TestMapHTMLPathToMarkdown(t *testing.T) {
	testCases := []struct {
		name       string
		sourcePath string
		newScheme  bool
		expected   string
	}{
		{name: "html extension", sourcePath: "path/to/page.html", newScheme: true, expected: "path/to/page.md"},
		{name: "htm extension", sourcePath: "path/to/page.htm", newScheme: true, expected: "path/to/page.md"},
		{name: "no extension", sourcePath: "path/to/index", newScheme: true, expected: "path/to/index.md"},
		{name: "php extension", sourcePath: "path/to/index.php", newScheme: true, expected: "path/to/index.md"},
		{name: "double extension", sourcePath: "path/to/archive.tar.gz", newScheme: true, expected: "path/to/archive.tar.md"},
		{name: "dotfile", sourcePath: ".NET", newScheme: true, expected: ".NET.md"},
		{name: "dotfile with extension", sourcePath: ".NET.Core", newScheme: true, expected: ".NET.md"},
		{name: "js named html page", sourcePath: "Node.js", newScheme: true, expected: "Node.md"},
		{name: "trailing slash", sourcePath: "path/to/dir/", newScheme: true, expected: "path/to/dir/index.md"},
		{name: "root", sourcePath: "/", newScheme: true, expected: "index.md"},
		{name: "empty", sourcePath: "", newScheme: true, expected: "index.md"},
		{name: "leading slash", sourcePath: "/foo/bar.html", newScheme: true, expected: "foo/bar.md"},
		{name: "old namespace a", sourcePath: "A/foo.html", newScheme: false, expected: "foo.md"},
		{name: "old namespace i", sourcePath: "I/pic", newScheme: false, expected: "pic.md"},
		{name: "new namespace literal a", sourcePath: "A/foo.html", newScheme: true, expected: "A/foo.md"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := MapHTMLPathToMarkdown(testCase.sourcePath, testCase.newScheme)
			if got != testCase.expected {
				t.Fatalf("MapHTMLPathToMarkdown(%q, %v) = %q, want %q", testCase.sourcePath, testCase.newScheme, got, testCase.expected)
			}
		})
	}
}

func TestReplaceFinalExtensionWithMarkdown(t *testing.T) {
	testCases := map[string]string{
		"page.html":      "page.md",
		"index":          "index.md",
		"index.php":      "index.md",
		"archive.tar.gz": "archive.tar.md",
		".NET":           ".NET.md",
		".AURINFO":       ".AURINFO.md",
		"Node.js":        "Node.md",
		"":               "index.md",
	}
	for input, expected := range testCases {
		if got := ReplaceFinalExtensionWithMarkdown(input); got != expected {
			t.Errorf("ReplaceFinalExtensionWithMarkdown(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestSanitizeRelativePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "traversal", input: "../../etc/passwd", expected: "_/_/etc/passwd"},
		{name: "single dot", input: "./a/./b", expected: "_/a/_/b"},
		{name: "control characters", input: "a/b\x00c", expected: "a/b_c"},
		{name: "illegal filesystem chars", input: `a<>:"|?*b`, expected: "a_______b"},
		{name: "empty segments collapse", input: "a//b", expected: "a/b"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := SanitizeRelativePath(testCase.input)
			if got != testCase.expected {
				t.Fatalf("SanitizeRelativePath(%q) = %q, want %q", testCase.input, got, testCase.expected)
			}
		})
	}
}

func TestSanitizeRelativePathTruncatesLongSegments(t *testing.T) {
	long := strings.Repeat("x", 500)
	got := SanitizeRelativePath(long)
	if len(got) > maxPathSegmentBytes {
		t.Fatalf("segment not truncated: length %d", len(got))
	}
	if !strings.Contains(got, "-") {
		t.Fatalf("expected a hash suffix in %q", got)
	}
}

func TestRelativeMarkdownLink(t *testing.T) {
	testCases := []struct {
		name     string
		from     string
		to       string
		fragment string
		expected string
	}{
		{name: "sibling", from: "root/a/current.md", to: "root/a/target.md", expected: "target.md"},
		{name: "parent", from: "root/a/b/current.md", to: "root/target.md", expected: "../../target.md"},
		{name: "fragment", from: "root/current.md", to: "root/target.md", fragment: "section", expected: "target.md#section"},
		{name: "spaces escaped", from: "root/current.md", to: "root/a target.md", expected: "a%20target.md"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := RelativeMarkdownLink(testCase.from, testCase.to, testCase.fragment)
			if got != testCase.expected {
				t.Fatalf("RelativeMarkdownLink() = %q, want %q", got, testCase.expected)
			}
		})
	}
}

func TestOutputPathAllocator(t *testing.T) {
	allocator := NewOutputPathAllocator()

	first, collision := allocator.Allocate("a", "root/foo.md")
	if collision {
		t.Fatalf("first allocation reported a collision")
	}
	if first != "root/foo.md" {
		t.Fatalf("first = %q", first)
	}

	second, collision := allocator.Allocate("b", "root/foo.md")
	if !collision {
		t.Fatalf("second allocation should collide")
	}
	if second == first {
		t.Fatalf("collision path equals original: %q", second)
	}
	if !strings.HasSuffix(second, ".md") {
		t.Fatalf("collision path lost extension: %q", second)
	}
}
