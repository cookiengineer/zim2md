// Package mathml converts presentation MathML (as emitted by the MediaWiki
// Math extension and by MathJax) into LaTeX, and extracts the original TeX
// source when an <annotation encoding="application/x-tex"> or an alttext
// attribute is available.
package mathml

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

// ExtractTeX returns the original TeX source carried by a <math> element, if
// any. It prefers the Mathoid-style TeX annotation and falls back to the
// alttext attribute. The returned string is cleaned of the surrounding
// {\displaystyle ...} wrapper.
func ExtractTeX(mathNode *html.Node) (string, bool) {
	if mathNode == nil {
		return "", false
	}

	var annotation string
	nodes.Walk(mathNode, func(node *html.Node) bool {
		if node.Type != html.ElementNode || node.Data != "annotation" {
			return true
		}
		if !strings.EqualFold(nodes.Attribute(node, "encoding"), "application/x-tex") {
			return true
		}
		annotation = nodes.TextContent(node)
		return false
	})
	if text := CleanTeX(annotation); text != "" {
		return text, true
	}

	if text := CleanTeX(nodes.Attribute(mathNode, "alttext")); text != "" {
		return text, true
	}
	return "", false
}

// CleanTeX removes the {\displaystyle ...} / {\textstyle ...} wrapper that the
// MediaWiki Math extension adds around annotation and alttext values.
func CleanTeX(tex string) string {
	tex = strings.TrimSpace(tex)
	for _, prefix := range []string{`{\displaystyle`, `{\textstyle`} {
		if strings.HasPrefix(tex, prefix) && strings.HasSuffix(tex, "}") {
			tex = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(tex, prefix), "}"))
		}
	}
	tex = strings.TrimSpace(tex)
	for _, prefix := range []string{`\displaystyle`, `\textstyle`} {
		tex = strings.TrimSpace(strings.TrimPrefix(tex, prefix))
	}
	return strings.TrimSpace(tex)
}

// ToLaTeX converts a presentation MathML subtree into a LaTeX string. It is
// best-effort: unknown constructs recurse into their children.
func ToLaTeX(node *html.Node) string {
	var builder strings.Builder
	write(&builder, node)
	return strings.TrimSpace(builder.String())
}

func write(builder *strings.Builder, node *html.Node) {
	if node == nil {
		return
	}
	switch node.Type {
	case html.TextNode:
		builder.WriteString(node.Data)
		return
	case html.ElementNode:
	default:
		return
	}

	switch node.Data {
	case "math", "mrow", "mstyle", "mpadded", "mphantom", "merror", "maction", "mstack":
		writeChildren(builder, node)
	case "semantics":
		writeFirstElement(builder, node)
	case "annotation", "annotation-xml", "mprescripts", "none":
		return
	case "mi":
		writeIdentifier(builder, node)
	case "mn":
		builder.WriteString(escapeIdentifiers(nodes.TextContent(node)))
	case "mo":
		writeOperator(builder, node)
	case "mtext":
		writeText(builder, node)
	case "mspace":
		writeSpace(builder, node)
	case "mfrac":
		writeFraction(builder, node)
	case "msqrt":
		builder.WriteString(`\sqrt{`)
		writeChildren(builder, node)
		builder.WriteString("}")
	case "mroot":
		writeRoot(builder, node)
	case "msub":
		writeScripts(builder, node, true, false)
	case "msup":
		writeScripts(builder, node, false, true)
	case "msubsup":
		writeScripts(builder, node, true, true)
	case "munder":
		writeUnderOver(builder, node, false, false)
	case "mover":
		writeUnderOver(builder, node, true, false)
	case "munderover":
		writeUnderOver(builder, node, true, true)
	case "mtable":
		writeTable(builder, node)
	case "mfenced":
		writeFenced(builder, node)
	case "menclose":
		builder.WriteString(`\overline{`)
		writeChildren(builder, node)
		builder.WriteString("}")
	case "mmultiscripts":
		writeMultiscripts(builder, node)
	default:
		writeChildren(builder, node)
	}
}

func writeChildren(builder *strings.Builder, node *html.Node) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		write(builder, child)
	}
}

func writeFirstElement(builder *strings.Builder, node *html.Node) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			write(builder, child)
			return
		}
	}
}

func elementChildren(node *html.Node) []*html.Node {
	children := make([]*html.Node, 0, 4)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			children = append(children, child)
		}
	}
	return children
}

