package mathml

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parseMath(t *testing.T, source string) *html.Node {
	t.Helper()

	document, err := html.Parse(strings.NewReader("<!doctype html><html><body>" + source + "</body></html>"))
	if err != nil {
		t.Fatalf("parsing MathML: %v", err)
	}

	var found *html.Node
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if found != nil {
			return
		}
		if node.Type == html.ElementNode && node.Data == "math" {
			found = node
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	if found == nil {
		t.Fatalf("no <math> element in source")
	}
	return found
}

func TestToLaTeX(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name:     "polynomial",
			source:   `<math><mi>P</mi><mo>=</mo><msub><mi>a</mi><mn>0</mn></msub><mo>+</mo><msub><mi>a</mi><mn>1</mn></msub><mi>X</mi></math>`,
			expected: `P = a_{0} + a_{1}X`,
		},
		{
			name:     "fraction",
			source:   `<math><mfrac><mi>a</mi><mi>b</mi></mfrac></math>`,
			expected: `\frac{a}{b}`,
		},
		{
			name:     "square root",
			source:   `<math><msqrt><mi>x</mi></msqrt></math>`,
			expected: `\sqrt{x}`,
		},
		{
			name:     "nth root",
			source:   `<math><mroot><mi>x</mi><mn>3</mn></mroot></math>`,
			expected: `\sqrt[3]{x}`,
		},
		{
			name:     "sum with limits",
			source:   `<math><munderover><mo>∑</mo><mrow><mi>i</mi><mo>=</mo><mn>1</mn></mrow><mi>n</mi></munderover></math>`,
			expected: `\sum_{i = 1}^{n}`,
		},
		{
			name:     "accent",
			source:   `<math><mover><mi>x</mi><mo>→</mo></mover></math>`,
			expected: `\vec{x}`,
		},
		{
			name:     "greek and relation",
			source:   `<math><mi>α</mi><mo>∈</mo><mi>ℝ</mi></math>`,
			expected: `\alpha \in \mathbb{R}`,
		},
		{
			name:     "text",
			source:   `<math><mtext>for all</mtext></math>`,
			expected: `\text{for all}`,
		},
		{
			name:     "function name",
			source:   `<math><mi mathvariant="normal">deg</mi></math>`,
			expected: `\deg`,
		},
		{
			name:     "semantics uses presentation tree",
			source:   `<math><semantics><mrow><msup><mi>x</mi><mn>2</mn></msup></mrow><annotation encoding="application/x-tex">x^2</annotation></semantics></math>`,
			expected: `x^{2}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ToLaTeX(parseMath(t, test.source))
			if got != test.expected {
				t.Errorf("ToLaTeX() = %q, want %q", got, test.expected)
			}
		})
	}
}

func TestCleanTeXStripsDisplayStyle(t *testing.T) {
	tests := map[string]string{
		`{\displaystyle x}`:                        `x`,
		`{\displaystyle \textstyle ax^{2}+bx+c=0}`: `ax^{2}+bx+c=0`,
		`x^2`:                             `x^2`,
		`  {\displaystyle \frac{a}{b}}  `: `\frac{a}{b}`,
	}
	for input, expected := range tests {
		if got := CleanTeX(input); got != expected {
			t.Errorf("CleanTeX(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestExtractTeXPrefersAnnotation(t *testing.T) {
	node := parseMath(t, `<math alttext="{\displaystyle y}"><semantics><mrow><mi>x</mi></mrow><annotation encoding="application/x-tex">{\displaystyle x}</annotation></semantics></math>`)
	got, ok := ExtractTeX(node)
	if !ok {
		t.Fatal("ExtractTeX reported no TeX")
	}
	if got != "x" {
		t.Errorf("ExtractTeX() = %q, want %q", got, "x")
	}
}

func TestExtractTeXFallsBackToAltText(t *testing.T) {
	node := parseMath(t, `<math alttext="{\displaystyle y}"><semantics><mrow><mi>y</mi></mrow></semantics></math>`)
	got, ok := ExtractTeX(node)
	if !ok {
		t.Fatal("ExtractTeX reported no TeX")
	}
	if got != "y" {
		t.Errorf("ExtractTeX() = %q, want %q", got, "y")
	}
}

func TestExtractTeXMissing(t *testing.T) {
	node := parseMath(t, `<math><mi>x</mi></math>`)
	if _, ok := ExtractTeX(node); ok {
		t.Fatal("ExtractTeX unexpectedly reported TeX")
	}
}
