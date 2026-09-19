// Package converters turns a sanitized HTML DOM into Markdown. It owns the
// Markdown writer, the element mapping, code-block language detection and
// link/image rewriting.
package converters

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

const (
	textNode    = html.TextNode
	elementNode = html.ElementNode
)

// LinkRewriteMode controls how internal links are translated.
type LinkRewriteMode int

const (
	// LinkRewriteIndex rewrites links by looking the target up in HTMLIndex.
	LinkRewriteIndex LinkRewriteMode = iota
	// LinkRewriteSuffix rewrites links by applying the ".md" suffix rule.
	LinkRewriteSuffix
	// LinkRewriteOff leaves every link untouched.
	LinkRewriteOff
)

// Options configures a single page conversion.
type Options struct {
	// SourceEntryPath is the HTML entry path inside the ZIM archive.
	SourceEntryPath string
	// OutputPath is the Markdown path relative to the archive output root.
	OutputPath string
	// HTMLIndex maps a source HTML entry path to its Markdown output path.
	HTMLIndex map[string]string
	// DefaultCodeLanguage is used when no language can be detected.
	DefaultCodeLanguage string
	// NewNamespaceScheme mirrors the archive namespace scheme.
	NewNamespaceScheme bool
	// LinkRewrite selects the link translation strategy.
	LinkRewrite LinkRewriteMode
}

// Converter converts HTML nodes to Markdown using fixed options.
type Converter struct {
	options Options

	documentLanguage    string
	documentLanguageSet bool
}

// NewConverter returns a converter for one page.
func NewConverter(options Options) *Converter {
	return &Converter{options: options}
}

var multipleBlankLines = regexp.MustCompile(`\n{3,}`)

// ConvertDocument renders the extracted content root to a Markdown document.
// When the content has no top-level heading and a title is known, a single
// leading H1 is added.
func (c *Converter) ConvertDocument(contentRoot *html.Node, title string, hasH1 bool) string {
	body := strings.TrimSpace(c.renderBlocks(contentRoot))
	body = multipleBlankLines.ReplaceAllString(body, "\n\n")

	var builder strings.Builder
	if title != "" && !hasH1 {
		builder.WriteString("# ")
		builder.WriteString(escapeMarkdownText(collapseWhitespace(title)))
		builder.WriteString("\n\n")
	}
	builder.WriteString(body)
	builder.WriteString("\n")
	return builder.String()
}
