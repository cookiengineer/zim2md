// Package parsers extracts the readable main content from an HTML document.
// It removes navigation, sidebars, scripts and other non-content structure and
// returns a sanitized DOM subtree that the converters package turns into
// Markdown.
package parsers

import (
	"strings"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

var unwantedElements = map[string]struct{}{
	"applet": {}, "area": {}, "audio": {}, "base": {}, "blink": {}, "button": {},
	"canvas": {}, "dialog": {}, "embed": {}, "fieldset": {}, "form": {}, "frame": {},
	"frameset": {}, "head": {}, "iframe": {}, "input": {}, "label": {}, "legend": {},
	"link": {}, "map": {}, "marquee": {}, "meta": {}, "nav": {}, "noscript": {},
	"object": {},
	"option": {}, "param": {}, "portal": {}, "script": {}, "select": {}, "source": {},
	"style": {}, "svg": {}, "template": {}, "textarea": {}, "track": {}, "video": {},
}

var unwantedRoles = map[string]struct{}{
	"alert": {}, "alertdialog": {}, "banner": {}, "complementary": {},
	"contentinfo": {}, "dialog": {}, "menu": {}, "menubar": {}, "navigation": {},
	"search": {}, "tablist": {},
}

var negativeClassTokens = map[string]struct{}{
	"ad": {}, "ads": {}, "advert": {}, "advertisement": {}, "banner": {},
	"breadcrumb": {}, "catlinks": {}, "comment": {}, "comments": {}, "cookie": {},
	"consent": {}, "disqus": {}, "editsection": {}, "footer": {}, "header": {},
	"masthead": {}, "menu": {}, "modal": {}, "nav": {}, "navbar": {}, "navigation": {},
	"navbox": {}, "overlay": {}, "pager": {}, "pagination": {}, "popup": {},
	"portal": {}, "printfooter": {}, "promo": {}, "recommended": {}, "related": {},
	"search": {}, "share": {}, "sharing": {}, "sidebar": {}, "sitesub": {},
	"skip": {}, "social": {}, "sponsor": {}, "toc": {}, "toolbar": {}, "widget": {},
}

var negativeClassSubstrings = []string{
	"sidebar", "navigation", "navbox", "breadcrumb", "pagination",
	"mw-editsection", "table-of-contents", "related-articles", "printfooter",
	"catlinks", "sitesub", "mw-jump", "vector-toc", "vector-menu",
	"share-buttons", "social-links", "cookie-banner", "advert",
}

// sanitizeDocument removes non-content structure in place.
func sanitizeDocument(root *html.Node) {
	var toRemove []*html.Node
	preserveMath := containsMath(root)

	nodes.Walk(root, func(node *html.Node) bool {
		switch node.Type {
		case html.CommentNode, html.DoctypeNode:
			if node.Type == html.CommentNode {
				toRemove = append(toRemove, node)
			}
			return true
		case html.ElementNode:
			if isUnwantedElement(node, preserveMath) {
				toRemove = append(toRemove, node)
			}
			return true
		}
		return true
	})

	for _, node := range toRemove {
		nodes.Remove(node)
	}
}

func isUnwantedElement(node *html.Node, preserveMath bool) bool {
	// Never drop the structural roots, regardless of their class list.
	if node.Data == "html" || node.Data == "body" {
		return false
	}

	// Math markup is often hidden behind display:none (the accessibility
	// variant of the Mathoid output), so it must survive the structural cleanup.
	keepForMath := func() bool {
		return preserveMath && containsMath(node)
	}

	if _, unwanted := unwantedElements[node.Data]; unwanted {
		return !keepForMath()
	}

	if nodes.HasAttribute(node, "hidden") {
		return !keepForMath()
	}
	if strings.EqualFold(nodes.Attribute(node, "aria-hidden"), "true") {
		return !keepForMath()
	}

	if role := strings.ToLower(strings.TrimSpace(nodes.Attribute(node, "role"))); role != "" {
		if _, unwanted := unwantedRoles[role]; unwanted {
			return !keepForMath()
		}
	}

	if style := strings.ToLower(nodes.Attribute(node, "style")); style != "" {
		if strings.Contains(style, "display:none") || strings.Contains(style, "display: none") ||
			strings.Contains(style, "visibility:hidden") || strings.Contains(style, "visibility: hidden") {
			return !keepForMath()
		}
	}

	if hasNegativeClassToken(node) {
		return !keepForMath()
	}
	return false
}

// containsMath reports whether the node holds a <math> subtree.
func containsMath(node *html.Node) bool {
	if node.Type == html.ElementNode && node.Data == "math" {
		return true
	}
	found := false
	nodes.Walk(node, func(current *html.Node) bool {
		if nodes.IsElement(current, "math") {
			found = true
			return false
		}
		return !found
	})
	return found
}

func hasNegativeClassToken(node *html.Node) bool {
	value := nodes.ClassAndID(node)
	if value == "" {
		return false
	}

	for _, token := range strings.Fields(value) {
		if _, negative := negativeClassTokens[token]; negative {
			return true
		}
		for _, fragment := range negativeClassSubstrings {
			if strings.Contains(token, fragment) {
				return true
			}
		}
	}
	return false
}

// removeEmptyElements drops structural wrappers that carry no text or media.
func removeEmptyElements(root *html.Node) {
	removable := map[string]struct{}{
		"article": {}, "blockquote": {}, "div": {}, "li": {}, "ol": {},
		"p": {}, "section": {}, "span": {}, "ul": {},
	}
	preserveMath := containsMath(root)

	for {
		var empty []*html.Node
		nodes.Walk(root, func(node *html.Node) bool {
			if node.Type != html.ElementNode {
				return true
			}
			if _, ok := removable[node.Data]; !ok {
				return true
			}
			if strings.TrimSpace(nodes.TextContent(node)) != "" {
				return true
			}
			if nodes.ContainsElement(node, "img") ||
				nodes.ContainsElement(node, "audio") ||
				nodes.ContainsElement(node, "video") ||
				nodes.ContainsElement(node, "canvas") ||
				(preserveMath && nodes.ContainsElement(node, "math")) ||
				nodes.ContainsElement(node, "hr") ||
				nodes.ContainsElement(node, "br") {
				return true
			}
			empty = append(empty, node)
			return true
		})
		if len(empty) == 0 {
			return
		}
		for _, node := range empty {
			nodes.Remove(node)
		}
	}
}