func writeIdentifier(builder *strings.Builder, node *html.Node) {
	text := nodes.TextContent(node)
	if text == "" {
		return
	}

	variant, hasVariant := mathVariantCommands[strings.ToLower(nodes.Attribute(node, "mathvariant"))]
	runes := []rune(text)
	if len(runes) == 1 {
		if hasVariant {
			builder.WriteString(variant)
			builder.WriteString("{")
			builder.WriteRune(runes[0])
			builder.WriteString("}")
			return
		}
		if command, ok := runeCommands[runes[0]]; ok {
			builder.WriteString(command)
			return
		}
		builder.WriteRune(runes[0])
		return
	}

	if command, ok := functionNames[text]; ok {
		builder.WriteString(command)
		return
	}
	if hasVariant {
		builder.WriteString(variant)
		builder.WriteString("{")
		builder.WriteString(escapeIdentifiers(text))
		builder.WriteString("}")
		return
	}

	builder.WriteString(`\mathrm{`)
	builder.WriteString(escapeIdentifiers(text))
	builder.WriteString("}")
}

func writeOperator(builder *strings.Builder, node *html.Node) {
	text := nodes.TextContent(node)
	if text == "" {
		return
	}
	if text == "\u2061" || text == "\u2063" {
		return
	}
	if text == "\u2062" {
		builder.WriteString(" ")
		return
	}

	command := text
	if mapped, ok := operatorCommands[text]; ok {
		command = mapped
	}

	class := classOrd
	if value, ok := texClass(nodes.Attribute(node, "data-mjx-texclass")); ok {
		class = value
	} else if value, ok := operatorClasses[text]; ok {
		class = value
	}
	writeSpacedOperator(builder, command, class)
}

func writeSpacedOperator(builder *strings.Builder, command string, class operatorClass) {
	switch class {
	case classBin, classRel:
		builder.WriteString(" ")
		builder.WriteString(command)
		builder.WriteString(" ")
	case classPunct:
		builder.WriteString(command)
		builder.WriteString(" ")
	default:
		builder.WriteString(command)
	}
}

func texClass(value string) (operatorClass, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "BIN":
		return classBin, true
	case "REL":
		return classRel, true
	case "OPEN":
		return classOpen, true
	case "CLOSE":
		return classClose, true
	case "PUNCT":
		return classPunct, true
	case "INNER":
		return classInner, true
	case "OP", "ORD":
		return classOrd, true
	}
	return classOrd, false
}

func writeText(builder *strings.Builder, node *html.Node) {
	text := nodes.TextContent(node)
	if text == "" {
		return
	}
	builder.WriteString(`\text{`)
	builder.WriteString(escapeText(text))
	builder.WriteString("}")
}

func writeSpace(builder *strings.Builder, node *html.Node) {
	width := strings.TrimSpace(nodes.Attribute(node, "width"))
	builder.WriteString(spaceForWidth(width))
}

func spaceForWidth(width string) string {
	if width == "" {
		return `\,`
	}
	if strings.HasSuffix(width, "em") {
		if value, err := strconv.ParseFloat(strings.TrimSuffix(width, "em"), 64); err == nil {
			switch {
			case value <= 0:
				return ""
			case value < 0.2:
				return `\,`
			case value < 0.31:
				return `\;`
			case value < 0.6:
				return `\ `
			case value < 1.5:
				return `\quad`
			default:
				return `\qquad`
			}
		}
	}
	return `\,`
}

func writeFraction(builder *strings.Builder, node *html.Node) {
	children := elementChildren(node)
	if len(children) < 2 {
		writeChildren(builder, node)
		return
	}
	builder.WriteString(`\frac{`)
	write(builder, children[0])
	builder.WriteString("}{")
	write(builder, children[1])
	builder.WriteString("}")
}

func writeRoot(builder *strings.Builder, node *html.Node) {
	children := elementChildren(node)
	if len(children) < 2 {
		writeChildren(builder, node)
		return
	}
	builder.WriteString(`\sqrt[`)
	write(builder, children[1])
	builder.WriteString("]{")
	write(builder, children[0])
	builder.WriteString("}")
}

func writeScripts(builder *strings.Builder, node *html.Node, hasSub, hasSup bool) {
	children := elementChildren(node)
	if len(children) < 2 {
		writeChildren(builder, node)
		return
	}

	builder.WriteString(wrapBase(renderToString(children[0])))
	if hasSub && len(children) > 1 {
		builder.WriteString("_{")
		write(builder, children[1])
		builder.WriteString("}")
	}
	if hasSup {
		index := 1
		if hasSub {
			index = 2
			if len(children) < 3 {
				return
			}
		}
		builder.WriteString("^{")
		write(builder, children[index])
		builder.WriteString("}")
	}
}

