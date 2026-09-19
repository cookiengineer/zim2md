package converters

import (
	"strings"
	"testing"
)

func TestCanonicalizeLanguage(t *testing.T) {
	testCases := map[string]string{
		"golang":    "go",
		"Go":        "go",
		"py":        "python",
		"python3":   "python",
		"js":        "javascript",
		"sh":        "bash",
		"shell":     "bash",
		"C++":       "cpp",
		"xhtml":     "html",
		"yml":       "yaml",
		"":          "",
		"bad lang!": "",
	}
	for input, expected := range testCases {
		if got := canonicalizeLanguage(input); got != expected {
			t.Errorf("canonicalizeLanguage(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestGuessLanguageFromCode(t *testing.T) {
	testCases := []struct {
		name     string
		code     string
		expected string
	}{
		{name: "go package", code: "package main\n\nimport \"fmt\"", expected: "go"},
		{name: "go walrus", code: "x := 1", expected: "go"},
		{name: "json", code: "{\n  \"a\": 1\n}", expected: "json"},
		{name: "html doctype", code: "<!DOCTYPE html>\n<html>", expected: "html"},
		{name: "xml", code: "<?xml version=\"1.0\"?>", expected: "xml"},
		{name: "bash prompt", code: "$ go version\ngo version go1.22", expected: "bash"},
		{name: "c include", code: "#include <stdio.h>", expected: "c"},
		{name: "python", code: "def main():\n    print('hi')", expected: "python"},
		{name: "sql", code: "SELECT id FROM users", expected: "sql"},
		{name: "yaml", code: "name: test\nversion: 1", expected: "yaml"},
		{name: "go map is not yaml", code: "m := map[string]int{\"a\": 1}", expected: "go"},
		{name: "empty", code: "", expected: ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := guessLanguageFromCode(testCase.code); got != testCase.expected {
				t.Fatalf("guessLanguageFromCode(%q) = %q, want %q", testCase.code, got, testCase.expected)
			}
		})
	}
}

func TestFenceForCode(t *testing.T) {
	if got := fenceForCode("plain"); got != "```" {
		t.Fatalf("fenceForCode(plain) = %q", got)
	}
	if got := fenceForCode("a ``` b"); got != "````" {
		t.Fatalf("fenceForCode with triple backticks = %q", got)
	}
}

func TestHTMLCodeBlockKeepsMarkupInsideFence(t *testing.T) {
	source := `<pre data-language="html">&lt;div class="x"&gt;&lt;/div&gt;</pre>`
	markdown := convertHTML(t, source, Options{})

	if !strings.Contains(markdown, "```html") {
		t.Fatalf("expected an html fenced block:\n%s", markdown)
	}
	if !strings.Contains(markdown, `<div class="x"></div>`) {
		t.Fatalf("html source should be preserved inside the fence:\n%s", markdown)
	}
}

func TestDetectLanguageFromAttributes(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		expected string
	}{
		{name: "data-language", source: `<pre data-language="go">x</pre>`, expected: "```go"},
		{name: "class language", source: `<pre class="language-python">x</pre>`, expected: "```python"},
		{name: "code class language", source: `<pre><code class="language-rust">x</code></pre>`, expected: "```rust"},
		{name: "program output", source: `<pre class="output" data-language="go">x</pre>`, expected: "```text"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			markdown := convertHTML(t, testCase.source, Options{})
			if !strings.Contains(markdown, testCase.expected) {
				t.Fatalf("expected %q in:\n%s", testCase.expected, markdown)
			}
		})
	}
}
