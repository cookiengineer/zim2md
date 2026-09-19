package converters

import (
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/cookiengineer/zim2md/internal/mappers"
	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

type resolvedReference struct {
	path     string
	fragment string
}

// renderLink renders an anchor, rewriting internal destinations to exported
// Markdown pages when possible.
func (c *Converter) renderLink(anchor *html.Node) string {
	inner := c.renderInlineChildren(anchor)
	if strings.TrimSpace(inner) == "" {
		return inner
	}

	destination := c.resolveLinkDestination(nodes.Attribute(anchor, "href"))
	if destination == "" {
		return inner
	}

	var builder strings.Builder
	builder.WriteByte('[')
	builder.WriteString(inner)
	builder.WriteString("](")
	builder.WriteString(formatDestination(destination))
	if title := nodes.Attribute(anchor, "title"); title != "" {
		plainText := strings.TrimSpace(nodes.TextContent(anchor))
		if !strings.EqualFold(strings.TrimSpace(title), plainText) {
			builder.WriteString(" \"")
			builder.WriteString(escapeLinkTitle(title))
			builder.WriteByte('"')
		}
	}
	builder.WriteByte(')')
	return builder.String()
}

// renderImage renders an <img> as Markdown, skipping tracking pixels.
func (c *Converter) renderImage(image *html.Node) string {
	if shouldSkipImage(image) {
		return ""
	}

	source := strings.TrimSpace(nodes.Attribute(image, "src"))
	alt := escapeMarkdownText(collapseWhitespace(nodes.Attribute(image, "alt")))
	if source == "" {
		return alt
	}
	if !isSafeURL(source) {
		return alt
	}

	var builder strings.Builder
	builder.WriteString("![")
	builder.WriteString(alt)
	builder.WriteString("](")
	builder.WriteString(formatDestination(source))
	if title := nodes.Attribute(image, "title"); title != "" {
		builder.WriteString(" \"")
		builder.WriteString(escapeLinkTitle(title))
		builder.WriteByte('"')
	}
	builder.WriteByte(')')
	return builder.String()
}

// resolveLinkDestination maps an href to a safe, export-aware destination.
func (c *Converter) resolveLinkDestination(href string) string {
	raw := strings.TrimSpace(href)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "#") {
		return raw
	}
	if hasControlCharacter(raw) {
		return ""
	}

	probe := strings.ToLower(strings.TrimLeft(raw, " \t\r\n"))
	if strings.HasPrefix(probe, "javascript:") || strings.HasPrefix(probe, "vbscript:") || strings.HasPrefix(probe, "data:") {
		return ""
	}
	if strings.HasPrefix(probe, "//") {
		return raw
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Scheme != "" {
		return raw
	}

	if c.options.LinkRewrite == LinkRewriteOff {
		return raw
	}

	resolved := c.resolveAgainstSource(raw)

	if c.options.LinkRewrite == LinkRewriteIndex && c.options.HTMLIndex != nil {
		if target, ok := c.options.HTMLIndex[resolved.path]; ok {
			return mappers.RelativeMarkdownLink(c.options.OutputPath, target, resolved.fragment)
		}
	}

	if c.options.LinkRewrite == LinkRewriteSuffix {
		if mapped, ok := c.suffixCandidate(resolved.path); ok {
			return mappers.RelativeMarkdownLink(c.options.OutputPath, mapped, resolved.fragment)
		}
	}

	return raw
}

// resolveAgainstSource resolves a relative reference against the source entry.
func (c *Converter) resolveAgainstSource(reference string) resolvedReference {
	base, err := url.Parse(c.options.SourceEntryPath)
	if err != nil {
		return resolvedReference{path: reference}
	}
	parsed, err := url.Parse(reference)
	if err != nil {
		return resolvedReference{path: reference}
	}
	full := base.ResolveReference(parsed)
	return resolvedReference{
		path:     strings.TrimPrefix(full.Path, "/"),
		fragment: full.Fragment,
	}
}

// suffixCandidate maps a likely HTML path with the extension rule only. Assets
// are deliberately never rewritten.
func (c *Converter) suffixCandidate(sourcePath string) (string, bool) {
	extension := strings.ToLower(path.Ext(sourcePath))
	if extension != "" && extension != ".html" && extension != ".htm" {
		return "", false
	}
	return mappers.MapHTMLPathToMarkdown(sourcePath, c.options.NewNamespaceScheme), true
}

// formatDestination percent-encodes characters that would break the Markdown
// link syntax.
func formatDestination(destination string) string {
	if destination == "" || strings.HasPrefix(destination, "#") {
		return destination
	}
	replacer := strings.NewReplacer(
		" ", "%20",
		"(", "%28",
		")", "%29",
		"<", "%3C",
		">", "%3E",
		"\"", "%22",
	)
	return replacer.Replace(destination)
}

// isSafeURL rejects active-content URLs while allowing normal links.
func isSafeURL(raw string) bool {
	probe := strings.ToLower(strings.TrimLeft(raw, " \t\r\n"))
	switch {
	case strings.HasPrefix(probe, "javascript:"),
		strings.HasPrefix(probe, "vbscript:"),
		strings.HasPrefix(probe, "data:"):
		return false
	}
	return true
}

func hasControlCharacter(value string) bool {
	for _, character := range value {
		if character < 0x20 || character == 0x7f {
			return true
		}
	}
	return false
}

func shouldSkipImage(image *html.Node) bool {
	width := parseIntegerAttribute(image, "width")
	height := parseIntegerAttribute(image, "height")
	if (width > 0 && width <= 2) || (height > 0 && height <= 2) {
		return true
	}

	for _, class := range nodes.ClassList(image) {
		switch strings.ToLower(class) {
		case "tracking-pixel", "trackingpixel", "spacer", "pixel", "clearfix-image",
			"run", "copy", "clipboard", "play", "edit-pencil", "edit-icon":
			return true
		}
	}
	return false
}

func parseIntegerAttribute(node *html.Node, name string) int {
	value := strings.TrimSpace(nodes.Attribute(node, name))
	if value == "" {
		return 0
	}
	value = strings.TrimSuffix(value, "px")
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
