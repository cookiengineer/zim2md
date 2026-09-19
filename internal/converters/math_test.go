package converters

import (
	"strings"
	"testing"
)

func TestMathoidAnnotationBecomesMath(t *testing.T) {
	source := `<p>Let <span class="mwe-math-element">` +
		`<span class="mwe-math-mathml-inline mwe-math-mathml-a11y" style="display: none;">` +
		`<math xmlns="http://www.w3.org/1998/Math/MathML" alttext="{\displaystyle x^2}"><semantics><mrow><msup><mi>x</mi><mn>2</mn></msup></mrow>` +
		`<annotation encoding="application/x-tex">{\displaystyle x^2}</annotation></semantics></math></span>` +
		`<img class="mwe-math-fallback-image-inline" alt="{\displaystyle x^2}" src="https://upload.example/math.png"></span> be a square.</p>`

	markdown := convertHTML(t, source, Options{})

	if !strings.Contains(markdown, "$x^2$") {
		t.Fatalf("TeX math missing:\n%s", markdown)
	}
	if strings.Count(markdown, "$x^2$") != 1 {
		t.Fatalf("formula rendered more than once:\n%s", markdown)
	}
	if strings.Contains(markdown, "![") || strings.Contains(markdown, "<math") {
		t.Fatalf("formula leaked as image or raw HTML:\n%s", markdown)
	}
}

func TestPureMathMLBecomesMath(t *testing.T) {
	markdown := convertHTML(t, `<p><math class="mwe-math-element mwe-math-element-inline"><msup><mi>x</mi><mn>2</mn></msup></math></p>`, Options{})

	if !strings.Contains(markdown, "$x^{2}$") {
		t.Fatalf("MathML was not converted:\n%s", markdown)
	}
}

func TestFallbackImageAloneBecomesMath(t *testing.T) {
	markdown := convertHTML(t, `<p>Formula <img class="mwe-math-fallback-image-inline" alt="{\displaystyle a+b}" src="math.png"> here.</p>`, Options{})

	if !strings.Contains(markdown, "$a+b$") {
		t.Fatalf("fallback TeX missing:\n%s", markdown)
	}
	if strings.Contains(markdown, "![") {
		t.Fatalf("fallback image rendered as an image:\n%s", markdown)
	}
}

func TestDisplayMathBecomesOwnBlock(t *testing.T) {
	source := `<p>before</p><div>` +
		`<math class="mwe-math-element mwe-math-element-block"><mfrac><mi>a</mi><mi>b</mi></mfrac></math>` +
		`</div><p>after</p>`

	markdown := convertHTML(t, source, Options{})

	if !strings.Contains(markdown, `$$\frac{a}{b}$$`) {
		t.Fatalf("display fraction missing:\n%s", markdown)
	}
	if !strings.Contains(markdown, "before\n\n") || !strings.Contains(markdown, "\n\nafter") {
		t.Fatalf("formula was not separated as a block:\n%s", markdown)
	}
}
