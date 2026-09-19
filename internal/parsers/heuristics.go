package parsers

import (
	"strings"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

var positiveClassTokens = []string{
	"article", "body-content", "chapter", "content", "documentation", "docs",
	"entry-content", "example", "main", "markdown", "mw-content-text",
	"mw-parser-output", "page", "post", "prose", "readme", "section", "text",
}

var paragraphTags = map[string]struct{}{
	"p": {}, "pre": {}, "td": {}, "li": {}, "blockquote": {},
}

type candidateScores map[*html.Node]float64

// selectMainContent scores content containers and returns the best candidate.
func selectMainContent(body *html.Node) (*html.Node, candidateScores) {
	scores := candidateScores{}

	nodes.Walk(body, func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return true
		}
		if _, ok := paragraphTags[node.Data]; !ok {
			return true
		}

		text := nodes.TextContent(node)
		trimmed := strings.TrimSpace(text)
		if len(trimmed) < 25 {
			return true
		}

		base := 1.0
		base += float64(minInt(3, strings.Count(text, ",")+1))
		base += float64(minInt(3, len(trimmed)/100))

		if parent := node.Parent; parent != nil && parent.Type == html.ElementNode {
			scores[parent] += base
			if grandparent := parent.Parent; grandparent != nil && grandparent.Type == html.ElementNode {
				scores[grandparent] += base / 2
			}
		}
		return true
	})

	nodes.Walk(body, func(node *html.Node) bool {
		if node.Type != html.ElementNode {
			return true
		}
		if weight := classWeight(node); weight > 0 {
			scores[node] += weight
		}
		return true
	})

	for node, value := range scores {
		final := value * (1.0 - linkDensity(node))
		if final < 0 {
			final = 0
		}
		scores[node] = final
	}

	return scores.best()
}

// expandToSiblings keeps content siblings of the candidate when the parent is a
// plausible content wrapper.
func expandToSiblings(candidate *html.Node, scores candidateScores) *html.Node {
	if candidate == nil {
		return nil
	}

	parent := candidate.Parent
	if parent == nil || parent.Type != html.ElementNode {
		return candidate
	}
	if parent.Data == "body" || parent.Data == "html" {
		return candidate
	}
	if parent.Data != "main" && parent.Data != "article" && classWeight(parent) <= 0 {
		return candidate
	}

	topScore := scores[candidate]
	threshold := topScore * 0.3
	if threshold < 20 {
		threshold = 20
	}

	qualifies := func(node *html.Node) bool {
		if node == candidate {
			return true
		}
		if scores[node] >= threshold {
			return true
		}
		return classWeight(node) > 0 && len(strings.TrimSpace(nodes.TextContent(node))) >= 200
	}

	qualifying := 0
	for child := parent.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		if qualifies(child) {
			qualifying++
		}
	}
	if qualifying <= 1 {
		return candidate
	}

	for child := parent.FirstChild; child != nil; {
		next := child.NextSibling
		if child.Type == html.ElementNode && !qualifies(child) {
			nodes.Remove(child)
		}
		child = next
	}
	return parent
}

func classWeight(node *html.Node) float64 {
	value := nodes.ClassAndID(node)
	if value == "" {
		return 0
	}

	weight := 0.0
	for _, token := range strings.Fields(value) {
		for _, positive := range positiveClassTokens {
			if strings.Contains(token, positive) {
				weight += 20
			}
		}
		if _, negative := negativeClassTokens[token]; negative {
			weight -= 25
		}
		for _, fragment := range negativeClassSubstrings {
			if strings.Contains(token, fragment) {
				weight -= 25
			}
		}
	}
	return weight
}

func linkDensity(node *html.Node) float64 {
	total := len(nodes.TextContent(node))
	if total == 0 {
		return 1
	}

	linkText := 0
	nodes.Walk(node, func(current *html.Node) bool {
		if current.Type == html.ElementNode && current.Data == "a" {
			linkText += len(nodes.TextContent(current))
			return false
		}
		return true
	})

	density := float64(linkText) / float64(total)
	if density > 1 {
		density = 1
	}
	return density
}

func (s candidateScores) best() (*html.Node, candidateScores) {
	var best *html.Node
	bestScore := 0.0
	for node, score := range s {
		if score > bestScore {
			bestScore = score
			best = node
		}
	}
	return best, s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
