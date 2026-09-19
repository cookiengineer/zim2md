package converters

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

// renderBlocks converts the block-level children of a node into Markdown.
// Consecutive inline content is grouped into paragraphs.
func (c *Converter) renderBlocks(parent *html.Node) string {
	chunks := make([]string, 0, 8)
	var inline strings.Builder

	flushInline := func() {
		text := strings.TrimSpace(inline.String())
		if text != "" {
			chunks = append(chunks, escapeLeadingBlockMarker(text))
		}
		inline.Reset()
	}

	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case textNode:
			inline.WriteString(escapeMarkdownText(collapseWhitespace(child.Data)))
		case elementNode:
			if isMathElement(child) && isDisplayMath(child) {
				flushInline()
				if chunk := c.renderMath(child, true); chunk != "" {
					chunks = append(chunks, chunk)
				}
			} else if isBlockElement(child.Data) {
				flushInline()
				if chunk := c.renderBlockElement(child); chunk != "" {
					chunks = append(chunks, chunk)
				}
			} else {
				inline.WriteString(c.renderInlineElement(child))
			}
		}
	}

	flushInline()
	return strings.Join(chunks, "\n\n")
}

// renderInlineChildren converts the children of a node into inline Markdown.
func (c *Converter) renderInlineChildren(parent *html.Node) string {
	var builder strings.Builder
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case textNode:
			builder.WriteString(escapeMarkdownText(collapseWhitespace(child.Data)))
		case elementNode:
			builder.WriteString(c.renderInlineElement(child))
		}
	}
	return builder.String()
}

// wrapInline wraps non-empty inline content with a symmetric marker.
func wrapInline(marker, inner string) string {
	trimmed := strings.TrimSpace(inner)
	if trimmed == "" {
		return inner
	}
	return marker + trimmed + marker
}

// prefixLines prefixes every line, turning empty lines into the bare prefix.
func prefixLines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	barePrefix := strings.TrimRight(prefix, " ")
	for index, line := range lines {
		if line == "" {
			lines[index] = barePrefix
		} else {
			lines[index] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// indentContinuation indents every line after the first by indent.
func indentContinuation(text, indent string) string {
	if !strings.Contains(text, "\n") {
		return text
	}
	lines := strings.Split(text, "\n")
	for index := 1; index < len(lines); index++ {
		if lines[index] != "" {
			lines[index] = indent + lines[index]
		}
	}
	return strings.Join(lines, "\n")
}

// collapseWhitespace folds every whitespace run into a single space.
func collapseWhitespace(value string) string {
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value))
	previousWasSpace := false

	for _, character := range value {
		if unicode.IsSpace(character) {
			if !previousWasSpace {
				builder.WriteByte(' ')
				previousWasSpace = true
			}
			continue
		}
		builder.WriteRune(character)
		previousWasSpace = false
	}

	return builder.String()
}

// escapeMarkdownText escapes characters that could be interpreted as Markdown
// so that no literal HTML or accidental formatting leaks out.
func escapeMarkdownText(value string) string {
	if value == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(value) + 8)

	for _, character := range value {
		escape := false
		switch character {
		case '\\', '`', '*', '_', '[', ']', '<', '>', '|', '~':
			escape = true
		}
		if escape {
			builder.WriteByte('\\')
		}
		builder.WriteRune(character)
	}

	return builder.String()
}

// escapeLeadingBlockMarker escapes list, blockquote and heading markers that
// appear at the start of a line, which is where Markdown would interpret them.
func escapeLeadingBlockMarker(text string) string {
	lines := strings.Split(text, "\n")
	for index, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		leading := line[:len(line)-len(trimmed)]
		if trimmed == "" || len(leading) > 3 {
			continue
		}

		escape := false
		switch trimmed[0] {
		case '#', '-', '+':
			escape = true
		default:
			if trimmed[0] >= '0' && trimmed[0] <= '9' {
				position := 0
				for position < len(trimmed) && trimmed[position] >= '0' && trimmed[position] <= '9' {
					position++
				}
				if position < len(trimmed) && (trimmed[position] == '.' || trimmed[position] == ')') &&
					(position+1 == len(trimmed) || trimmed[position+1] == ' ') {
					escape = true
				}
			}
		}

		if escape {
			lines[index] = leading + "\\" + trimmed
		}
	}
	return strings.Join(lines, "\n")
}

// escapeLinkTitle escapes a link title for use inside double quotes.
func escapeLinkTitle(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

// formatCodeSpan renders inline code, choosing a fence that is not present in
// the content as required by CommonMark.
func formatCodeSpan(content string) string {
	content = strings.ReplaceAll(content, "\n", " ")
	content = collapseWhitespace(content)

	fence := "`"
	for strings.Contains(content, fence) {
		fence += "`"
	}

	if strings.HasPrefix(content, "`") || strings.HasSuffix(content, "`") {
		content = " " + content + " "
	}
	return fence + content + fence
}
