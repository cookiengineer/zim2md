package converters

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

var blockElements = map[string]struct{}{
	"address": {}, "article": {}, "aside": {}, "blockquote": {}, "body": {},
	"caption": {}, "center": {}, "dd": {}, "details": {}, "div": {}, "dl": {},
	"dt": {}, "fieldset": {}, "figcaption": {}, "figure": {}, "footer": {},
	"form": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"header": {}, "hr": {}, "html": {}, "legend": {}, "li": {}, "main": {},
	"nav": {}, "ol": {}, "p": {}, "pre": {}, "section": {}, "summary": {},
	"table": {}, "tbody": {}, "td": {}, "tfoot": {}, "th": {}, "thead": {},
	"tr": {}, "ul": {},
}

func isBlockElement(tag string) bool {
	_, ok := blockElements[tag]
	return ok
}

// renderBlockElement renders a single block-level element to Markdown.
func (c *Converter) renderBlockElement(node *html.Node) string {
	if isMathElement(node) {
		return c.renderMath(node, isDisplayMath(node))
	}

	switch node.Data {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		level := int(node.Data[1] - '0')
		text := strings.TrimSpace(c.renderInlineChildren(node))
		if text == "" {
			return ""
		}
		return strings.Repeat("#", level) + " " + escapeLeadingBlockMarker(text)
	case "p":
		return escapeLeadingBlockMarker(strings.TrimSpace(c.renderInlineChildren(node)))
	case "hr":
		return "---"
	case "br":
		return ""
	case "blockquote":
		inner := strings.TrimSpace(c.renderBlocks(node))
		if inner == "" {
			return ""
		}
		return prefixLines(inner, "> ")
	case "ul", "ol":
		return c.renderList(node)
	case "dl":
		return c.renderDefinitionList(node)
	case "pre":
		return c.renderCodeBlock(node)
	case "table":
		return c.renderTable(node)
	case "figure":
		return c.renderFigure(node)
	case "details":
		return c.renderDetails(node)
	default:
		return strings.TrimSpace(c.renderBlocks(node))
	}
}

// renderInlineElement renders a single inline-level element to Markdown.
func (c *Converter) renderInlineElement(node *html.Node) string {
	if isMathElement(node) {
		return c.renderMath(node, false)
	}

	switch node.Data {
	case "strong", "b":
		return wrapInline("**", c.renderInlineChildren(node))
	case "em", "i":
		return wrapInline("*", c.renderInlineChildren(node))
	case "del", "s", "strike":
		return wrapInline("~~", c.renderInlineChildren(node))
	case "code", "kbd", "samp", "var":
		return formatCodeSpan(nodes.TextContent(node))
	case "a":
		return c.renderLink(node)
	case "img":
		return c.renderImage(node)
	case "br":
		return "  \n"
	case "q":
		return "\"" + c.renderInlineChildren(node) + "\""
	default:
		if isBlockElement(node.Data) {
			return strings.TrimSpace(c.renderBlocks(node))
		}
		return c.renderInlineChildren(node)
	}
}

// renderList renders an ordered or unordered list with nested indentation.
func (c *Converter) renderList(list *html.Node) string {
	ordered := list.Data == "ol"
	nextNumber := 1
	if ordered {
		if start, err := strconv.Atoi(nodes.Attribute(list, "start")); err == nil {
			nextNumber = start
		}
	}

	items := make([]string, 0, 8)
	loose := false

	for child := list.FirstChild; child != nil; child = child.NextSibling {
		if !nodes.IsElement(child, "li") {
			continue
		}

		content := strings.TrimSpace(c.renderBlocks(child))
		if strings.Contains(content, "\n\n") {
			loose = true
		}

		marker := "-"
		if ordered {
			marker = fmt.Sprintf("%d.", nextNumber)
			nextNumber++
		}

		indent := strings.Repeat(" ", len(marker)+1)
		items = append(items, marker+" "+indentContinuation(content, indent))
	}

	if len(items) == 0 {
		return ""
	}

	separator := "\n"
	if loose {
		separator = "\n\n"
	}
	return strings.Join(items, separator)
}

// renderDefinitionList renders a <dl> as bold terms followed by definitions.
func (c *Converter) renderDefinitionList(list *html.Node) string {
	chunks := make([]string, 0, 8)
	for child := list.FirstChild; child != nil; child = child.NextSibling {
		switch {
		case nodes.IsElement(child, "dt"):
			text := strings.TrimSpace(c.renderInlineChildren(child))
			if text != "" {
				chunks = append(chunks, "**"+text+"**")
			}
		case nodes.IsElement(child, "dd"):
			text := strings.TrimSpace(c.renderBlocks(child))
			if text != "" {
				chunks = append(chunks, indentContinuation(": "+text, "  "))
			}
		}
	}
	return strings.Join(chunks, "\n")
}

