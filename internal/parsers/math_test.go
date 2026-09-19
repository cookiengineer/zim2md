package parsers

import (
	"strings"
	"testing"

	"github.com/cookiengineer/zim2md/internal/nodes"
	"golang.org/x/net/html"
)

func TestHiddenMathAnnotationSurvivesSanitizer(t *testing.T) {
	source := `<html><body><p>` + strings.Repeat("meaningful mathematical content ", 20) + `</p>` +
		`<span class="mwe-math-element">` +
		`<span class="mwe-math-mathml-inline mwe-math-mathml-a11y" style="display: none;" aria-hidden="true">` +
		`<math xmlns="http://www.w3.org/1998/Math/MathML" alttext="{\displaystyle x}">` +
		`<semantics><mrow><mi>x</mi></mrow>` +
		`<annotation encoding="application/x-tex">{\displaystyle x}</annotation></semantics></math>` +
		`</span></span></body></html>`

	document, err := ParseHTMLDocument([]byte(source), "text/html")
	if err != nil {
		t.Fatalf("ParseHTMLDocument: %v", err)
	}

	if !nodes.ContainsElement(document.Root, "math") {
		t.Fatalf("hidden <math> subtree was stripped")
	}

	foundAnnotation := false
	nodes.Walk(document.Root, func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "annotation" {
			foundAnnotation = true
			return false
		}
		return true
	})
	if !foundAnnotation {
		t.Fatalf("TeX annotation was stripped")
	}
}