func writeUnderOver(builder *strings.Builder, node *html.Node, over, under bool) {
	children := elementChildren(node)
	if len(children) < 2 {
		writeChildren(builder, node)
		return
	}

	base := children[0]
	if over && !under {
		if command, ok := accentCommands[strings.TrimSpace(nodes.TextContent(children[1]))]; ok {
			builder.WriteString(command)
			builder.WriteString("{")
			write(builder, base)
			builder.WriteString("}")
			return
		}
	}

	builder.WriteString(wrapBase(renderToString(base)))
	if under && len(children) > 1 {
		builder.WriteString("_{")
		write(builder, children[1])
		builder.WriteString("}")
	}
	if over {
		index := 1
		if under {
			index = 2
			if len(children) < 3 {
				return
			}
		}
		builder.WriteString("^{")
		write(builder, children[index])
		builder.WriteString("}")
	}
}

func writeTable(builder *strings.Builder, node *html.Node) {
	environment := matrixEnvironment(node)
	builder.WriteString(`\begin{`)
	builder.WriteString(environment)
	builder.WriteString("}")

	for rowIndex, row := range elementChildren(node) {
		if row.Data != "mtr" {
			write(builder, row)
			continue
		}
		if rowIndex > 0 {
			builder.WriteString(` \\ `)
		}
		for cellIndex, cell := range elementChildren(row) {
			if cellIndex > 0 {
				builder.WriteString(` & `)
			}
			write(builder, cell)
		}
	}

	builder.WriteString(`\end{`)
	builder.WriteString(environment)
	builder.WriteString("}")
}

func matrixEnvironment(node *html.Node) string {
	parent := node.Parent
	if parent == nil || parent.Data != "mfenced" {
		return "matrix"
	}
	open := nodes.AttributeOr(parent, "open", "(")
	close := nodes.AttributeOr(parent, "close", ")")
	switch open + close {
	case "()":
		return "pmatrix"
	case "[]":
		return "bmatrix"
	case "{}":
		return "Bmatrix"
	case "||":
		return "vmatrix"
	}
	return "matrix"
}

func writeFenced(builder *strings.Builder, node *html.Node) {
	open := nodes.AttributeOr(node, "open", "(")
	close := nodes.AttributeOr(node, "close", ")")
	builder.WriteString(`\left` + delimiterFor(open) + " ")

	children := elementChildren(node)
	for index, child := range children {
		if index > 0 {
			builder.WriteString(", ")
		}
		write(builder, child)
	}

	builder.WriteString(` \right` + delimiterFor(close))
}

func delimiterFor(value string) string {
	switch value {
	case "":
		return "."
	case "‖":
		return `\|`
	default:
		return value
	}
}

func writeMultiscripts(builder *strings.Builder, node *html.Node) {
	children := elementChildren(node)
	if len(children) == 0 {
		return
	}
	write(builder, children[0])
	for index := 1; index < len(children); index++ {
		child := children[index]
		switch child.Data {
		case "mprescripts":
			continue
		case "none":
			continue
		default:
			builder.WriteString("_{")
			write(builder, child)
			builder.WriteString("}")
		}
	}
}

func renderToString(node *html.Node) string {
	var builder strings.Builder
	write(&builder, node)
	return builder.String()
}

func wrapBase(base string) string {
	if base == "" {
		return ""
	}
	if utf8.RuneCountInString(base) <= 1 {
		return base
	}
	if strings.HasPrefix(base, `\`) && !strings.ContainsAny(strings.TrimPrefix(base, `\`), "{} ") {
		return base
	}
	return "{" + base + "}"
}

func escapeText(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\textbackslash{}`,
		`{`, `\{`,
		`}`, `\}`,
		`$`, `\$`,
		`&`, `\&`,
		`#`, `\#`,
		`%`, `\%`,
		`_`, `\_`,
		`^`, `\textasciicircum{}`,
		`~`, `\textasciitilde{}`,
	)
	return replacer.Replace(value)
}

func escapeIdentifiers(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\backslash`,
		`{`, `\{`,
		`}`, `\}`,
		`$`, `\$`,
		`&`, `\&`,
		`#`, `\#`,
		`%`, `\%`,
		`_`, `\_`,
		`^`, `\^{}`,
	)
	return replacer.Replace(value)
}