// renderFigure renders a figure and an optional caption.
func (c *Converter) renderFigure(figure *html.Node) string {
	chunks := make([]string, 0, 4)
	for child := figure.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != elementNode {
			continue
		}
		if child.Data == "figcaption" {
			caption := strings.TrimSpace(c.renderInlineChildren(child))
			if caption != "" {
				chunks = append(chunks, "*"+caption+"*")
			}
			continue
		}
		if chunk := c.renderBlockElement(child); chunk != "" {
			chunks = append(chunks, chunk)
		}
	}
	return strings.Join(chunks, "\n\n")
}

// renderDetails renders <details>/<summary> as a bold summary and body.
func (c *Converter) renderDetails(details *html.Node) string {
	summary := ""
	chunks := make([]string, 0, 4)
	for child := details.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != elementNode {
			continue
		}
		if child.Data == "summary" {
			summary = strings.TrimSpace(c.renderInlineChildren(child))
			continue
		}
		if chunk := c.renderBlockElement(child); chunk != "" {
			chunks = append(chunks, chunk)
		}
	}
	if summary != "" {
		chunks = append([]string{"**" + summary + "**"}, chunks...)
	}
	return strings.Join(chunks, "\n\n")
}

// renderTable renders a data table as a GFM table and a layout table by
// unwrapping its cells in document order.
func (c *Converter) renderTable(table *html.Node) string {
	if isLayoutTable(table) {
		return c.renderLayoutTable(table)
	}
	return c.renderDataTable(table)
}

func isLayoutTable(table *html.Node) bool {
	if nodes.ContainsElement(table, "caption") ||
		nodes.ContainsElement(table, "th") ||
		nodes.ContainsElement(table, "thead") {
		return false
	}

	blockTags := map[string]struct{}{
		"p": {}, "pre": {}, "div": {}, "ul": {}, "ol": {}, "table": {},
		"blockquote": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {},
		"h6": {}, "section": {}, "article": {}, "figure": {}, "dl": {},
		"details": {},
	}

	for _, row := range collectTableRows(table) {
		for _, cell := range row {
			if nodes.CountElements(cell, blockTags) > 0 {
				return true
			}
		}
	}
	return false
}

func (c *Converter) renderLayoutTable(table *html.Node) string {
	chunks := make([]string, 0, 8)
	for _, row := range collectTableRows(table) {
		for _, cell := range row {
			if chunk := strings.TrimSpace(c.renderBlocks(cell)); chunk != "" {
				chunks = append(chunks, chunk)
			}
		}
	}
	return strings.Join(chunks, "\n\n")
}

func (c *Converter) renderDataTable(table *html.Node) string {
	rows := collectTableRows(table)
	if len(rows) == 0 {
		return ""
	}

	headerIndex := -1
	for index, row := range rows {
		for _, cell := range row {
			if cell.Data == "th" {
				headerIndex = index
				break
			}
		}
		if headerIndex >= 0 {
			break
		}
	}

	var headerRow []*html.Node
	bodyRows := rows
	if headerIndex >= 0 {
		headerRow = rows[headerIndex]
		bodyRows = append(append([][]*html.Node{}, rows[:headerIndex]...), rows[headerIndex+1:]...)
	} else {
		headerRow = rows[0]
		bodyRows = rows[1:]
	}

	columnCount := len(headerRow)
	for _, row := range bodyRows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	if columnCount == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(formatTableRow(c, headerRow, columnCount))
	builder.WriteString("\n")
	builder.WriteString(strings.Repeat("| --- ", columnCount))
	builder.WriteString("|")

	for _, row := range bodyRows {
		builder.WriteString("\n")
		builder.WriteString(formatTableRow(c, row, columnCount))
	}

	return builder.String()
}

func formatTableRow(converter *Converter, row []*html.Node, columnCount int) string {
	var builder strings.Builder
	for column := 0; column < columnCount; column++ {
		builder.WriteString("| ")
		if column < len(row) {
			cell := strings.TrimSpace(converter.renderInlineChildren(row[column]))
			cell = strings.ReplaceAll(cell, "\n", " ")
			cell = strings.Join(strings.Fields(cell), " ")
			builder.WriteString(cell)
		}
		builder.WriteByte(' ')
	}
	builder.WriteByte('|')
	return builder.String()
}

func collectTableRows(table *html.Node) [][]*html.Node {
	rows := make([][]*html.Node, 0, 8)
	collectRowsInto(table, &rows)
	return rows
}

func collectRowsInto(container *html.Node, rows *[][]*html.Node) {
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != elementNode {
			continue
		}
		switch child.Data {
		case "tr":
			*rows = append(*rows, cellsOfRow(child))
		case "table":
			continue
		default:
			collectRowsInto(child, rows)
		}
	}
}

func cellsOfRow(row *html.Node) []*html.Node {
	cells := make([]*html.Node, 0, 4)
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == elementNode && (child.Data == "td" || child.Data == "th") {
			cells = append(cells, child)
		}
	}
	return cells
}
