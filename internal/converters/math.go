package converters

import (
	"strings"

	"github.com/cookiengineer/zim2md/internal/mathml"
	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

// isMathElement reports whether a node is a MathML root or a MediaWiki math
// wrapper that contains one.
func isMathElement(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	if node.Data == "math" {
		return true
	}
	if node.Data != "span" {
		return false
	}
	for _, class := range nodes.ClassList(node) {
		lower := strings.ToLower(class)
		if lower == "mwe-math-element" || strings.HasPrefix(lower, "mwe-math-mathml") {
			return true
		}
	}
	return false
}

// isMathFallbackImage reports whether an <img> is the rasterized fallback of a
// formula rather than an ordinary illustration.
func isMathFallbackImage(node *html.Node) bool {
	if !nodes.IsElement(node, "img") {
		return false
	}
	for _, class := range nodes.ClassList(node) {
		lower := strings.ToLower(class)
		if strings.HasPrefix(lower, "mwe-math-fallback") || lower == "tex" {
			return true
		}
	}
	return false
}

// isDisplayMath reports whether a math node is rendered as a display formula.
func isDisplayMath(node *html.Node) bool {
	display := false
	nodes.Walk(node, func(current *html.Node) bool {
		if current.Type != html.ElementNode {
			return true
		}
		if current.Data == "math" && strings.EqualFold(nodes.Attribute(current, "display"), "block") {
			display = true
			return false
		}
		for _, class := range nodes.ClassList(current) {
			lower := strings.ToLower(class)
			if strings.Contains(lower, "-block") || strings.Contains(lower, "-display") {
				display = true
				return false
			}
		}
		return !display
	})
	return display
}

// renderMath renders a MathML element, its MediaWiki wrapper or a fallback
// image as standard Markdown math ($...$ inline, $$...$$ display) carrying the
// exact TeX source.
func (c *Converter) renderMath(node *html.Node, display bool) string {
	tex := strings.TrimSpace(c.extractMathTeX(node))
	if tex == "" {
		return ""
	}
	tex = collapseWhitespace(tex)
	if display {
		return "$$" + tex + "$$"
	}
	return "$" + tex + "$"
}

func (c *Converter) extractMathTeX(node *html.Node) string {
	if isMathFallbackImage(node) {
		return mathml.CleanTeX(nodes.Attribute(node, "alt"))
	}

	mathNode := findMathNode(node)
	if mathNode == nil {
		if image := findMathFallbackImage(node); image != nil {
			return mathml.CleanTeX(nodes.Attribute(image, "alt"))
		}
		return ""
	}

	if tex, ok := mathml.ExtractTeX(mathNode); ok {
		return tex
	}
	if image := findMathFallbackImage(node); image != nil {
		if alt := mathml.CleanTeX(nodes.Attribute(image, "alt")); alt != "" {
			return alt
		}
	}
	return mathml.ToLaTeX(mathNode)
}

func findMathNode(node *html.Node) *html.Node {
	if nodes.IsElement(node, "math") {
		return node
	}
	var found *html.Node
	nodes.Walk(node, func(current *html.Node) bool {
		if nodes.IsElement(current, "math") {
			found = current
			return false
		}
		return true
	})
	return found
}

func findMathFallbackImage(node *html.Node) *html.Node {
	var found *html.Node
	nodes.Walk(node, func(current *html.Node) bool {
		if isMathFallbackImage(current) {
			found = current
			return false
		}
		return true
	})
	return found
}
