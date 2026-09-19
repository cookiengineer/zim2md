package parsers

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

// Document is the sanitized, reader-mode view of an HTML page.
type Document struct {
	// Root is the extracted main content subtree.
	Root *html.Node
	// Title is the document or first-heading title.
	Title string
	// HasH1 reports whether the content already contains a top-level heading.
	HasH1 bool
	// Language is the document language from <html lang>, if present.
	Language string
}

// ParseHTMLDocument decodes, parses and sanitizes an HTML document.
func ParseHTMLDocument(raw []byte, contentType string) (*Document, error) {
	decoded := raw
	if value, err := decodeToUTF8(raw, contentType); err == nil {
		decoded = value
	}

	root, err := html.Parse(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("parsing HTML document: %w", err)
	}

	document := &Document{
		Root:     root,
		Title:    extractDocumentTitle(root),
		Language: extractDocumentLanguage(root),
	}

	sanitizeDocument(root)

	body := findElement(root, "body")
	if body == nil {
		body = root
	}

	candidate, scores := selectMainContent(body)
	if candidate == nil {
		candidate = body
	}
	candidate = expandToSiblings(candidate, scores)
	if candidate == nil {
		candidate = body
	}

	removeEmptyElements(candidate)

	document.Root = candidate
	document.HasH1 = nodes.ContainsElement(candidate, "h1")
	return document, nil
}

func decodeToUTF8(raw []byte, contentType string) ([]byte, error) {
	reader, err := charset.NewReader(bytes.NewReader(raw), contentType)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func extractDocumentTitle(root *html.Node) string {
	if title := findElement(root, "title"); title != nil {
		if text := strings.TrimSpace(nodes.TextContent(title)); text != "" {
			return text
		}
	}
	if heading := findElement(root, "h1"); heading != nil {
		return strings.TrimSpace(nodes.TextContent(heading))
	}
	return ""
}

func extractDocumentLanguage(root *html.Node) string {
	htmlElement := findElement(root, "html")
	if htmlElement == nil {
		return ""
	}
	return strings.TrimSpace(nodes.Attribute(htmlElement, "lang"))
}

func findElement(root *html.Node, tag string) *html.Node {
	var found *html.Node
	nodes.Walk(root, func(node *html.Node) bool {
		if nodes.IsElement(node, tag) {
			found = node
			return false
		}
		return true
	})
	return found
}
