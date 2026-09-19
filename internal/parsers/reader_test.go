package parsers

import (
	"strings"
	"testing"

	"github.com/cookiengineer/zim2md/internal/nodes"
)

func TestExtractMainContentPrefersArticle(t *testing.T) {
	longParagraph := strings.Repeat("This is meaningful article content. ", 20)
	source := `<html lang="en"><head><title>Document title</title></head><body>` +
		`<nav><a href="/">Home</a><a href="/about">About</a></nav>` +
		`<aside class="sidebar"><p>Sidebar links and widgets</p></aside>` +
		`<article><h1>Article heading</h1><p>` + longParagraph + `</p><p>` + longParagraph + `</p></article>` +
		`<footer><p>Footer text</p></footer></body></html>`

	document, err := ParseHTMLDocument([]byte(source), "text/html")
	if err != nil {
		t.Fatalf("ParseHTMLDocument: %v", err)
	}

	if document.Title != "Document title" {
		t.Fatalf("title = %q", document.Title)
	}
	if !document.HasH1 {
		t.Fatalf("expected HasH1 to be true")
	}
	if document.Language != "en" {
		t.Fatalf("language = %q", document.Language)
	}

	text := nodes.TextContent(document.Root)
	if !strings.Contains(text, "meaningful article content") {
		t.Fatalf("main content missing:\n%s", text)
	}
	if strings.Contains(text, "Sidebar links") {
		t.Fatalf("sidebar leaked into content:\n%s", text)
	}
}

func TestRemoveScriptsStylesAndNavigation(t *testing.T) {
	source := `<html><body><nav>menu</nav><script>alert("x")</script>` +
		`<style>body{color:red}</style><noscript>fallback</noscript>` +
		`<p>` + strings.Repeat("readable paragraph ", 20) + `</p></body></html>`

	document, err := ParseHTMLDocument([]byte(source), "text/html")
	if err != nil {
		t.Fatalf("ParseHTMLDocument: %v", err)
	}

	text := nodes.TextContent(document.Root)
	for _, forbidden := range []string{"alert(", "body{", "fallback", "menu"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("forbidden %q leaked into content:\n%s", forbidden, text)
		}
	}
}

func TestStructuralRootsAreNeverRemoved(t *testing.T) {
	// Regression: an <html> element carrying a "vector-toc-..." class used to
	// match a negative substring and the whole document was dropped.
	source := `<html class="vector-toc-not-available skin-vector"><body class="mw-hide-empty-elt">` +
		`<p>` + strings.Repeat("readable content ", 20) + `</p></body></html>`

	document, err := ParseHTMLDocument([]byte(source), "text/html")
	if err != nil {
		t.Fatalf("ParseHTMLDocument: %v", err)
	}
	if !strings.Contains(nodes.TextContent(document.Root), "readable content") {
		t.Fatalf("content was dropped with the structural root")
	}
}

func TestTitleFallsBackToFirstHeading(t *testing.T) {
	source := `<html><head></head><body><section><h1>Only heading</h1>` +
		`<p>` + strings.Repeat("content ", 30) + `</p></section></body></html>`

	document, err := ParseHTMLDocument([]byte(source), "text/html")
	if err != nil {
		t.Fatalf("ParseHTMLDocument: %v", err)
	}
	if document.Title != "Only heading" {
		t.Fatalf("title = %q, want %q", document.Title, "Only heading")
	}
}

func TestFailsafeOnEmptyAndMalformedInput(t *testing.T) {
	for _, source := range []string{"", "<p><b>unclosed", "not html at all", "<html>", "<table><tr><td>x"} {
		document, err := ParseHTMLDocument([]byte(source), "")
		if err != nil {
			t.Fatalf("ParseHTMLDocument(%q) returned error: %v", source, err)
		}
		if document == nil || document.Root == nil {
			t.Fatalf("ParseHTMLDocument(%q) returned a nil root", source)
		}
	}
}
