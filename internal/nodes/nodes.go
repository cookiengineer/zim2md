// Package nodes provides small, allocation-conscious helpers for working with
// golang.org/x/net/html node trees.
package nodes

import (
	"strings"

	"golang.org/x/net/html"
)

// Attribute returns the value of an attribute, or "" when it is absent.
func Attribute(node *html.Node, name string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return ""
}

// AttributeOr returns the attribute value or fallback when absent.
func AttributeOr(node *html.Node, name, fallback string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return attribute.Val
		}
	}
	return fallback
}

// HasAttribute reports whether the node carries the named attribute.
func HasAttribute(node *html.Node, name string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key == name {
			return true
		}
	}
	return false
}

// ClassList returns the whitespace separated classes of an element.
func ClassList(node *html.Node) []string {
	class := Attribute(node, "class")
	if class == "" {
		return nil
	}
	return strings.Fields(class)
}

// HasClass reports whether the element carries the given class token.
func HasClass(node *html.Node, class string) bool {
	for _, token := range ClassList(node) {
		if token == class {
			return true
		}
	}
	return false
}

// ClassAndID returns class and id together, lower-cased, for token matching.
func ClassAndID(node *html.Node) string {
	class := strings.ToLower(Attribute(node, "class"))
	id := strings.ToLower(Attribute(node, "id"))
	switch {
	case class == "":
		return id
	case id == "":
		return class
	default:
		return class + " " + id
	}
}

// IsElement reports whether the node is an element with the given tag name.
func IsElement(node *html.Node, tag string) bool {
	return node != nil && node.Type == html.ElementNode && node.Data == tag
}

// TextContent returns the concatenated text of all descendant text nodes.
func TextContent(node *html.Node) string {
	var builder strings.Builder
	collectText(node, &builder)
	return builder.String()
}

// TextContentTrimmed returns TextContent with surrounding space removed.
func TextContentTrimmed(node *html.Node) string {
	return strings.TrimSpace(TextContent(node))
}

// ElementChildren returns the direct element children of a node.
func ElementChildren(node *html.Node) []*html.Node {
	children := make([]*html.Node, 0, 8)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			children = append(children, child)
		}
	}
	return children
}

// FirstElementChild returns the first direct element child, or nil.
func FirstElementChild(node *html.Node) *html.Node {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			return child
		}
	}
	return nil
}

// Remove detaches a node from its parent.
func Remove(node *html.Node) {
	if node != nil && node.Parent != nil {
		node.Parent.RemoveChild(node)
	}
}

// Unwrap replaces a node with its children, keeping document order.
func Unwrap(node *html.Node) {
	parent := node.Parent
	if parent == nil {
		return
	}
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		node.RemoveChild(child)
		parent.InsertBefore(child, node)
		child = next
	}
	parent.RemoveChild(node)
}

// Walk visits every descendant (including the node itself) in document order.
// Returning false from the visitor stops the walk.
func Walk(node *html.Node, visit func(*html.Node) bool) {
	if node == nil {
		return
	}
	if !visit(node) {
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		Walk(child, visit)
	}
}

// CountElements counts descendants (including the node) whose tag is in tags.
func CountElements(node *html.Node, tags map[string]struct{}) int {
	count := 0
	Walk(node, func(current *html.Node) bool {
		if current.Type == html.ElementNode {
			if _, ok := tags[current.Data]; ok {
				count++
			}
		}
		return true
	})
	return count
}

// ContainsElement reports whether node or any descendant has the given tag.
func ContainsElement(node *html.Node, tag string) bool {
	found := false
	Walk(node, func(current *html.Node) bool {
		if IsElement(current, tag) {
			found = true
			return false
		}
		return !found
	})
	return found
}

func collectText(node *html.Node, builder *strings.Builder) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case html.TextNode:
			builder.WriteString(child.Data)
		case html.ElementNode:
			if child.Data == "br" {
				builder.WriteByte('\n')
				continue
			}
			collectText(child, builder)
		}
	}
}
